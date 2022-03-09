package otelcourier

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	courier "github.com/gojek/courier-go"
)

func TestPublishTraceSpan(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	mwf := NewTracer("test-service", WithTracerProvider(tp))
	uErr := errors.New("error_from_upstream")

	p := mwf.publisher(courier.PublisherFunc(func(ctx context.Context, topic string, qos courier.QOSLevel, retained bool, message interface{}) error {
		return uErr
	}))

	err := p.Publish(context.Background(), "test-topic", courier.QOSOne, false, "hello-world")
	assert.EqualError(t, err, uErr.Error())

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		got := attribute.NewSet(span.Attributes()...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ServiceNameKey.String("test-service"),
			MQTTTopic.String("test-topic"),
			MQTTQoS.Int(1),
			MQTTRetained.Bool(false),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, publishSpanName, span.Name())
		assert.Equal(t, oteltrace.SpanKindProducer, span.SpanKind())
	})

	t.Run("Events", func(t *testing.T) {
		got := attribute.NewSet(span.Events()[0].Attributes...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ExceptionMessageKey.String(uErr.Error()),
			semconv.ExceptionTypeKey.String(reflect.TypeOf(uErr).String()),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, codes.Error, span.Status().Code)
		assert.Equal(t, publishErrMessage, span.Status().Description)
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
