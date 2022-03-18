package otelcourier

import (
	"go.opentelemetry.io/otel/trace"

	courier "github.com/gojekfarm/courier-go"
)

const (
	tracerName = "github.com/gojekfarm/courier-go/otelcourier"
)

// Tracer implements tracing abilities using OpenTelemetry SDK.
type Tracer struct {
	service    string
	tracer     trace.Tracer
	tracePaths tracePath
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
		service:    service,
		tracer:     tracer,
		tracePaths: to.tracePaths,
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

	if t.tracePaths.match(traceUnsubsriber) {
		c.UseUnsubscriberMiddleware(t.unsubscriber)
	}
}
