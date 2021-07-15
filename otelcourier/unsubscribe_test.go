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

func TestUnsubscriberTraceSpan(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	mwf := NewMiddleware("test-service", WithTracerProvider(tp))
	uErr := errors.New("error_from_upstream")

	u := mwf.Unsubscriber().Middleware(courier.UnsubscriberFunc(func(ctx context.Context, topics ...string) error {
		return uErr
	}))

	err := u.Unsubscribe(context.Background(), "test-topic-1", "test-topic-2")
	assert.EqualError(t, err, uErr.Error())

	spans := sr.Completed()
	require.Len(t, spans, 1)

	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		attrs := span.Attributes()
		assert.Equal(t, unsubscribeSpanName, span.Name())
		assert.Equal(t, oteltrace.SpanKindClient, span.SpanKind())
		assert.Equal(t, semconv.ServiceNameKey.String("test-service").Value, attrs[semconv.ServiceNameKey])
		assert.Equal(t, MQTTTopic.Array([]string{"test-topic-1", "test-topic-2"}).Value, attrs[MQTTTopic])
	})

	t.Run("Events", func(t *testing.T) {
		assert.Equal(t, semconv.ExceptionMessageKey.String(uErr.Error()).Value, span.Events()[0].Attributes[semconv.ExceptionMessageKey])
		assert.Equal(t, codes.Error, span.StatusCode())
		assert.Equal(t, unsubscribeErrMessage, span.StatusMessage())
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
