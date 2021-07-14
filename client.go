package courier

import (
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"

	"***REMOVED***/metrics"
)

var newClientFunc = mqtt.NewClient

// Client allows to communicate with an MQTT broker
type Client struct {
	options         *options
	metricsExchange *metricsExchange
	mqttClient      mqtt.Client

	publisher     Publisher
	subscriber    Subscriber
	unsubscriber  Unsubscriber
	pMiddlewares  []publishMiddleware
	sMiddlewares  []subscribeMiddleware
	usMiddlewares []unsubscribeMiddleware
}

// NewClient creates the Client struct with the options provided,
// it can return error when prometheus.DefaultRegisterer has already
// been used to register the collected metrics
func NewClient(opts ...Option) (*Client, error) {
	o := defaultOptions()
	for _, f := range opts {
		f(o)
	}
	if o.metricsCollector == nil {
		m := metrics.NewPrometheus()
		if err := m.AddToRegistry(prometheus.DefaultRegisterer); err != nil {
			return nil, err
		}
		o.metricsCollector = m
	}
	me := &metricsExchange{
		updates:   make(chan updateEvent),
		collector: o.metricsCollector,
	}
	c := &Client{options: o, metricsExchange: me}
	go me.monitor()
	c.mqttClient = newClientFunc(toClientOptions(c, c.options))
	c.publisher = publishHandler(c)
	c.subscriber = subscriberFuncs(c)
	c.unsubscriber = unsubscriberHandler(c)
	return c, nil
}

// IsConnected checks whether the client is connected to the broker
func (c *Client) IsConnected() bool {
	return c.mqttClient != nil && c.mqttClient.IsConnectionOpen()
}

// Start will attempt to connect to the broker.
func (c *Client) Start() error {
	t := c.mqttClient.Connect()
	if !t.WaitTimeout(c.options.connectTimeout) {
		return ErrConnectTimeout
	}
	if err := t.Error(); err != nil {
		return err
	}
	return nil
}

// Stop will disconnect from the broker and finish up any pending work on internal
// communication workers. This can only block until the period configured with
// the Option WithGracefulShutdownPeriod.
func (c *Client) Stop() {
	c.mqttClient.Disconnect(uint(c.options.gracefulShutdownPeriod / time.Millisecond))
}

func (c *Client) handleToken(t mqtt.Token, w *eventWrapper, timeoutErr error) error {
	if !t.WaitTimeout(c.options.writeTimeout) {
		w.types |= timeoutEvent
		return timeoutErr
	}
	if err := t.Error(); err != nil {
		w.types |= errorEvent
		return err
	}
	w.types |= successEvent
	return nil
}

func toClientOptions(c *Client, o *options) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	if hostname, err := os.Hostname(); err == nil {
		opts.SetClientID(hostname)
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

func reconnectHandler(client *Client, o *options) mqtt.ReconnectHandler {
	return func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		if o.onReconnectHandler != nil {
			o.onReconnectHandler(client)
		}
	}
}

func connectionLostHandler(o *options) mqtt.ConnectionLostHandler {
	return func(_ mqtt.Client, err error) {
		if o.onConnectionLostHandler != nil {
			o.onConnectionLostHandler(err)
		}
	}
}

func onConnectHandler(client *Client, o *options) mqtt.OnConnectHandler {
	return func(_ mqtt.Client) {
		if o.onConnectHandler != nil {
			o.onConnectHandler(client)
		}
	}
}
