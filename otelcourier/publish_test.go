package otelcourier

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

type testTextMapCarrier struct {
	payload string
	headers map[string]string
}

func (t *testTextMapCarrier) Get(key string) string {
	return t.headers[key]
}

func (t *testTextMapCarrier) Set(key string, value string) {
	t.headers[key] = value
}

func (t *testTextMapCarrier) Keys() []string {
	var keys []string
	for k := range t.headers {
		keys = append(keys, k)
	}
	return keys
}

func TestPublishTraceSpan(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	mtmc := newMockTextMapCarrier(t, "hello-world")

	mwf := NewTracer("test-service", WithTracerProvider(tp),
		WithTextMapPropagator(propagation.NewCompositeTextMapPropagator(&propagation.TraceContext{})))
	uErr := errors.New("error_from_upstream")

	p := mwf.publisher(courier.PublisherFunc(func(ctx context.Context, topic string, message interface{}, opts ...courier.Option) error {
		return uErr
	}))

	traceParentRegex := regexp.MustCompile(`^\d{2}-\w{32}-\w{16}-\d{2}$`)
	mtmc.On("Set", "traceparent", mock.MatchedBy(func(in string) bool {
		return traceParentRegex.MatchString(in)
	}))

	err := p.Publish(context.Background(), "test-topic", mtmc, courier.QOSOne, courier.Retained(false))

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

	mtmc.AssertExpectations(t)
}

func TestPublishSpanNotInstrumented(t *testing.T) {
	p := courier.PublisherFunc(func(ctx context.Context, _ string, _ interface{}, _ ...courier.Option) error {
		span := oteltrace.SpanFromContext(ctx)
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		return nil
	})

	err := p.Publish(context.Background(), "test-topic", "hello-world", courier.QOSOne)
	assert.NoError(t, err)
}

func newMockTextMapCarrier(t *testing.T, payload string) *mockTextMapCarrier {
	m := &mockTextMapCarrier{Payload: payload}
	m.Test(t)
	return m
}

type mockTextMapCarrier struct {
	Payload string `json:"payload"`
	mock.Mock
}

func (m *mockTextMapCarrier) Get(key string) string {
	return m.Called(key).String(0)
}

func (m *mockTextMapCarrier) Set(key string, value string) {
	m.Called(key, value)
}

func (m *mockTextMapCarrier) Keys() []string {
	if v, ok := m.Called().Get(0).([]string); ok {
		return v
	}

	return nil
}
