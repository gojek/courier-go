package courier

import (
	"context"
)

// Subscriber defines behaviour of an MQTT subscriber that can create subscriptions.
type Subscriber interface {
	// Subscribe allows to subscribe to messages from an MQTT broker
	Subscribe(ctx context.Context, topic string, callback MessageHandler, opts ...Option) error

	// SubscribeMultiple allows to subscribe to messages on multiple topics from an MQTT broker
	SubscribeMultiple(ctx context.Context, topicsWithQos map[string]QOSLevel, callback MessageHandler) error
}

type subscribeMiddleware interface {
	// Middleware helps chain Subscriber(s).
	Middleware(subscriber Subscriber) Subscriber
}

// SubscriberMiddlewareFunc functions are closures that intercept Subscriber.Subscribe calls.
type SubscriberMiddlewareFunc func(Subscriber) Subscriber

// Middleware allows SubscriberMiddlewareFunc to implement the subscribeMiddleware interface.
func (smw SubscriberMiddlewareFunc) Middleware(subscriber Subscriber) Subscriber {
	return smw(subscriber)
}

// SubscriberFuncs defines signature of a Subscribe function.
type SubscriberFuncs struct {
	subscribe         func(context.Context, string, MessageHandler, ...Option) error
	subscribeMultiple func(context.Context, map[string]QOSLevel, MessageHandler) error
}

// NewSubscriberFuncs is a helper function to create SubscriberFuncs
func NewSubscriberFuncs(
	subscribeFunc func(context.Context, string, MessageHandler, ...Option) error,
	subscribeMultipleFunc func(context.Context, map[string]QOSLevel, MessageHandler) error,
) SubscriberFuncs {
	return SubscriberFuncs{subscribe: subscribeFunc, subscribeMultiple: subscribeMultipleFunc}
}

// Subscribe implements Subscriber interface on SubscriberFuncs.
func (s SubscriberFuncs) Subscribe(ctx context.Context, topic string, callback MessageHandler, opts ...Option) error {
	return s.subscribe(ctx, topic, callback, opts...)
}

// SubscribeMultiple implements Subscriber interface on SubscriberFuncs.
func (s SubscriberFuncs) SubscribeMultiple(
	ctx context.Context,
	topicsWithQos map[string]QOSLevel,
	callback MessageHandler,
) error {
	return s.subscribeMultiple(ctx, topicsWithQos, callback)
}
