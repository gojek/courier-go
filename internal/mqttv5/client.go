// Package mqttv5 holds the MQTT v5 client used by courier when MQTTv5 is selected.
//
// Client implements the gojek/paho.mqtt.golang Client interface on top of eclipse/paho.golang,
// so the pool, resolver, telemetry and middleware code in courier works unchanged for both protocol versions.
package mqttv5

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/eclipse/paho.golang/paho/log"

	mqtt "github.com/gojek/paho.mqtt.golang"
)

const (
	defaultOpTimeout = 30 * time.Second
	// defaultMaxReconnectInterval matches the v3 client's default.
	defaultMaxReconnectInterval = 10 * time.Minute
	// MQTT v5 reason codes at or above this value are failures.
	reasonFailure = 0x80
	// defaultConnectRetryInterval matches the v3 client's default.
	defaultConnectRetryInterval = 30 * time.Second
	// stableConnection is how long a connection must hold for the next loss to count as a fresh one.
	stableConnection = 3 * time.Second
)

// ErrNotStarted is returned when an operation is attempted before Connect.
var ErrNotStarted = errors.New("courier: mqtt v5 client not connected")

// nowFunc is swapped in tests.
var nowFunc = time.Now

type subscription struct {
	qos byte
	cb  mqtt.MessageHandler
	// from AddRoute, which registers a handler without subscribing
	noSub bool
}

// Client is an MQTT v5 implementation of mqtt.Client.
type Client struct {
	// never connected, only serves OptionsReader() so client metadata keeps working.
	mqtt.Client

	opts *mqtt.ClientOptions
	v5   Options
	errs log.Logger

	bmu         sync.Mutex
	lastAttempt time.Time
	lastDelay   time.Duration

	connectMu sync.Mutex

	mu           sync.Mutex
	cm           *autopaho.ConnectionManager
	cancel       context.CancelFunc
	connectToken *token
	subs         map[string]subscription

	up           atomic.Bool
	closed       atomic.Bool
	reconnecting atomic.Bool
}

// Options are the settings that exist only in MQTT v5.
type Options struct {
	// CleanStart clears any existing session on the first connection.
	CleanStart bool
	// SessionExpiryInterval is how long the broker keeps the session after disconnect, 0 ends it on disconnect.
	SessionExpiryInterval time.Duration
}

// New creates an MQTT v5 client from v3 style options plus the v5 only Options.
// KeepAlive, ConnectTimeout and ConnectRetry(Interval) are read from opts, CleanSession is not used.
func New(opts *mqtt.ClientOptions, v5 Options) *Client {
	_, errs := pahoLoggers(opts)

	return &Client{
		Client: mqtt.NewClient(opts),
		opts:   opts,
		v5:     v5,
		errs:   errs,
		subs:   map[string]subscription{},
	}
}

// IsConnected reports whether the connection to the broker is up.
func (v *Client) IsConnected() bool { return v.up.Load() }

// IsConnectionOpen reports whether the connection to the broker is up.
func (v *Client) IsConnectionOpen() bool { return v.up.Load() }

// Connect starts connecting, the token completes on first CONNACK.
// Calling it again before Disconnect returns the token of the attempt already running.
func (v *Client) Connect() mqtt.Token {
	v.connectMu.Lock()
	defer v.connectMu.Unlock()

	v.mu.Lock()
	running, t := v.cm != nil && !v.closed.Load(), v.connectToken
	v.mu.Unlock()

	// a second manager would keep retrying under the same client ID and take over the first one's session
	if running {
		return t
	}

	t = newToken()

	v.closed.Store(false)
	v.reconnecting.Store(false)

	ctx, cancel := context.WithCancel(context.Background())

	cm, err := autopaho.NewConnection(ctx, v.config(t, cancel))
	if err != nil {
		cancel()
		t.complete(err)

		return t
	}

	v.mu.Lock()
	v.cm, v.cancel, v.connectToken = cm, cancel, t
	v.mu.Unlock()

	return t
}

