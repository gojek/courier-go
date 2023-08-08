package courier

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ErrClientNotInitialized is returned when the client is not initialized
var ErrClientNotInitialized = errors.New("courier: client not initialized")

var newClientFunc = defaultNewClientFunc()

// Client allows to communicate with an MQTT broker
type Client struct {
	options    *clientOptions
	mqttClient mqtt.Client

	publisher     Publisher
	subscriber    Subscriber
	unsubscriber  Unsubscriber
	pMiddlewares  []publishMiddleware
	sMiddlewares  []subscribeMiddleware
	usMiddlewares []unsubscribeMiddleware

	mu sync.RWMutex
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

	c := &Client{options: co}

	if len(co.brokerAddress) != 0 {
		c.mqttClient = newClientFunc.Load().(func(*mqtt.ClientOptions) mqtt.Client)(toClientOptions(c, c.options))
	}

	c.publisher = publishHandler(c)
	c.subscriber = subscriberFuncs(c)
	c.unsubscriber = unsubscriberHandler(c)

	return c, nil
}

// IsConnected checks whether the client is connected to the broker
func (c *Client) IsConnected() bool {
	var online bool

	err := c.execute(func(cc mqtt.Client) {
		online = cc.IsConnectionOpen()
	})

	return err == nil && online
}

// Start will attempt to connect to the broker.
func (c *Client) Start() (err error) {
	if len(c.options.brokerAddress) != 0 {
		err = c.execute(func(cc mqtt.Client) {
			t := cc.Connect()
			if !t.WaitTimeout(c.options.connectTimeout) {
				err = ErrConnectTimeout

				return
			}

			err = t.Error()
		})
	}

	if c.options.resolver != nil {
		// try first connect attempt on start, then start a watcher on channel
		select {
		case <-time.After(c.options.connectTimeout):
			err = ErrConnectTimeout

			return
		case addrs := <-c.options.resolver.UpdateChan():
			c.attemptConnection(addrs)
		}

		go c.watchAddressUpdates(c.options.resolver)
	}

	return
}

// Stop will disconnect from the broker and finish up any pending work on internal
// communication workers. This can only block until the period configured with
// the ClientOption WithGracefulShutdownPeriod.
func (c *Client) Stop() {
	_ = c.execute(func(cc mqtt.Client) {
		cc.Disconnect(uint(c.options.gracefulShutdownPeriod / time.Millisecond))
	})
}

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
	c.Stop()

	return nil
}

func (c *Client) execute(f func(mqtt.Client)) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.mqttClient == nil {
		return ErrClientNotInitialized
	}

	f(c.mqttClient)

	return nil
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

func toClientOptions(c *Client, o *clientOptions) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()

	if hostname, err := os.Hostname(); o.clientID == "" && err == nil {
		opts.SetClientID(hostname)
	} else {
		opts.SetClientID(o.clientID)
	}

	setCredentials(o, opts)

	opts.AddBroker(formatAddressWithProtocol(o)).
		SetTLSConfig(o.tlsConfig).
		SetAutoReconnect(o.autoReconnect).
		SetCleanSession(o.cleanSession).
		SetOrderMatters(o.maintainOrder).
		SetKeepAlive(o.keepAlive).
		SetConnectTimeout(o.connectTimeout).
		SetMaxReconnectInterval(o.maxReconnectInterval).
		SetReconnectingHandler(reconnectHandler(c, o)).
		SetConnectionLostHandler(connectionLostHandler(o)).
		SetOnConnectHandler(onConnectHandler(c, o))

	return opts
}

func setCredentials(o *clientOptions, opts *mqtt.ClientOptions) {
	if o.credentialFetcher != nil {
		ctx, cancel := context.WithTimeout(context.Background(), o.credentialFetchTimeout)
		defer cancel()

		if c, err := o.credentialFetcher.Credentials(ctx); err == nil {
			opts.SetUsername(c.Username)
			opts.SetPassword(c.Password)

			return
		}
	}

	opts.SetUsername(o.username)
	opts.SetPassword(o.password)
}

func formatAddressWithProtocol(opts *clientOptions) string {
	if opts.tlsConfig != nil {
		return fmt.Sprintf("tls://%s", opts.brokerAddress)
	}

	return fmt.Sprintf("tcp://%s", opts.brokerAddress)
}

func reconnectHandler(client PubSub, o *clientOptions) mqtt.ReconnectHandler {
	return func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		if o.onReconnectHandler != nil {
			o.onReconnectHandler(client)
		}
	}
}

func connectionLostHandler(o *clientOptions) mqtt.ConnectionLostHandler {
	return func(_ mqtt.Client, err error) {
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
