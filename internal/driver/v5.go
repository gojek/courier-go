// Package driver holds the MQTT v5 client used by courier when MQTTv5 is selected.
//
// V5 implements the gojek/paho.mqtt.golang Client interface on top of eclipse/paho.golang,
// so the pool, resolver, telemetry and middleware code in courier works unchanged for both protocol versions.
package driver

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"

	mqtt "github.com/gojek/paho.mqtt.golang"
)

const (
	defaultOpTimeout = 30 * time.Second
	// MQTT v5 reason codes at or above this value are failures.
	reasonFailure = 0x80
)

// ErrNotStarted is returned when an operation is attempted before Connect.
var ErrNotStarted = errors.New("courier: mqtt v5 client not connected")

// PublishProps are the MQTT v5 properties of a PUBLISH packet.
type PublishProps struct {
	UserProperties  map[string]string
	ContentType     string
	ResponseTopic   string
	CorrelationData []byte
}

// PropsPublisher is implemented by clients that can publish with MQTT v5 properties.
type PropsPublisher interface {
	PublishWithProps(topic string, qos byte, retained bool, payload interface{}, props *PublishProps) mqtt.Token
}

// MessageWithProps is implemented by received messages that carry MQTT v5 properties.
type MessageWithProps interface {
	mqtt.Message
	Props() *PublishProps
}

type subscription struct {
	qos byte
	cb  mqtt.MessageHandler
}

// V5 is an MQTT v5 implementation of mqtt.Client.
type V5 struct {
	// never connected, only serves OptionsReader() so client metadata keeps working.
	mqtt.Client

	opts   *mqtt.ClientOptions
	v5     V5Options
	router *paho.StandardRouter

	mu     sync.Mutex
	cm     *autopaho.ConnectionManager
	cancel context.CancelFunc
	subs   map[string]subscription

	up     atomic.Bool
	closed atomic.Bool
}

// V5Options are the settings that exist only in MQTT v5.
type V5Options struct {
	// CleanStart clears any existing session on the first connection.
	CleanStart bool
	// SessionExpiryInterval is how long the broker keeps the session after disconnect, 0 ends it on disconnect.
	SessionExpiryInterval time.Duration
}

// NewV5 creates an MQTT v5 client from v3 style options plus the v5 only V5Options.
// KeepAlive, ConnectTimeout and ConnectRetry(Interval) are read from opts, CleanSession is not used.
func NewV5(opts *mqtt.ClientOptions, v5 V5Options) *V5 {
	return &V5{
		Client: mqtt.NewClient(opts),
		opts:   opts,
		v5:     v5,
		router: paho.NewStandardRouter(),
		subs:   map[string]subscription{},
	}
}

// IsConnected reports whether the connection to the broker is up.
func (v *V5) IsConnected() bool { return v.up.Load() }

// IsConnectionOpen reports whether the connection to the broker is up.
func (v *V5) IsConnectionOpen() bool { return v.up.Load() }

// Connect starts connecting, the token completes on first CONNACK.
// shortcut: reconnect delay is constant (ConnectRetryInterval), upgrade to
// autopaho.NewExponentialBackoff if brokers need it.
func (v *V5) Connect() mqtt.Token {
	t := newToken()

	v.closed.Store(false)

	ctx, cancel := context.WithCancel(context.Background())

	cm, err := autopaho.NewConnection(ctx, v.config(t, cancel))
	if err != nil {
		cancel()
		t.complete(err)

		return t
	}

	v.mu.Lock()
	v.cm, v.cancel = cm, cancel
	v.mu.Unlock()

	return t
}

