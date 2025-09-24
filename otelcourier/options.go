package otelcourier

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Option helps configure trace options.
type Option interface{ apply(*options) }

// TopicAttributeTransformer helps transform topic before making an attribute for it.
// It is used in metric recording only. Traces use the original topic.
type TopicAttributeTransformer func(context.Context, string) string

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return optFn(func(opts *options) { opts.tracerProvider = provider })
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optFn(func(opts *options) { opts.meterProvider = provider })
}

// WithTextMapPropagator specifies the propagator to use for extracting/injecting key-value texts.
// If none is specified, the global provider is used.
func WithTextMapPropagator(propagator propagation.TextMapPropagator) Option {
	return optFn(func(opts *options) { opts.propagator = propagator })
}

// WithTextMapCarrierExtractFunc is used to specify the function which should be used to
// extract propagation.TextMapCarrier from the ongoing context.Context.
func WithTextMapCarrierExtractFunc(fn func(context.Context) propagation.TextMapCarrier) Option {
	return optFn(func(opts *options) { opts.textMapCarrierExtractor = fn })
}

// WithInfoHandlerFrom is used to specify the handler which should be used to
// extract client information from the courier.Client instance.
func WithInfoHandlerFrom(c interface{ InfoHandler() http.Handler }) Option {
	return optFn(func(opts *options) { opts.infoHandler = c.InfoHandler() })
}

// WithAttributes adds custom attributes to all spans and metrics created.
func WithAttributes(attrs ...attribute.KeyValue) Option {
	return optFn(func(opts *options) { opts.attributes = attrs })
}

// DisableCallbackTracing disables implicit tracing on subscription callbacks.
var DisableCallbackTracing = &disableTracePathOpt{traceCallback}

// DisablePublisherTracing disables courier.Publisher tracing.
var DisablePublisherTracing = &disableTracePathOpt{tracePublisher}

// DisableSubscriberTracing disables courier.Subscriber tracing.
var DisableSubscriberTracing = &disableTracePathOpt{traceSubscriber}

// DisableUnsubscriberTracing disables courier.Unsubscriber tracing.
var DisableUnsubscriberTracing = &disableTracePathOpt{traceUnsubscriber}

// DefaultTopicAttributeTransformer is the default transformer for topic attribute.
func DefaultTopicAttributeTransformer(_ context.Context, topic string) string { return topic }

// BucketBoundaries helps override default histogram bucket boundaries for metrics.
type BucketBoundaries struct {
	Publisher, Subscriber, Unsubscriber, Callback []float64
}

/// private types and functions

type optFn func(*options)

func (fn optFn) apply(opts *options) { fn(opts) }

func defaultOptions() *options {
	return &options{
		tracerProvider:      otel.GetTracerProvider(),
		meterProvider:       otel.GetMeterProvider(),
		propagator:          otel.GetTextMapPropagator(),
		tracePaths:          tracePublisher + traceSubscriber + traceUnsubscriber + traceCallback,
		topicTransformer:    DefaultTopicAttributeTransformer,
		histogramBoundaries: defBucketBoundaries,
	}
}

const (
	tracePublisher tracePath = 1 << iota
	traceSubscriber
	traceUnsubscriber
	traceCallback
)

type options struct {
	tracerProvider          oteltrace.TracerProvider
	meterProvider           metric.MeterProvider
	propagator              propagation.TextMapPropagator
	textMapCarrierExtractor func(context.Context) propagation.TextMapCarrier
	tracePaths              tracePath
	topicTransformer        TopicAttributeTransformer
	infoHandler             http.Handler
	histogramBoundaries     map[tracePath][]float64
	attributes              []attribute.KeyValue
}

type tracePath uint

func (tp tracePath) match(o tracePath) bool { return tp&o != 0 }

type disableTracePathOpt struct{ tracePath }

func (o *disableTracePathOpt) apply(opts *options) { opts.tracePaths &^= o.tracePath }

func (t TopicAttributeTransformer) apply(opts *options) { opts.topicTransformer = t }

func (b BucketBoundaries) apply(opts *options) {
	if b.Publisher != nil {
		opts.histogramBoundaries[tracePublisher] = b.Publisher
	}

	if b.Subscriber != nil {
		opts.histogramBoundaries[traceSubscriber] = b.Subscriber
	}

	if b.Unsubscriber != nil {
		opts.histogramBoundaries[traceUnsubscriber] = b.Unsubscriber
	}

	if b.Callback != nil {
		opts.histogramBoundaries[traceCallback] = b.Callback
	}
}
