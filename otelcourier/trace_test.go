package otelcourier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"

	courier "github.com/gojek/courier-go"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)
	otel.SetTracerProvider(tp)

	mwf := NewTracer("test-service")

	p := mwf.PublisherMiddleware(courier.PublisherFunc(func(ctx context.Context, topic string, message interface{}, opts ...courier.Option) error {
		span := oteltrace.SpanFromContext(ctx)
		_, ok := span.(trace.ReadWriteSpan)
		assert.True(t, ok)
		return nil
	}))

	err := p.Publish(context.Background(), "test-topic", "hello-world", courier.QOSOne)
	assert.NoError(t, err)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	m := NewTracer("test-service", WithTracerProvider(tp))

	p := m.PublisherMiddleware(courier.PublisherFunc(func(ctx context.Context, topic string, message interface{}, opts ...courier.Option) error {
		span := oteltrace.SpanFromContext(ctx)
		_, ok := span.(trace.ReadWriteSpan)
		assert.True(t, ok)
		return nil
	}))

	err := p.Publish(context.Background(), "test-topic", "hello-world", courier.QOSOne)
	assert.NoError(t, err)
}

func TestInstrumentClient(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	tr := NewTracer("test-service", WithTracerProvider(tp))
	c, err := courier.NewClient(courier.WithAddress("localhost", 1883))
	assert.NoError(t, err)
	tr.ApplyTraceMiddlewares(c)
}