func (v *Client) config(connectToken *token, cancel context.CancelFunc) autopaho.ClientConfig {
	o := v.opts

	debug, errs := pahoLoggers(o)

	cfg := autopaho.ClientConfig{
		Debug:                         debug,
		Errors:                        errs,
		PahoDebug:                     debug,
		PahoErrors:                    errs,
		ServerUrls:                    o.Servers,
		TlsCfg:                        o.TLSConfig,
		KeepAlive:                     uint16(o.KeepAlive),
		CleanStartOnInitialConnection: v.v5.CleanStart,
		ConnectTimeout:                o.ConnectTimeout,
		ReconnectBackoff:              v.reconnectBackoff,
		ConnectPacketBuilder:          v.credentials,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, ack *paho.Connack) {
			v.up.Store(true)
			v.reconnecting.Store(false)
			connectToken.complete(nil)

			if !ack.SessionPresent {
				go v.resubscribe(cm)
			}

			if o.OnConnect != nil {
				go o.OnConnect(v)
			}
		},
		OnConnectError: func(err error) {
			if v.reconnecting.Load() {
				// v3 calls OnReconnecting before every retry, autopaho only reports after each failed attempt.
				if o.AutoReconnect && o.OnReconnecting != nil {
					go o.OnReconnecting(v, o)
				}

				return
			}

			if !o.ConnectRetry {
				connectToken.complete(err)
				cancel()
			}
		},
		ClientConfig: paho.ClientConfig{
			ClientID:          o.ClientID,
			PacketTimeout:     o.AckTimeout,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){v.route},
			OnClientError:     func(err error) { v.connectionLost(err, cancel) },
			OnServerDisconnect: func(d *paho.Disconnect) {
				v.connectionLost(fmt.Errorf("courier: server disconnected, reason code 0x%x", d.ReasonCode), cancel)
			},
		},
	}

	if secs := v.v5.SessionExpiryInterval / time.Second; secs > math.MaxUint32 {
		cfg.SessionExpiryInterval = math.MaxUint32
	} else {
		cfg.SessionExpiryInterval = uint32(secs)
	}

	return cfg
}

// reconnectBackoff is the delay before a connection attempt. The v3 client treats the two cases
// differently: a fixed ConnectRetryInterval until it first connects, and a doubling delay after a loss.
func (v *Client) reconnectBackoff(attempt int) time.Duration {
	if v.reconnecting.Load() {
		return v.lossBackoff()
	}

	if attempt == 0 {
		return 0
	}

	if d := v.opts.ConnectRetryInterval; d > 0 {
		return d
	}

	return defaultConnectRetryInterval
}

// lossBackoff mirrors the v3 backoff controller: a connection that stayed up starts over with no
// delay, otherwise the delay starts at 1s and doubles up to MaxReconnectInterval.
// autopaho resets its own attempt counter on every CONNACK, so the count has to be kept here.
func (v *Client) lossBackoff() time.Duration {
	maxDelay := v.opts.MaxReconnectInterval
	if maxDelay <= 0 {
		maxDelay = defaultMaxReconnectInterval
	}

	v.bmu.Lock()
	defer v.bmu.Unlock()

	now := nowFunc()
	if now.Sub(v.lastAttempt) > 2*stableConnection+v.lastDelay {
		v.lastAttempt, v.lastDelay = now, 0

		return 0
	}

	v.lastAttempt = now

	switch {
	case v.lastDelay <= 0:
		v.lastDelay = time.Second
	case v.lastDelay*2 >= maxDelay:
		v.lastDelay = maxDelay
	default:
		v.lastDelay *= 2
	}

	return v.lastDelay
}

// credentials is called for every connection attempt so refreshed credentials are picked up.
func (v *Client) credentials(c *paho.Connect, _ *url.URL) (*paho.Connect, error) {
	username, password := v.opts.Username, v.opts.Password
	if v.opts.CredentialsProvider != nil {
		username, password = v.opts.CredentialsProvider()
	}

	if username != "" {
		c.UsernameFlag, c.Username = true, username
	}

	if password != "" {
		c.PasswordFlag, c.Password = true, []byte(password)
	}

	return c, nil
}

// connectionLost can be called several times for one drop, only the first after a successful connect is acted on.
func (v *Client) connectionLost(err error, cancel context.CancelFunc) {
	if !v.up.Swap(false) || v.closed.Load() {
		return
	}

	if !v.opts.AutoReconnect {
		cancel() // autopaho always reconnects unless cancelled
	}

	v.reconnecting.Store(true)

	go func() {
		if v.opts.OnConnectionLost != nil {
			v.opts.OnConnectionLost(v, err)
		}

		if v.opts.AutoReconnect && v.opts.OnReconnecting != nil {
			v.opts.OnReconnecting(v, v.opts)
		}
	}()
}

