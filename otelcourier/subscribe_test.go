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
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	courier "github.com/gojek/courier-go"
)

type traceparent int

const traceparentKey traceparent = 0

func TestSubscribeTraceSpan(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	mwf := NewTracer("test-service", WithTracerProvider(tp))
	uErr := errors.New("error_from_upstream")

	u := mwf.SubscriberMiddleware(courier.NewSubscriberFuncs(
		func(_ context.Context, _ string, _ courier.MessageHandler, _ ...courier.Option) error {
			return uErr
		},
		func(_ context.Context, _ map[string]courier.QOSLevel, _ courier.MessageHandler) error {
			return nil
		},
	))

	err := u.Subscribe(context.Background(), "test-topic",
		func(_ context.Context, _ courier.PubSub, _ *courier.Message) {},
		courier.QOSOne,
	)
	assert.EqualError(t, err, uErr.Error())

	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		assert.Equal(t, subscribeSpanName, span.Name())
		assert.Equal(t, oteltrace.SpanKindClient, span.SpanKind())

		got := attribute.NewSet(span.Attributes()...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ServiceNameKey.String("test-service"),
			MQTTTopic.String("test-topic"),
			MQTTQoS.Int(1),
		}...)

		assert.Equal(t, expected, got)
	})

	t.Run("Events", func(t *testing.T) {
		got := attribute.NewSet(span.Events()[0].Attributes...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ExceptionMessageKey.String(uErr.Error()),
			semconv.ExceptionTypeKey.String(reflect.TypeOf(uErr).String()),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, codes.Error, span.Status().Code)
		assert.Equal(t, subscribeErrMessage, span.Status().Description)
	})
}

func TestSubscribeMultipleTraceSpan(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	mwf := NewTracer("test-service", WithTracerProvider(tp))
	uErr := errors.New("error_from_upstream")

	u := mwf.SubscriberMiddleware(courier.NewSubscriberFuncs(
		func(_ context.Context, _ string, _ courier.MessageHandler, _ ...courier.Option) error {
			return nil
		},
		func(_ context.Context, _ map[string]courier.QOSLevel, _ courier.MessageHandler) error {
			return uErr
		},
	))

	err := u.SubscribeMultiple(context.Background(), map[string]courier.QOSLevel{
		"test-topic-1": courier.QOSOne,
		"test-topic-2": courier.QOSTwo,
	}, func(_ context.Context, _ courier.PubSub, _ *courier.Message) {})
	assert.EqualError(t, err, uErr.Error())

	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		got := attribute.NewSet(span.Attributes()...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ServiceNameKey.String("test-service"),
			MQTTTopicWithQoS.StringSlice([]string{
				"test-topic-1 | qos[1]",
				"test-topic-2 | qos[2]",
			}),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, subscribeMultipleSpanName, span.Name())
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
		assert.Equal(t, subscribeMultipleErrMessage, span.Status().Description)
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
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	extFn := func(ctx context.Context) propagation.TextMapCarrier {
		return ctx.Value(traceparentKey).(propagation.TextMapCarrier)
	}

	m := NewTracer("test-service",
		WithTracerProvider(tp),
		WithTextMapPropagator(propagation.NewCompositeTextMapPropagator(&propagation.TraceContext{})),
		WithTextMapCarrierExtractFunc(extFn),
	)

	callback := m.instrumentCallback(func(_ context.Context, _ courier.PubSub, _ *courier.Message) {})

	c, _ := courier.NewClient()
	callback(context.WithValue(context.Background(), traceparentKey, &propagation.MapCarrier{
		"traceparent": "00-c8e801456e8232f618c49c6f65f101db-1986c136102242cd-01",
	}), c, &courier.Message{})

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "c8e801456e8232f618c49c6f65f101db", span.SpanContext().TraceID().String())
	assert.Equal(t, "github.com/gojek/courier-go/otelcourier.Test_instrumentCallback.func2", span.Name())
}

func Test_instrumentCallbackDisabled(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	m := NewTracer("test-service", WithTracerProvider(tp), DisableCallbackTracing)

	callback := m.instrumentCallback(func(_ context.Context, _ courier.PubSub, _ *courier.Message) {})

	c, _ := courier.NewClient()
	callback(context.Background(), c, &courier.Message{})

	spans := sr.Ended()
	require.Len(t, spans, 0)
}
