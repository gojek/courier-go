package courier

import (
	"context"
)

// OnConnectHandler is a callback that is called when the client
// state changes from disconnected to connected. Both
// at initial connection and on reconnection
type OnConnectHandler func(PubSub)

// OnConnectionLostHandler is a callback type which can be set to be
// executed upon an unintended disconnection from the MQTT broker.
// Disconnects caused by calling Disconnect or ForceDisconnect will
// not cause an WithOnConnectionLost callback to execute.
type OnConnectionLostHandler func(error)

// OnReconnectHandler is invoked prior to reconnecting after
// the initial connection is lost
type OnReconnectHandler func(PubSub)

// MessageHandler is the type that all callbacks being passed
// to Subscriber must satisfy.
type MessageHandler func(context.Context, PubSub, *Message)

// MQTTVersion is the MQTT protocol version used to connect to the broker.
type MQTTVersion uint8

const (
	// MQTTv311 is MQTT v3.1.1, the default.
	MQTTv311 MQTTVersion = 4 // matches the protocol level in the CONNECT packet
	// MQTTv5 is MQTT v5.0.
	MQTTv5 MQTTVersion = 5
)