// resubscribe restores subscriptions the broker did not retain across the reconnect.
func (v *Client) resubscribe(cm *autopaho.ConnectionManager) {
	v.mu.Lock()
	subs := make([]paho.SubscribeOptions, 0, len(v.subs))

	for topic, s := range v.subs {
		if s.noSub {
			continue
		}

		subs = append(subs, paho.SubscribeOptions{Topic: topic, QoS: s.qos})
	}
	v.mu.Unlock()

	if len(subs) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), v.opTimeout())
	defer cancel()

	resp, err := cm.Subscribe(ctx, &paho.Subscribe{Subscriptions: subs})

	// Nothing can return this failure to a caller, and the client stays connected but deaf.
	logResubscribeFailures(v.errs, subs, resp, err)
}

// logResubscribeFailures reports a resubscribe the broker refused, naming each topic it refused.
func logResubscribeFailures(errs log.Logger, subs []paho.SubscribeOptions, resp *paho.Suback, err error) {
	if err != nil {
		errs.Printf("courier: resubscribing to %d topic(s) failed, they receive nothing: %v", len(subs), err)

		return
	}

	if resp == nil {
		return
	}

	for i, r := range resp.Reasons {
		if r >= reasonFailure && i < len(subs) {
			errs.Printf("courier: resubscribing to %q rejected, reason code 0x%x, it receives nothing",
				subs[i].Topic, r)
		}
	}
}

// Disconnect ends the connection, quiesce is the time in milliseconds allowed for a clean shutdown.
func (v *Client) Disconnect(quiesce uint) {
	v.closed.Store(true)

	v.mu.Lock()
	cm, cancel := v.cm, v.cancel
	v.mu.Unlock()

	if cm == nil {
		return
	}

	ctx, stop := context.WithTimeout(context.Background(), time.Duration(quiesce)*time.Millisecond)
	defer stop()

	_ = cm.Disconnect(ctx)

	cancel()
	v.up.Store(false)
}

// Publish publishes a message.
func (v *Client) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	b, err := toBytes(payload)
	if err != nil {
		return completed(err)
	}

	p := &paho.Publish{Topic: topic, QoS: qos, Retain: retained, Payload: b}

	return v.run(context.Background(), func(ctx context.Context, cm *autopaho.ConnectionManager) error {
		resp, err := cm.Publish(ctx, p)

		// The v3 client stores a publish made while it is reconnecting and sends it on the new
		// connection, so wait for the reconnect here rather than failing straight away.
		if errors.Is(err, autopaho.ConnectionDownError) && v.opts.AutoReconnect {
			if awaitErr := cm.AwaitConnection(ctx); awaitErr != nil {
				return err
			}

			resp, err = cm.Publish(ctx, p)
		}

		if err != nil {
			return err
		}

		if resp != nil && resp.ReasonCode >= reasonFailure {
			return fmt.Errorf("courier: publish rejected, reason code 0x%x", resp.ReasonCode)
		}

		return nil
	})
}

// Subscribe subscribes to a single topic filter.
func (v *Client) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	return v.SubscribeMultiple(map[string]byte{topic: qos}, callback)
}

// SubscribeMultiple subscribes to multiple topic filters. The route is registered
// immediately even if the client is not yet connected, so a handler added before Connect
// sees whatever a resumed session already had queued; the broker gets the wire SUBSCRIBE
// once connected, unless the session resumed and already has it.
func (v *Client) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	for topic, qos := range filters {
		v.addRoute(topic, qos, callback)
	}

	v.mu.Lock()
	cm := v.cm
	v.mu.Unlock()

	if cm == nil {
		return completed(nil)
	}

	subs := make([]paho.SubscribeOptions, 0, len(filters))
	for topic, qos := range filters {
		subs = append(subs, paho.SubscribeOptions{Topic: topic, QoS: qos})
	}

	return v.run(context.Background(), func(ctx context.Context, cm *autopaho.ConnectionManager) error {
		resp, err := cm.Subscribe(ctx, &paho.Subscribe{Subscriptions: subs})
		if err == nil {
			err = firstFailure("subscribe", resp.Reasons)
		}

		if err != nil {
			for topic := range filters {
				v.removeRoute(topic)
			}
		}

		return err
	})
}

// Unsubscribe removes the subscriptions for the topic filters.
func (v *Client) Unsubscribe(topics ...string) mqtt.Token {
	return v.run(context.Background(), func(ctx context.Context, cm *autopaho.ConnectionManager) error {
		resp, err := cm.Unsubscribe(ctx, &paho.Unsubscribe{Topics: topics})
		if err != nil {
			return err
		}

		for _, topic := range topics {
			v.removeRoute(topic)
		}

		return firstFailure("unsubscribe", resp.Reasons)
	})
}

// AddRoute adds a handler for messages on a topic filter without subscribing.
func (v *Client) AddRoute(topic string, callback mqtt.MessageHandler) {
	v.mu.Lock()
	v.subs[topic] = subscription{cb: callback, noSub: true}
	v.mu.Unlock()
}

