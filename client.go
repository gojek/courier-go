package courier

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"os"
	"sync"
	"time"
)

var newClientFunc = mqtt.NewClient

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
	o := defaultOptions()

	for _, f := range opts {
		f(o)
	}

	c := &Client{options: o}

	c.mqttClient = newClientFunc(toClientOptions(c, c.options))
	c.publisher = publishHandler(c)
	c.subscriber = subscriberFuncs(c)
	c.unsubscriber = unsubscriberHandler(c)

	if o.resolver != nil {
		go c.watchAddressUpdates(o.resolver)
	}

	return c, nil
}

func (c *Client) execute(f func(mqtt.Client)) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	f(c.mqttClient)
}

// IsConnected checks whether the client is connected to the broker
func (c *Client) IsConnected() (online bool) {
	c.execute(func(cc mqtt.Client) {
		online = cc != nil && cc.IsConnectionOpen()
	})
	return
}

// Start will attempt to connect to the broker.
func (c *Client) Start() (err error) {
	c.execute(func(cc mqtt.Client) {
		t := cc.Connect()
		if !t.WaitTimeout(c.options.connectTimeout) {
			err = ErrConnectTimeout
			return
		}

		err = t.Error()
	})
	return
}

// Stop will disconnect from the broker and finish up any pending work on internal
// communication workers. This can only block until the period configured with
// the ClientOption WithGracefulShutdownPeriod.
func (c *Client) Stop() {
	c.execute(func(cc mqtt.Client) {
		cc.Disconnect(uint(c.options.gracefulShutdownPeriod / time.Millisecond))
	})
}

func (c *Client) handleToken(t mqtt.Token, timeoutErr error) error {
	if !t.WaitTimeout(c.options.writeTimeout) {
		return timeoutErr
	}

	if err := t.Error(); err != nil {
		return err
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

	opts.AddBroker(o.brokerAddress).
		SetUsername(o.username).
		SetPassword(o.password).
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
