package courier

import (
	"context"
)

type Publisher interface {
	// Publish allows to publish messages to an MQTT broker
	Publish(ctx context.Context, topic string, qos QOSLevel, retained bool, message interface{}) error
}

// PublisherFunc defines signature of a Publish function.
type PublisherFunc func(context.Context, string, QOSLevel, bool, interface{}) error

// Publish implements Publisher interface on PublisherFunc.
func (f PublisherFunc) Publish(ctx context.Context, topic string, qos QOSLevel, retained bool, message interface{}) error {
	return f(ctx, topic, qos, retained, message)
}

type publishMiddleware interface {
	// Middleware helps chain Publisher(s).
	Middleware(publisher Publisher) Publisher
}

// PublisherMiddlewareFunc functions are closures that intercept Publisher.Publish calls.
type PublisherMiddlewareFunc func(Publisher) Publisher

// Middleware allows PublisherMiddlewareFunc to implement the publishMiddleware interface.
func (pmw PublisherMiddlewareFunc) Middleware(publisher Publisher) Publisher {
	return pmw(publisher)
}