func (v *V5) config(connectToken *token, cancel context.CancelFunc) autopaho.ClientConfig {
	o := v.opts

	cfg := autopaho.ClientConfig{
		ServerUrls:                    o.Servers,
		TlsCfg:                        o.TLSConfig,
		KeepAlive:                     uint16(o.KeepAlive),
		CleanStartOnInitialConnection: v.v5.CleanStart,
		ConnectTimeout:                o.ConnectTimeout,
		ReconnectBackoff: func(attempt int) time.Duration {
			if attempt == 0 { // autopaho also waits before the first attempt
				return 0
			}

			if o.ConnectRetryInterval > 0 {
				return o.ConnectRetryInterval
			}

			return 10 * time.Second
		},
		ConnectPacketBuilder: v.credentials,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, ack *paho.Connack) {
			v.up.Store(true)
			connectToken.complete(nil)

			if !ack.SessionPresent {
				go v.resubscribe(cm)
			}

			if o.OnConnect != nil {
				go o.OnConnect(v)
			}
		},
		OnConnectError: func(err error) {
			if !o.ConnectRetry {
				connectToken.complete(err)
				cancel()
			}
		},
		ClientConfig: paho.ClientConfig{
			ClientID:      o.ClientID,
			Router:        v.router,
			PacketTimeout: o.AckTimeout,
			OnClientError: func(err error) { v.connectionLost(err, cancel) },
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

// credentials is called for every connection attempt so refreshed credentials are picked up.
func (v *V5) credentials(c *paho.Connect, _ *url.URL) (*paho.Connect, error) {
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
func (v *V5) connectionLost(err error, cancel context.CancelFunc) {
	if !v.up.Swap(false) || v.closed.Load() {
		return
	}

	if !v.opts.AutoReconnect {
		cancel() // autopaho always reconnects unless cancelled
	}

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
func (v *V5) resubscribe(cm *autopaho.ConnectionManager) {
	v.mu.Lock()
	subs := make([]paho.SubscribeOptions, 0, len(v.subs))

	for topic, s := range v.subs {
		subs = append(subs, paho.SubscribeOptions{Topic: topic, QoS: s.qos})
	}
	v.mu.Unlock()

	if len(subs) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), v.opTimeout())
	defer cancel()

	_, _ = cm.Subscribe(ctx, &paho.Subscribe{Subscriptions: subs})
}

// Disconnect ends the connection, quiesce is the time in milliseconds allowed for a clean shutdown.
func (v *V5) Disconnect(quiesce uint) {
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

// Publish publishes without MQTT v5 properties.
func (v *V5) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	return v.PublishWithProps(topic, qos, retained, payload, nil)
}

// PublishWithProps publishes with the given MQTT v5 properties.
func (v *V5) PublishWithProps(
	topic string, qos byte, retained bool, payload interface{}, props *PublishProps,
) mqtt.Token {
	b, err := toBytes(payload)
	if err != nil {
		return failed(err)
	}

	p := &paho.Publish{Topic: topic, QoS: qos, Retain: retained, Payload: b}

	if props != nil {
		p.Properties = &paho.PublishProperties{
			ContentType:     props.ContentType,
			ResponseTopic:   props.ResponseTopic,
			CorrelationData: props.CorrelationData,
			User:            userProperties(props.UserProperties),
		}
	}

	return v.run(func(ctx context.Context, cm *autopaho.ConnectionManager) error {
		resp, err := cm.Publish(ctx, p)
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
func (v *V5) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	return v.SubscribeMultiple(map[string]byte{topic: qos}, callback)
}

// SubscribeMultiple subscribes to multiple topic filters.
func (v *V5) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	subs := make([]paho.SubscribeOptions, 0, len(filters))
	for topic, qos := range filters {
		subs = append(subs, paho.SubscribeOptions{Topic: topic, QoS: qos})
	}

	return v.run(func(ctx context.Context, cm *autopaho.ConnectionManager) error {
		for topic, qos := range filters {
			v.addRoute(topic, qos, callback)
		}

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
func (v *V5) Unsubscribe(topics ...string) mqtt.Token {
	return v.run(func(ctx context.Context, cm *autopaho.ConnectionManager) error {
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
func (v *V5) AddRoute(topic string, callback mqtt.MessageHandler) {
	v.router.RegisterHandler(topic, v.handler(callback))
}

func (v *V5) addRoute(topic string, qos byte, callback mqtt.MessageHandler) {
	v.mu.Lock()
	v.subs[topic] = subscription{qos: qos, cb: callback}
	v.mu.Unlock()

	v.router.UnregisterHandler(topic)
	v.router.RegisterHandler(topic, v.handler(callback))
}

func (v *V5) removeRoute(topic string) {
	v.mu.Lock()
	delete(v.subs, topic)
	v.mu.Unlock()

	v.router.UnregisterHandler(topic)
}

func (v *V5) handler(callback mqtt.MessageHandler) paho.MessageHandler {
	return func(p *paho.Publish) { callback(v, &message{p: p}) }
}

func (v *V5) run(f func(context.Context, *autopaho.ConnectionManager) error) mqtt.Token {
	v.mu.Lock()
	cm := v.cm
	v.mu.Unlock()

	if cm == nil {
		return failed(ErrNotStarted)
	}

	t := newToken()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), v.opTimeout())
		defer cancel()

		t.complete(f(ctx, cm))
	}()

	return t
}

func (v *V5) opTimeout() time.Duration {
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

func userProperties(m map[string]string) paho.UserProperties {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	up := make(paho.UserProperties, 0, len(keys))
	for _, k := range keys {
		up.Add(k, m[k])
	}

	return up
}

// message adapts a received PUBLISH to mqtt.Message and MessageWithProps.
type message struct{ p *paho.Publish }

func (m *message) Duplicate() bool   { return m.p.Duplicate() }
func (m *message) Qos() byte         { return m.p.QoS }
func (m *message) Retained() bool    { return m.p.Retain }
func (m *message) Topic() string     { return m.p.Topic }
func (m *message) MessageID() uint16 { return m.p.PacketID }
func (m *message) Payload() []byte   { return m.p.Payload }
func (m *message) Ack()              {}

func (m *message) Props() *PublishProps {
	if m.p.Properties == nil {
		return nil
	}

	props := &PublishProps{
		ContentType:     m.p.Properties.ContentType,
		ResponseTopic:   m.p.Properties.ResponseTopic,
		CorrelationData: m.p.Properties.CorrelationData,
	}

	if len(m.p.Properties.User) > 0 {
		props.UserProperties = make(map[string]string, len(m.p.Properties.User))
		for _, u := range m.p.Properties.User {
			props.UserProperties[u.Key] = u.Value
		}
	}

	return props
}

type token struct {
	once sync.Once
	done chan struct{}
	err  error
}

func newToken() *token { return &token{done: make(chan struct{})} }

func failed(err error) *token {
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
