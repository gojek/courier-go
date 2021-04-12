package courier

import (
	"context"
)

type Publisher interface {
	// Publish allows to publish messages to an MQTT broker
	Publish(topic string, qos QOSLevel, retained bool, message interface{}) error
}

type Subscriber interface {
	// Subscribe allows to subscribe to messages from an MQTT broker
	Subscribe(topic string, qos QOSLevel, callback MessageHandler) error

	// SubscribeMultiple allows to subscribe to messages on multiple topics from an MQTT broker
	SubscribeMultiple(topicsWithQos map[string]QOSLevel, callback MessageHandler) error

	// Unsubscribe removes any subscription to messages from an MQTT broker
	Unsubscribe(topics ...string) error
}

type ConnectionInformer interface {
	// IsConnected checks whether the client is connected to the broker
	IsConnected() bool
}

type PubSub interface {
	Publisher
	Subscriber
	ConnectionInformer
}

type ConnectionManager interface {
	// Start will connect to a broker and will block until the Context passed has been cancelled,
	// cancelling the Context will disconnect the client
	Start(ctx context.Context) error
}
