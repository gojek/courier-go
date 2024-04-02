package otelcourier

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/gojek/courier-go"
)

const (
	tracerName = "github.com/gojek/courier-go/otelcourier"
)

// UseMiddleware is an interface that defines the methods to
// apply middlewares to a courier.Client or similar instance.
type UseMiddleware interface {
	UsePublisherMiddleware(mwf ...courier.PublisherMiddlewareFunc)
	UseSubscriberMiddleware(mwf ...courier.SubscriberMiddlewareFunc)
	UseUnsubscriberMiddleware(mwf ...courier.UnsubscriberMiddlewareFunc)
}

// OTel implements tracing & metric abilities using OpenTelemetry SDK.
type OTel struct {
	service            string
	tracePaths         tracePath
	tracer             trace.Tracer
	meter              metric.Meter
	propagator         propagation.TextMapPropagator
	textMapCarrierFunc func(context.Context) propagation.TextMapCarrier
	topicTransformer   TopicAttributeTransformer

	rc   recorder
	tnow func() time.Time
}

// New creates a new OTel with Option(s).
func New(service string, opts ...Option) *OTel {
	to := defaultOptions()

	for _, opt := range opts {
		opt.apply(to)
	}

	vsn := fmt.Sprintf("semver:%s", courier.Version())
	tracer := to.tracerProvider.Tracer(
		tracerName,
		trace.WithInstrumentationVersion(vsn),
	)
	meter := to.meterProvider.Meter(
		tracerName,
		metric.WithInstrumentationVersion(vsn),
	)

	t := &OTel{
		service:            service,
		tracer:             tracer,
		meter:              meter,
		propagator:         to.propagator,
		textMapCarrierFunc: to.textMapCarrierExtractor,
		topicTransformer:   to.topicTransformer,
		tracePaths:         to.tracePaths,
		rc:                 make(recorder),
		tnow:               time.Now,
	}

	t.initRecorders()

	return t
}

// ApplyMiddlewares will instrument all the operations of a UseMiddleware instance
// according to Option(s) used.
func (t *OTel) ApplyMiddlewares(c UseMiddleware) {
	if t.tracePaths.match(tracePublisher) {
		c.UsePublisherMiddleware(t.PublisherMiddleware)
	}

	if t.tracePaths.match(traceSubscriber) {
		c.UseSubscriberMiddleware(t.SubscriberMiddleware)
	}

	if t.tracePaths.match(traceUnsubscriber) {
		c.UseUnsubscriberMiddleware(t.UnsubscriberMiddleware)
	}
}
