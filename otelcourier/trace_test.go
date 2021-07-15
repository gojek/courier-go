package otelcourier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/oteltest"
	oteltrace "go.opentelemetry.io/otel/trace"

	courier "***REMOVED***"
	"***REMOVED***/metrics"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())

	mwf := NewMiddleware("test-service")

	p := mwf.Publisher().Middleware(courier.PublisherFunc(func(ctx context.Context, topic string, qos courier.QOSLevel, retained bool, message interface{}) error {
		span := oteltrace.SpanFromContext(ctx)
		_, ok := span.(*oteltest.Span)
		assert.True(t, ok)
		return nil
	}))

	err := p.Publish(context.Background(), "test-topic", courier.QOSOne, false, "hello-world")
	assert.NoError(t, err)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	tp := oteltest.NewTracerProvider()
	m := NewMiddleware("test-service", WithTracerProvider(tp))

	p := m.Publisher().Middleware(courier.PublisherFunc(func(ctx context.Context, topic string, qos courier.QOSLevel, retained bool, message interface{}) error {
		span := oteltrace.SpanFromContext(ctx)
		_, ok := span.(*oteltest.Span)
		assert.True(t, ok)
		return nil
	}))

	err := p.Publish(context.Background(), "test-topic", courier.QOSOne, false, "hello-world")
	assert.NoError(t, err)
}

func TestInstrumentClient(t *testing.T) {
	tp := oteltest.NewTracerProvider()
	m := NewMiddleware("test-service", WithTracerProvider(tp))
	c, err := courier.NewClient(courier.WithCustomMetrics(metrics.NewPrometheus()))
	assert.NoError(t, err)
	InstrumentClient(c, m)
}
