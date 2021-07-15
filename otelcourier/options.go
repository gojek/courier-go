package otelcourier

import (
	oteltrace "go.opentelemetry.io/otel/trace"
)

type middlewareOptions struct {
	tracerProvider         oteltrace.TracerProvider
	disableCallbackTracing bool
}

// Option helps configure middleware options.
type Option func(*middlewareOptions)

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return func(opts *middlewareOptions) {
		opts.tracerProvider = provider
	}
}

// WithCallbackTracingDisabled disables implicit tracing on subscription callbacks
func WithCallbackTracingDisabled() Option {
	return func(opts *middlewareOptions) {
		opts.disableCallbackTracing = true
	}
}
