package otelcourier

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	courier "***REMOVED***"
)

const (
	tracerName = "***REMOVED***/otelcourier"
)

type Middleware struct {
	service                string
	tracer                 trace.Tracer
	disableCallbackTracing bool
}

func NewMiddleware(service string, opts ...Option) *Middleware {
	mo := middlewareOptions{tracerProvider: otel.GetTracerProvider()}
	for _, opt := range opts {
		opt(&mo)
	}
	tracer := mo.tracerProvider.Tracer(
		tracerName,
		trace.WithInstrumentationVersion("semver:"+courier.Version()),
	)
	return &Middleware{
		service:                service,
		tracer:                 tracer,
		disableCallbackTracing: mo.disableCallbackTracing,
	}
}

// InstrumentClient will instrument all the operations of a courier.Client instance
func InstrumentClient(c *courier.Client, m *Middleware) {
	c.UsePublisherMiddleware(m.Publisher())
	c.UseSubscriberMiddleware(m.Subscriber())
	c.UseUnsubscriberMiddleware(m.Unsubscriber())
}
