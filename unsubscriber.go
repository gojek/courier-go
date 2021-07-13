package courier

import (
	"context"
)

type Unsubscriber interface {
	// Unsubscribe removes any subscription to messages from an MQTT broker
	Unsubscribe(ctx context.Context, topics ...string) error
}

type unsubscribeMiddleware interface {
	// Middleware helps chain Unsubscriber(s).
	Middleware(unsubscriber Unsubscriber) Unsubscriber
}

// UnsubscriberMiddlewareFunc functions are closures that intercept Unsubscriber.Unsubscribe calls.
type UnsubscriberMiddlewareFunc func(Unsubscriber) Unsubscriber

// Middleware allows UnsubscriberMiddlewareFunc to implement the unsubscribeMiddleware interface.
func (usmw UnsubscriberMiddlewareFunc) Middleware(unsubscriber Unsubscriber) Unsubscriber {
	return usmw(unsubscriber)
}

// UnsubscriberFunc defines signature of a Unsubscribe function.
type UnsubscriberFunc func(context.Context, ...string) error

// Unsubscribe implements Unsubscriber interface on UnsubscriberFunc.
func (f UnsubscriberFunc) Unsubscribe(ctx context.Context, topics ...string) error {
	return f(ctx, topics...)
}
