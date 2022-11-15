package otelcourier

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	courier "github.com/gojek/courier-go"
)

const (
	tracerName = "github.com/gojek/courier-go/otelcourier"
)

// Tracer implements tracing abilities using OpenTelemetry SDK.
type Tracer struct {
	service            string
	tracePaths         tracePath
	tracer             trace.Tracer
	propagator         propagation.TextMapPropagator
	textMapCarrierFunc func(context.Context) propagation.TextMapCarrier
}

// NewTracer creates a new Tracer with Option(s).
func NewTracer(service string, opts ...Option) *Tracer {
	to := defaultOptions()

	for _, opt := range opts {
		opt(to)
	}

	tracer := to.tracerProvider.Tracer(
		tracerName,
		trace.WithInstrumentationVersion("semver:"+courier.Version()),
	)

	return &Tracer{
		service:            service,
		tracer:             tracer,
		propagator:         to.propagator,
		textMapCarrierFunc: to.textMapCarrierExtractor,
		tracePaths:         to.tracePaths,
	}
}

// ApplyTraceMiddlewares will instrument all the operations of a courier.Client instance
func (t *Tracer) ApplyTraceMiddlewares(c *courier.Client) {
	if t.tracePaths.match(tracePublisher) {
		c.UsePublisherMiddleware(t.publisher)
	}

	if t.tracePaths.match(traceSubscriber) {
		c.UseSubscriberMiddleware(t.subscriber)
	}

	if t.tracePaths.match(traceUnsubscriber) {
		c.UseUnsubscriberMiddleware(t.unsubscriber)
	}
}
