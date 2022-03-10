package courier

import (
	"context"
)

// Publisher defines behaviour of an MQTT publisher that can send messages.
type Publisher interface {
	// Publish allows to publish messages to an MQTT broker
	Publish(ctx context.Context, topic string, message interface{}, options ...Option) error
}

// PublisherFunc defines signature of a Publish function.
type PublisherFunc func(context.Context, string, interface{}, ...Option) error

// Publish implements Publisher interface on PublisherFunc.
func (f PublisherFunc) Publish(
	ctx context.Context,
	topic string,
	message interface{},
	opts ...Option,
) error {
	return f(ctx, topic, message, opts...)
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
