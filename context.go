package courier

import "context"

type clientIDCallbackKey struct{}
type clientIDSubscribedKey struct{}

// ClientIDCallback is a function that receives the client ID used for an operation.
type ClientIDCallback func(clientID string)

// WithClientIDCallback returns a context with the callback that will receive
// the client ID of the underlying MQTT client used for an operation.
func WithClientIDCallback(ctx context.Context, cb ClientIDCallback) context.Context {
	return context.WithValue(ctx, clientIDCallbackKey{}, cb)
}

func invokeClientIDCallback(ctx context.Context, clientID string) {
	if cb, ok := ctx.Value(clientIDCallbackKey{}).(ClientIDCallback); ok && cb != nil {
		cb(clientID)
	}
}

func withClientID(ctx context.Context, clientID string) context.Context {
	return context.WithValue(ctx, clientIDSubscribedKey{}, clientID)
}

// ClientIDFromContext returns the client ID from the context.
// This is available in subscribe callbacks to identify which MQTT connection received the message.
func ClientIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(clientIDSubscribedKey{}).(string); ok {
		return id
	}

	return ""
}
