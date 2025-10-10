package courier

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/gojek/paho.mqtt.golang"
)

// ErrClientNotInitialized is returned when the client is not initialized
var ErrClientNotInitialized = errors.New("courier: client not initialized")

var newClientFunc = defaultNewClientFunc()

// Client allows to communicate with an MQTT broker
type Client struct {
	options *clientOptions

	subscriptions map[string]*subscriptionMeta
	mqttClient    mqtt.Client
	mqttClients   map[string]*internalState

	publisher      Publisher
	subscriber     Subscriber
	unsubscriber   Unsubscriber
	stopHandler    Stopper
	pMiddlewares   []publishMiddleware
	sMiddlewares   []subscribeMiddleware
	usMiddlewares  []unsubscribeMiddleware
	stopMiddleware stopMiddleware

	rrCounter         *atomicCounter
	multiConnRevision atomic.Uint64
	rndPool           *sync.Pool
	clientMu          sync.RWMutex
	subMu             sync.RWMutex

	stopInfoEmitter context.CancelFunc
}

// NewClient creates the Client struct with the clientOptions provided,
// it can return error when prometheus.DefaultRegisterer has already
// been used to register the collected metrics
func NewClient(opts ...ClientOption) (*Client, error) {
	co := defaultClientOptions()

	for _, opt := range opts {
		opt.apply(co)
	}

	if len(co.brokerAddress) == 0 && co.resolver == nil {
		return nil, fmt.Errorf("at least WithAddress or WithResolver ClientOption should be used")
	}

	if co.infoEmitterCfg != nil && co.infoEmitterCfg.Emitter != nil && co.infoEmitterCfg.Interval.Seconds() < 1 {
		return nil, fmt.Errorf("client info emitter interval must be greater than or equal to 1s")
	}

	c := &Client{
		options:       co,
		subscriptions: map[string]*subscriptionMeta{},
		rrCounter:     &atomicCounter{value: 0},
		rndPool: &sync.Pool{New: func() any {
			return rand.New(rand.NewSource(time.Now().UnixNano()))
		}},
	}

	if len(co.brokerAddress) != 0 {
		c.mqttClient = newClientFunc.Load().(func(*mqtt.ClientOptions) mqtt.Client)(toClientOptions(c, c.options, ""))
	}

	c.publisher = publishHandler(c)
	c.subscriber = subscriberFuncs(c)
	c.unsubscriber = unsubscriberHandler(c)
	c.stopHandler = stopHandlerFunc(c)

	return c, nil
}

// IsConnected checks whether the client is connected to the broker
func (c *Client) IsConnected() bool {
	val := &atomic.Bool{}

	return c.execute(func(cc mqtt.Client) error {
		if cc.IsConnectionOpen() {
			val.CompareAndSwap(false, true)
		}

		return nil
	}, execAll) == nil && val.Load()
}

// Start will attempt to connect to the broker.
func (c *Client) Start() error {
	if c.options.resolver != nil {
		return c.runResolver()
	}

	return c.runConnect()
}

// Stop will disconnect from the broker and finish up any pending work on internal
// communication workers. This can only block until the period configured with
// the ClientOption WithGracefulShutdownPeriod.
func (c *Client) Stop() { _ = c.stopHandler.Stop() }

// Run will start running the Client. This makes Client compatible with github.com/gojekfarm/xrun package.
// https://pkg.go.dev/github.com/gojekfarm/xrun
func (c *Client) Run(ctx context.Context) error {
	if c.options.startOptions != nil {
		exponentialStartStrategy(ctx, c, c.options.startOptions)
	} else {
		if err := c.Start(); err != nil {
			return err
		}
	}

	<-ctx.Done()

	return c.stop()
}

func (c *Client) stop() error {
	err := c.execute(func(cc mqtt.Client) error {
		cc.Disconnect(uint(c.options.gracefulShutdownPeriod / time.Millisecond))

		return nil
	}, execAll)

	if c.stopInfoEmitter != nil {
		c.stopInfoEmitter()
	}

	if err == nil {
		c.clientMu.Lock()
		defer c.clientMu.Unlock()

		c.mqttClient = nil
		c.mqttClients = nil
	}

	return err
}

func (c *Client) handleInfoEmitter() {
	if c.options.infoEmitterCfg != nil && c.options.infoEmitterCfg.Emitter != nil {
		ctx, cancel := context.WithCancel(context.Background())
		c.stopInfoEmitter = cancel

		go c.runBrokerInfoEmitter(ctx)
	}
}

func (c *Client) handleToken(ctx context.Context, t mqtt.Token, timeoutErr error) error {
	if err := c.waitForToken(ctx, t, timeoutErr); err != nil {
		return err
	}

	if err := t.Error(); err != nil {
		return err
	}

	return nil
}

func (c *Client) waitForToken(ctx context.Context, t mqtt.Token, timeoutErr error) error {
	if _, ok := ctx.Deadline(); ok {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.Done():
			return t.Error()
		}
	}

	if !t.WaitTimeout(c.options.writeTimeout) {
		return timeoutErr
	}

	return nil
}

func (c *Client) runResolver() error {
	// try first connect attempt on start, then start a watcher on channel
	select {
	case <-time.After(c.options.connectTimeout):
		return ErrConnectTimeout
	case addrs := <-c.options.resolver.UpdateChan():
		if err := c.attemptConnections(addrs); err != nil {
			return err
		}
	}

	c.handleInfoEmitter()

	go c.watchAddressUpdates(c.options.resolver)

	return nil
}

