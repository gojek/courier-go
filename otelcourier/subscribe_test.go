package otelcourier

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/semconv"
	oteltrace "go.opentelemetry.io/otel/trace"

	courier "***REMOVED***"
)

func TestSubscribeTraceSpan(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	mwf := NewMiddleware("test-service", WithTracerProvider(tp))
	uErr := errors.New("error_from_upstream")

	u := mwf.Subscriber().Middleware(courier.NewSubscriberFuncs(
		func(_ context.Context, _ string, _ courier.QOSLevel, _ courier.MessageHandler) error {
			return uErr
		},
		func(_ context.Context, _ map[string]courier.QOSLevel, _ courier.MessageHandler) error {
			return nil
		},
	))

	err := u.Subscribe(context.Background(), "test-topic", courier.QOSOne, func(_ context.Context, _ courier.PubSub, _ courier.Decoder) {})
	assert.EqualError(t, err, uErr.Error())

	spans := sr.Completed()
	require.Len(t, spans, 1)

	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		attrs := span.Attributes()
		assert.Equal(t, subscribeSpanName, span.Name())
		assert.Equal(t, oteltrace.SpanKindClient, span.SpanKind())
		assert.Equal(t, semconv.ServiceNameKey.String("test-service").Value, attrs[semconv.ServiceNameKey])
		assert.Equal(t, MQTTTopic.String("test-topic").Value, attrs[MQTTTopic])
		assert.Equal(t, MQTTQoS.Int(1).Value, attrs[MQTTQoS])
	})

	t.Run("Events", func(t *testing.T) {
		assert.Equal(t, semconv.ExceptionMessageKey.String(uErr.Error()).Value, span.Events()[0].Attributes[semconv.ExceptionMessageKey])
		assert.Equal(t, codes.Error, span.StatusCode())
		assert.Equal(t, subscribeErrMessage, span.StatusMessage())
	})
}

func TestSubscribeMultipleTraceSpan(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	mwf := NewMiddleware("test-service", WithTracerProvider(tp))
	uErr := errors.New("error_from_upstream")

	u := mwf.Subscriber().Middleware(courier.NewSubscriberFuncs(
		func(_ context.Context, _ string, _ courier.QOSLevel, _ courier.MessageHandler) error {
			return nil
		},
		func(_ context.Context, _ map[string]courier.QOSLevel, _ courier.MessageHandler) error {
			return uErr
		},
	))

	err := u.SubscribeMultiple(context.Background(), map[string]courier.QOSLevel{
		"test-topic-1": courier.QOSOne,
		"test-topic-2": courier.QOSTwo,
	}, func(_ context.Context, _ courier.PubSub, _ courier.Decoder) {})
	assert.EqualError(t, err, uErr.Error())

	spans := sr.Completed()
	require.Len(t, spans, 1)

	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		attrs := span.Attributes()
		assert.Equal(t, subscribeMultipleSpanName, span.Name())
		assert.Equal(t, oteltrace.SpanKindClient, span.SpanKind())
		assert.Equal(t, semconv.ServiceNameKey.String("test-service").Value, attrs[semconv.ServiceNameKey])
		assert.Equal(t, MQTTTopicWithQoS.Array([]string{
			"test-topic-1 | qos[1]",
			"test-topic-2 | qos[2]",
		}).Value, attrs[MQTTTopicWithQoS])
	})

	t.Run("Events", func(t *testing.T) {
		assert.Equal(t, semconv.ExceptionMessageKey.String(uErr.Error()).Value, span.Events()[0].Attributes[semconv.ExceptionMessageKey])
		assert.Equal(t, codes.Error, span.StatusCode())
		assert.Equal(t, subscribeMultipleErrMessage, span.StatusMessage())
	})
}

func TestSubscriberSpanNotInstrumented(t *testing.T) {
	u := courier.UnsubscriberFunc(func(ctx context.Context, _ ...string) error {
		span := oteltrace.SpanFromContext(ctx)
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		return nil
	})

	err := u.Unsubscribe(context.Background(), "test-topic-1", "test-topic-2")
	assert.NoError(t, err)
}

func Test_instrumentCallback(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	m := NewMiddleware("test-service", WithTracerProvider(tp))

	callback := m.instrumentCallback(func(_ context.Context, _ courier.PubSub, _ courier.Decoder) {})

	c, _ := courier.NewClient()
	callback(context.Background(), c, json.NewDecoder(strings.NewReader("test-injection")))

	spans := sr.Completed()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "***REMOVED***/otelcourier.Test_instrumentCallback.func1", span.Name())
}

func Test_instrumentCallbackDisabled(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	m := NewMiddleware("test-service", WithTracerProvider(tp), WithCallbackTracingDisabled())

	callback := m.instrumentCallback(func(_ context.Context, _ courier.PubSub, _ courier.Decoder) {})

	c, _ := courier.NewClient()
	callback(context.Background(), c, json.NewDecoder(strings.NewReader("test-injection")))

	spans := sr.Completed()
	require.Len(t, spans, 0)
}
