package courier

import (
	"context"

	"github.com/gojek/courier-go/internal/driver"
)

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

type userPropertiesKey struct{}

// WithUserProperties returns a context carrying MQTT v5 user properties to be sent with Publish,
// they are ignored when the client uses MQTT v3.1.1.
func WithUserProperties(ctx context.Context, props map[string]string) context.Context {
	return context.WithValue(ctx, userPropertiesKey{}, props)
}

// UserPropertiesFromContext returns the user properties set with WithUserProperties, nil if none.
func UserPropertiesFromContext(ctx context.Context) map[string]string {
	props, _ := ctx.Value(userPropertiesKey{}).(map[string]string)

	return props
}

type (
	contentTypeKey     struct{}
	responseTopicKey   struct{}
	correlationDataKey struct{}
)

// WithContentType returns a context carrying the MQTT v5 Content Type to be sent with Publish,
// it is ignored when the client uses MQTT v3.1.1.
func WithContentType(ctx context.Context, contentType string) context.Context {
	return context.WithValue(ctx, contentTypeKey{}, contentType)
}

// WithResponseTopic returns a context carrying the MQTT v5 Response Topic to be sent with Publish,
// it is ignored when the client uses MQTT v3.1.1.
func WithResponseTopic(ctx context.Context, topic string) context.Context {
	return context.WithValue(ctx, responseTopicKey{}, topic)
}

// WithCorrelationData returns a context carrying the MQTT v5 Correlation Data to be sent with Publish,
// it is ignored when the client uses MQTT v3.1.1.
func WithCorrelationData(ctx context.Context, data []byte) context.Context {
	return context.WithValue(ctx, correlationDataKey{}, data)
}

func publishPropsFromContext(ctx context.Context) *driver.PublishProps {
	props := &driver.PublishProps{UserProperties: UserPropertiesFromContext(ctx)}

	props.ContentType, _ = ctx.Value(contentTypeKey{}).(string)
	props.ResponseTopic, _ = ctx.Value(responseTopicKey{}).(string)
	props.CorrelationData, _ = ctx.Value(correlationDataKey{}).([]byte)

	return props
}
