package otelcourier

import (
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type tracePath uint

func (tp tracePath) match(o tracePath) bool { return tp&o != 0 }

const (
	tracePublisher tracePath = 1 << iota
	traceSubscriber
	traceUnsubsriber
	traceCallback
)

type traceOptions struct {
	tracerProvider oteltrace.TracerProvider
	tracePaths     tracePath
}

// Option helps configure trace options.
type Option func(*traceOptions)

func defaultOptions() *traceOptions {
	return &traceOptions{
		tracerProvider: otel.GetTracerProvider(),
		tracePaths:     tracePublisher + traceSubscriber + traceUnsubsriber + traceCallback,
	}
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return func(opts *traceOptions) {
		opts.tracerProvider = provider
	}
}

// DisableCallbackTracing disables implicit tracing on subscription callbacks.
func DisableCallbackTracing(opts *traceOptions) {
	opts.tracePaths &^= traceCallback
}

// DisablePublisherTracing disables courier.Publisher tracing.
func DisablePublisherTracing(opts *traceOptions) {
	opts.tracePaths &^= tracePublisher
}

// DisableSubscriberTracing disables courier.Subscriber tracing.
func DisableSubscriberTracing(opts *traceOptions) {
	opts.tracePaths &^= traceSubscriber
}

// DisableUnsubscriberTracing disables courier.Unsubscriber tracing.
func DisableUnsubscriberTracing(opts *traceOptions) {
	opts.tracePaths &^= traceUnsubsriber
}