func (c *Client) runConnect() error {
	err := c.execute(func(cc mqtt.Client) error {
		t := cc.Connect()
		if !t.WaitTimeout(c.options.connectTimeout) {
			return ErrConnectTimeout
		}

		return t.Error()
	}, execAll)

	if err != nil {
		return err
	}

	c.handleInfoEmitter()

	return nil
}

func (c *Client) attemptSingleConnection(addrs []TCPAddress) error {
	if len(addrs) == 0 {
		c.reloadClient(nil)

		return nil
	}

	cc := c.newClient(addrs, 0)
	c.reloadClient(cc)

	return c.resumeSubscriptions()
}

func (c *Client) removeStoredSubsCalled(cc mqtt.Client) {
	c.clientMu.RLock()
	defer c.clientMu.RUnlock()

	for _, v := range c.mqttClients {
		if v.client == cc {
			v.mu.Lock()
			v.subsCalled.Delete(v.subsCalled.Values()...)
			v.mu.Unlock()
		}
	}
}

func toClientOptions(c *Client, o *clientOptions, idSuffix string) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()

	if hostname, err := os.Hostname(); o.clientID == "" && err == nil {
		opts.SetClientID(fmt.Sprintf("%s%s", hostname, idSuffix))
	} else {
		opts.SetClientID(fmt.Sprintf("%s%s", o.clientID, idSuffix))
	}

	setCredentials(o, opts)

	if o.connectRetryPolicy.enabled {
		opts.SetConnectRetry(true)
		opts.SetConnectRetryInterval(o.connectRetryPolicy.interval)
	}

	opts.AddBroker(formatAddressWithProtocol(o)).
		SetResumeSubs(o.resumeSubscriptions).
		SetTLSConfig(o.tlsConfig).
		SetAutoReconnect(o.autoReconnect).
		SetCleanSession(o.cleanSession).
		SetOrderMatters(o.maintainOrder).
		SetKeepAlive(o.keepAlive).
		SetConnectTimeout(o.connectTimeout).
		SetMaxReconnectInterval(o.maxReconnectInterval).
		SetReconnectingHandler(reconnectHandler(c, o)).
		SetConnectionLostHandler(connectionLostHandler(c, o)).
		SetOnConnectHandler(onConnectHandler(c, o)).
		SetWriteTimeout(o.writeTimeout).
		SetLogLevel(o.pahoLogLevel).
		SetAckTimeout(o.ackTimeout)

	return opts
}

func setCredentials(o *clientOptions, opts *mqtt.ClientOptions) {
	if o.credentialFetcher != nil {
		refreshCredentialsWithFetcher(o, opts)

		opts.SetCredentialsProvider(credentialsRefresher(o))

		return
	}

	opts.SetUsername(o.username)
	opts.SetPassword(o.password)
}

func credentialsRefresher(o *clientOptions) func() (string, string) {
	return func() (string, string) {
		ctx, cancel := context.WithTimeout(context.Background(), o.credentialFetchTimeout)
		defer cancel()

		c, err := o.credentialFetcher.Credentials(ctx)
		if err != nil {
			o.logger.Error(ctx, err, map[string]any{"message": "failed to fetch credentials"})

			return "<unknown>", ""
		}

		return c.Username, c.Password
	}
}

func refreshCredentialsWithFetcher(o *clientOptions, opts *mqtt.ClientOptions) {
	ctx, cancel := context.WithTimeout(context.Background(), o.credentialFetchTimeout)
	defer cancel()

	c, err := o.credentialFetcher.Credentials(ctx)
	if err != nil {
		o.logger.Error(ctx, err, map[string]any{"message": "failed to fetch credentials"})

		return
	}

	opts.SetUsername(c.Username)
	opts.SetPassword(c.Password)
}

func formatAddressWithProtocol(opts *clientOptions) string {
	if opts.tlsConfig != nil {
		return fmt.Sprintf("tls://%s", opts.brokerAddress)
	}

	return fmt.Sprintf("tcp://%s", opts.brokerAddress)
}

func reconnectHandler(client PubSub, o *clientOptions) mqtt.ReconnectHandler {
	return func(_ mqtt.Client, opts *mqtt.ClientOptions) {
		if o.logger != nil {
			o.logger.Info(context.Background(), "reconnecting", map[string]any{"client_id": opts.ClientID})
		}

		if o.onReconnectHandler != nil {
			o.onReconnectHandler(client)
		}
	}
}

func connectionLostHandler(c *Client, o *clientOptions) mqtt.ConnectionLostHandler {
	return func(cc mqtt.Client, err error) {
		if o.logger != nil {
			o.logger.Error(context.Background(), err, map[string]any{
				"message":   "connection lost",
				"client_id": clientIDMapper(cc),
			})
		}

		c.removeStoredSubsCalled(cc)

		if o.onConnectionLostHandler != nil {
			o.onConnectionLostHandler(err)
		}
	}
}

func onConnectHandler(client PubSub, o *clientOptions) mqtt.OnConnectHandler {
	return func(_ mqtt.Client) {
		if o.onConnectHandler != nil {
			o.onConnectHandler(client)
		}
	}
}

func defaultNewClientFunc() *atomic.Value {
	v := &atomic.Value{}
	v.Store(mqtt.NewClient)

	return v
}