func (v *Client) addRoute(topic string, qos byte, callback mqtt.MessageHandler) {
	v.mu.Lock()
	v.subs[topic] = subscription{qos: qos, cb: callback}
	v.mu.Unlock()
}

func (v *Client) removeRoute(topic string) {
	v.mu.Lock()
	delete(v.subs, topic)
	v.mu.Unlock()
}

// route delivers a received message to the handlers whose filter matches it. paho's own router runs
// handlers while holding a read lock it also needs to register one, so a handler that subscribes
// there deadlocks. The handlers are copied out first so none of them runs under v.mu.
func (v *Client) route(pr paho.PublishReceived) (bool, error) {
	v.mu.Lock()
	handlers := make([]mqtt.MessageHandler, 0, len(v.subs))

	for filter, s := range v.subs {
		if topicMatches(filter, pr.Packet.Topic) {
			handlers = append(handlers, s.cb)
		}
	}
	v.mu.Unlock()

	if len(handlers) == 0 {
		// paho acks it anyway, so the broker drops it, and withholding the ack would stall every
		// later one behind it in paho's in-order tracker.
		v.errs.Printf("courier: no handler for a message on %q, it was dropped", pr.Packet.Topic)

		return false, nil
	}

	for _, h := range handlers {
		h(v, &message{p: pr.Packet})
	}

	return true, nil
}

// topicMatches reports whether an MQTT topic filter matches a topic name.
func topicMatches(filter, topic string) bool {
	if filter == topic {
		return true
	}

	f := strings.Split(filter, "/")
	if len(f) > 2 && f[0] == "$share" {
		f = f[2:] // a shared subscription is delivered under the topic behind the group
	}

	t := strings.Split(topic, "/")

	for i, part := range f {
		if part == "#" {
			return i == len(f)-1
		}

		if i >= len(t) || (part != "+" && part != t[i]) {
			return false
		}
	}

	return len(f) == len(t)
}

// run calls f and returns an already completed token. The v5 library blocks until the broker
// answers, unlike the v3 one, so there is nothing to wait for afterwards. The call ends when
// the caller's ctx ends or opTimeout passes, whichever comes first.
func (v *Client) run(ctx context.Context, f func(context.Context, *autopaho.ConnectionManager) error) mqtt.Token {
	v.mu.Lock()
	cm := v.cm
	v.mu.Unlock()

	if cm == nil {
		return completed(ErrNotStarted)
	}

	ctx, cancel := context.WithTimeout(ctx, v.opTimeout())
	defer cancel()

	return completed(f(ctx, cm))
}

func (v *Client) opTimeout() time.Duration {
	if v.opts.AckTimeout > 0 {
		return v.opts.AckTimeout
	}

	return defaultOpTimeout
}

func firstFailure(op string, reasons []byte) error {
	for _, r := range reasons {
		if r >= reasonFailure {
			return fmt.Errorf("courier: %s rejected, reason code 0x%x", op, r)
		}
	}

	return nil
}

func toBytes(payload interface{}) ([]byte, error) {
	switch p := payload.(type) {
	case []byte:
		return p, nil
	case string:
		return []byte(p), nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("courier: unsupported payload type %T", payload)
	}
}

// message adapts a received PUBLISH to mqtt.Message.
type message struct {
	p *paho.Publish
}

func (m *message) Duplicate() bool   { return m.p.Duplicate() }
func (m *message) Qos() byte         { return m.p.QoS }
func (m *message) Retained() bool    { return m.p.Retain }
func (m *message) Topic() string     { return m.p.Topic }
func (m *message) MessageID() uint16 { return m.p.PacketID }
func (m *message) Payload() []byte   { return m.p.Payload }
func (m *message) Ack()              {}

type token struct {
	once sync.Once
	done chan struct{}
	err  error
}

func newToken() *token { return &token{done: make(chan struct{})} }

// completed returns a token that already holds the result of a finished call.
func completed(err error) *token {
	t := newToken()
	t.complete(err)

	return t
}

func (t *token) complete(err error) {
	t.once.Do(func() {
		t.err = err
		close(t.done)
	})
}

func (t *token) Wait() bool {
	<-t.done

	return true
}

func (t *token) WaitTimeout(d time.Duration) bool {
	select {
	case <-t.done:
		return true
	case <-time.After(d):
		return false
	}
}

func (t *token) Done() <-chan struct{} { return t.done }

func (t *token) Error() error {
	select {
	case <-t.done:
		return t.err
	default:
		return nil
	}
}
