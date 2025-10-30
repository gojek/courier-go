package courier

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/gojek/paho.mqtt.golang"
)

var inMemoryPersistence = NewMemoryStore()

func defaultSharedSubscriptionPredicate(topic string) bool {
	return strings.HasPrefix(topic, "$share/")
}

// ClientOption allows to configure the behaviour of a Client.
type ClientOption interface{ apply(*clientOptions) }

// WithClientID sets the clientID to be used while connecting to an MQTT broker.
// According to the MQTT v3.1 specification, a client id must be no longer than 23 characters.
func WithClientID(clientID string) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.clientID = clientID
	})
}

// WithUsername sets the username to be used while connecting to an MQTT broker.
func WithUsername(username string) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.username = username
	})
}

// WithPassword sets the password to be used while connecting to an MQTT broker.
func WithPassword(password string) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.password = password
	})
}

// WithTLS sets the TLs configuration to be used while connecting to an MQTT broker.
func WithTLS(tlsConfig *tls.Config) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.tlsConfig = tlsConfig
	})
}

// WithAutoReconnect sets whether the automatic reconnection logic should be used
// when the connection is lost, even if disabled the WithOnConnectionLost is still called.
func WithAutoReconnect(autoReconnect bool) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.autoReconnect = autoReconnect
	})
}

// WithCleanSession will set the "clean session" flag in the connect message
// when this client connects to an MQTT broker. By setting this flag, you are
// indicating that no messages saved by the broker for this client should be
// delivered. Any messages that were going to be sent by this client before
// disconnecting but didn't, will not be sent upon connecting to the
// broker.
func WithCleanSession(cleanSession bool) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.cleanSession = cleanSession
	})
}

// WithMaintainOrder will set the message routing to guarantee order within
// each QoS level. By default, this value is true. If set to false (recommended),
// this flag indicates that messages can be delivered asynchronously
// from the client to the application and possibly arrive out of order.
// Specifically, the message handler is called in its own go routine.
// Note that setting this to true does not guarantee in-order delivery
// (this is subject to broker settings like "max_inflight_messages=1")
// and if true then  MessageHandler callback must not block.
func WithMaintainOrder(maintainOrder bool) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.maintainOrder = maintainOrder
	})
}

// WithOnConnect will set the OnConnectHandler callback to be called when the client is connected.
// Both at initial connection time and upon automatic reconnect.
func WithOnConnect(handler OnConnectHandler) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.onConnectHandler = handler
	})
}

// WithOnConnectionLost will set the OnConnectionLostHandler callback to be executed
// in the case where the client unexpectedly loses connection with the MQTT broker.
func WithOnConnectionLost(handler OnConnectionLostHandler) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.onConnectionLostHandler = handler
	})
}

// WithOnReconnect sets the OnReconnectHandler callback to be executed prior
// to the client attempting a reconnect to the MQTT broker.
func WithOnReconnect(handler OnReconnectHandler) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.onReconnectHandler = handler
	})
}

// WithTCPAddress sets the broker address to be used.
// Default values for hostname is "127.0.0.1" and for port is 1883.
//
// Deprecated: This Option used to work with plain TCP connections,
// it's now possible to use TLS with WithAddress and WithTLS combination.
func WithTCPAddress(host string, port uint16) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.brokerAddress = fmt.Sprintf("%s:%d", host, port)
	})
}

// WithAddress sets the broker address to be used.
// To establish a TLS connection, use WithTLS Option along with this.
// Default values for hostname is "127.0.0.1" and for port is 1883.
func WithAddress(host string, port uint16) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.brokerAddress = fmt.Sprintf("%s:%d", host, port)
	})
}

// KeepAlive will set the amount of time that the client
// should wait before sending a PING request to the broker. This will
// allow the client to know that a connection has not been lost with the
// server.
// Default value is 60 seconds.
// Note: Practically, when KeepAlive >= 10s, the client will check every 5s, if it needs to send a PING.
// In other cases, the client will check every KeepAlive/2.
type KeepAlive time.Duration

func (ka KeepAlive) apply(o *clientOptions) { o.keepAlive = time.Duration(ka) }

// WithKeepAlive will set the amount of time (in seconds) that the client
// should wait before sending a PING request to the broker. This will
// allow the client to know that a connection has not been lost with the
// server.
// Deprecated: Use KeepAlive instead.
func WithKeepAlive(duration time.Duration) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.keepAlive = duration
	})
}

// WithConnectTimeout limits how long the client will wait when trying to open a connection
// to an MQTT server before timing out. A duration of 0 never times out.
// Default 15 seconds.
func WithConnectTimeout(duration time.Duration) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.connectTimeout = duration
	})
}

// WithWriteTimeout limits how long the client will wait when trying to publish, subscribe or unsubscribe on topic
// when a context deadline is not set while calling Publisher.Publish, Subscriber.Subscribe,
// Subscriber.SubscribeMultiple or Unsubscriber.Unsubscribe.
func WithWriteTimeout(duration time.Duration) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.writeTimeout = duration
	})
}

// WithMaxReconnectInterval sets the maximum time that will be waited between reconnection attempts.
// when connection is lost
func WithMaxReconnectInterval(duration time.Duration) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.maxReconnectInterval = duration
	})
}

