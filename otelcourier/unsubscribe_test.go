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

	courier "github.com/gojekfarm/courier-go"
)

func TestUnsubscriberTraceSpan(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	mwf := NewTracer("test-service", WithTracerProvider(tp))
	uErr := errors.New("error_from_upstream")

	u := mwf.unsubscriber(courier.UnsubscriberFunc(func(ctx context.Context, topics ...string) error {
		return uErr
	}))

	err := u.Unsubscribe(context.Background(), "test-topic-1", "test-topic-2")
	assert.EqualError(t, err, uErr.Error())

	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		got := attribute.NewSet(span.Attributes()...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ServiceNameKey.String("test-service"),
			MQTTTopic.StringSlice([]string{"test-topic-1", "test-topic-2"}),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, unsubscribeSpanName, span.Name())
		assert.Equal(t, oteltrace.SpanKindClient, span.SpanKind())
	})

	t.Run("Events", func(t *testing.T) {
		got := attribute.NewSet(span.Events()[0].Attributes...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ExceptionMessageKey.String(uErr.Error()),
			semconv.ExceptionTypeKey.String(reflect.TypeOf(uErr).String()),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, codes.Error, span.Status().Code)
		assert.Equal(t, unsubscribeErrMessage, span.Status().Description)
	})
}

func TestUnsubscriberSpanNotInstrumented(t *testing.T) {
	u := courier.UnsubscriberFunc(func(ctx context.Context, _ ...string) error {
		span := oteltrace.SpanFromContext(ctx)
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		return nil
	})

	err := u.Unsubscribe(context.Background(), "test-topic-1", "test-topic-2")
	assert.NoError(t, err)
}
