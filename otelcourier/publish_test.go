package otelcourier

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/semconv"
	oteltrace "go.opentelemetry.io/otel/trace"

	courier "***REMOVED***"
)

func TestPublishTraceSpan(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	mwf := NewMiddleware("test-service", WithTracerProvider(tp))
	uErr := errors.New("error_from_upstream")

	p := mwf.Publisher().Middleware(courier.PublisherFunc(func(ctx context.Context, topic string, qos courier.QOSLevel, retained bool, message interface{}) error {
		return uErr
	}))

	err := p.Publish(context.Background(), "test-topic", courier.QOSOne, false, "hello-world")
	assert.EqualError(t, err, uErr.Error())

	spans := sr.Completed()
	require.Len(t, spans, 1)
	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		attrs := span.Attributes()
		assert.Equal(t, publishSpanName, span.Name())
		assert.Equal(t, oteltrace.SpanKindProducer, span.SpanKind())
		assert.Equal(t, semconv.ServiceNameKey.String("test-service").Value, attrs[semconv.ServiceNameKey])
		assert.Equal(t, MQTTTopic.String("test-topic").Value, attrs[MQTTTopic])
		assert.Equal(t, MQTTQoS.Int(1).Value, attrs[MQTTQoS])
		assert.Equal(t, MQTTRetained.Bool(false).Value, attrs[MQTTRetained])
	})

	t.Run("Events", func(t *testing.T) {
		assert.Equal(t, semconv.ExceptionMessageKey.String(uErr.Error()).Value, span.Events()[0].Attributes[semconv.ExceptionMessageKey])
		assert.Equal(t, codes.Error, span.StatusCode())
		assert.Equal(t, publishErrMessage, span.StatusMessage())
	})
}

func TestPublishSpanNotInstrumented(t *testing.T) {
	p := courier.PublisherFunc(func(ctx context.Context, _ string, _ courier.QOSLevel, _ bool, _ interface{}) error {
		span := oteltrace.SpanFromContext(ctx)
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		return nil
	})

	err := p.Publish(context.Background(), "test-topic", courier.QOSOne, false, "hello-world")
	assert.NoError(t, err)
}