// WithGracefulShutdownPeriod sets the limit that is allowed for existing work to be completed.
func WithGracefulShutdownPeriod(duration time.Duration) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.gracefulShutdownPeriod = duration
	})
}

// WithPersistence allows to configure the store to be used by broker
// Default persistence is in-memory persistence with mqtt.MemoryStore
func WithPersistence(store Store) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.store = store
	})
}

// WithCustomEncoder allows to transform objects into the desired message bytes.
func WithCustomEncoder(encoderFunc EncoderFunc) ClientOption { return encoderFunc }

// WithCustomDecoder allows to decode message bytes into the desired object.
func WithCustomDecoder(decoderFunc DecoderFunc) ClientOption { return decoderFunc }

// WithUseBase64Decoder configures a json decoder with a base64.StdEncoding wrapped decoder
// which decodes base64 encoded message bytes into the passed object.
func WithUseBase64Decoder() ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.newDecoder = base64JsonDecoder
	})
}

// WithPoolSize configures the client to use a connection pool with the specified size.
func WithPoolSize(size int) ClientOption {
	return optionFunc(func(o *clientOptions) {
		if size > 0 {
			o.poolSize = size
			o.poolEnabled = size > 1
		}
	})
}

// WithExponentialStartOptions configures the client to use ExponentialStartStrategy
// along with the passed StartOption(s) when using the Client.Run method.
func WithExponentialStartOptions(options ...StartOption) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.startOptions = defaultStartOptions()
		for _, opt := range options {
			opt(o.startOptions)
		}
	})
}

func WithAckTimeout(duration time.Duration) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.ackTimeout = duration
	})
}

// ConnectRetryInterval allows to configure the interval between connection retries.
// Default value is 10 seconds.
type ConnectRetryInterval time.Duration

func (i ConnectRetryInterval) apply(o *clientOptions) {
	o.connectRetryPolicy.enabled = true
	o.connectRetryPolicy.interval = time.Duration(i)
}

// SharedSubscriptionPredicate allows to configure the predicate function that determines
// whether a topic is a shared subscription topic.
type SharedSubscriptionPredicate func(topic string) bool

func (ssp SharedSubscriptionPredicate) apply(o *clientOptions) { o.sharedSubscriptionPredicate = ssp }

// UseMultiConnectionMode allows to configure the client to use multiple connections when available.
//
// This is useful when working with shared subscriptions and multiple connections can be created
// to subscribe on the same application.
var UseMultiConnectionMode = multiConnMode{}

// ResumeSubscriptions allows resuming of stored (un)subscribe messages when connecting
// but not reconnecting if CleanSession is false. Otherwise, these messages are discarded.
var ResumeSubscriptions = resumeSubscriptions{}

type multiConnMode struct{}

func (mcm multiConnMode) apply(o *clientOptions) { o.multiConnectionMode = true }

type resumeSubscriptions struct{}

func (rs resumeSubscriptions) apply(o *clientOptions) { o.resumeSubscriptions = true }

type clientOptions struct {
	username, clientID, password,
	brokerAddress string
	resolver          Resolver
	credentialFetcher CredentialFetcher

	tlsConfig *tls.Config

	autoReconnect, maintainOrder, cleanSession,
	multiConnectionMode, resumeSubscriptions,
	poolEnabled bool

	connectTimeout, writeTimeout, keepAlive,
	maxReconnectInterval, gracefulShutdownPeriod,
	credentialFetchTimeout, ackTimeout time.Duration

	connectRetryPolicy connectRetryPolicy
	startOptions       *startOptions

	onConnectHandler            OnConnectHandler
	onConnectionLostHandler     OnConnectionLostHandler
	onReconnectHandler          OnReconnectHandler
	sharedSubscriptionPredicate SharedSubscriptionPredicate
	logger                      Logger
	infoEmitterCfg              *ClientInfoEmitterConfig
	pahoLogLevel                mqtt.LogLevel

	newEncoder EncoderFunc
	newDecoder DecoderFunc
	store      Store

	poolSize int
}

type connectRetryPolicy struct {
	enabled  bool
	interval time.Duration
}

type optionFunc func(*clientOptions)

func (f optionFunc) apply(o *clientOptions) { f(o) }

func defaultClientOptions() *clientOptions {
	return &clientOptions{
		autoReconnect:               true,
		maintainOrder:               true,
		connectTimeout:              15 * time.Second,
		writeTimeout:                10 * time.Second,
		maxReconnectInterval:        5 * time.Minute,
		gracefulShutdownPeriod:      30 * time.Second,
		keepAlive:                   60 * time.Second,
		credentialFetchTimeout:      10 * time.Second,
		connectRetryPolicy:          connectRetryPolicy{interval: 10 * time.Second},
		newEncoder:                  DefaultEncoderFunc,
		newDecoder:                  DefaultDecoderFunc,
		store:                       inMemoryPersistence,
		sharedSubscriptionPredicate: defaultSharedSubscriptionPredicate,
		logger:                      defaultLogger,
		pahoLogLevel:                defaultPahoLogLevel,
	}
}
