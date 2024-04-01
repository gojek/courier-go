package otelcourier

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type tracePath uint

func (tp tracePath) match(o tracePath) bool { return tp&o != 0 }

const (
	tracePublisher tracePath = 1 << iota
	traceSubscriber
	traceUnsubscriber
	traceCallback
)

type traceOptions struct {
	tracerProvider          oteltrace.TracerProvider
	meterProvider           metric.MeterProvider
	propagator              propagation.TextMapPropagator
	textMapCarrierExtractor func(context.Context) propagation.TextMapCarrier
	tracePaths              tracePath
}

// Option helps configure trace options.
type Option func(*traceOptions)

func defaultOptions() *traceOptions {
	return &traceOptions{
		tracerProvider: otel.GetTracerProvider(),
		meterProvider:  otel.GetMeterProvider(),
		propagator:     otel.GetTextMapPropagator(),
		tracePaths:     tracePublisher + traceSubscriber + traceUnsubscriber + traceCallback,
	}
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return func(opts *traceOptions) { opts.tracerProvider = provider }
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return func(opts *traceOptions) { opts.meterProvider = provider }
}

// WithTextMapPropagator specifies the propagator to use for extracting/injecting key-value texts.
// If none is specified, the global provider is used.
func WithTextMapPropagator(propagator propagation.TextMapPropagator) Option {
	return func(opts *traceOptions) { opts.propagator = propagator }
}

// WithTextMapCarrierExtractFunc is used to specify the function which should be used to
// extract propagation.TextMapCarrier from the ongoing context.Context.
func WithTextMapCarrierExtractFunc(fn func(context.Context) propagation.TextMapCarrier) Option {
	return func(opts *traceOptions) { opts.textMapCarrierExtractor = fn }
}

// DisableCallbackTracing disables implicit tracing on subscription callbacks.
func DisableCallbackTracing(opts *traceOptions) { opts.tracePaths &^= traceCallback }

// DisablePublisherTracing disables courier.Publisher tracing.
func DisablePublisherTracing(opts *traceOptions) { opts.tracePaths &^= tracePublisher }

// DisableSubscriberTracing disables courier.Subscriber tracing.
func DisableSubscriberTracing(opts *traceOptions) { opts.tracePaths &^= traceSubscriber }

// DisableUnsubscriberTracing disables courier.Unsubscriber tracing.
func DisableUnsubscriberTracing(opts *traceOptions) { opts.tracePaths &^= traceUnsubscriber }
