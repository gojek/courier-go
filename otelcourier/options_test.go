package otelcourier

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func TestOption(t *testing.T) {
	testcases := []struct {
		name    string
		options []Option
		want    *options
	}{
		{
			name: "DefaultOptions",
			want: &options{
				tracerProvider:      otel.GetTracerProvider(),
				meterProvider:       otel.GetMeterProvider(),
				propagator:          otel.GetTextMapPropagator(),
				tracePaths:          tracePublisher + traceSubscriber + traceUnsubscriber + traceCallback,
				histogramBoundaries: defBucketBoundaries,
			},
		},
		{
			name:    "DisablePublisher",
			options: []Option{DisablePublisherTracing},
			want: &options{
				tracerProvider:      otel.GetTracerProvider(),
				meterProvider:       otel.GetMeterProvider(),
				propagator:          otel.GetTextMapPropagator(),
				tracePaths:          traceSubscriber + traceUnsubscriber + traceCallback,
				histogramBoundaries: defBucketBoundaries,
			},
		},
		{
			name:    "DisableSubscriber",
			options: []Option{DisableSubscriberTracing},
			want: &options{
				tracerProvider:      otel.GetTracerProvider(),
				meterProvider:       otel.GetMeterProvider(),
				propagator:          otel.GetTextMapPropagator(),
				tracePaths:          tracePublisher + traceUnsubscriber + traceCallback,
				histogramBoundaries: defBucketBoundaries,
			},
		},
		{
			name:    "DisableUnsubscriber",
			options: []Option{DisableUnsubscriberTracing},
			want: &options{
				tracerProvider:      otel.GetTracerProvider(),
				meterProvider:       otel.GetMeterProvider(),
				propagator:          otel.GetTextMapPropagator(),
				tracePaths:          tracePublisher + traceSubscriber + traceCallback,
				histogramBoundaries: defBucketBoundaries,
			},
		},
		{
			name:    "DisableCallback",
			options: []Option{DisableCallbackTracing},
			want: &options{
				tracerProvider:      otel.GetTracerProvider(),
				meterProvider:       otel.GetMeterProvider(),
				propagator:          otel.GetTextMapPropagator(),
				tracePaths:          tracePublisher + traceSubscriber + traceUnsubscriber,
				histogramBoundaries: defBucketBoundaries,
			},
		},
		{
			name:    "DisableCallbackTwice",
			options: []Option{DisableCallbackTracing, DisableCallbackTracing},
			want: &options{
				tracerProvider:      otel.GetTracerProvider(),
				meterProvider:       otel.GetMeterProvider(),
				propagator:          otel.GetTextMapPropagator(),
				tracePaths:          tracePublisher + traceSubscriber + traceUnsubscriber,
				histogramBoundaries: defBucketBoundaries,
			},
		},
		{
			name:    "DisableAllButCallback",
			options: []Option{DisablePublisherTracing, DisableSubscriberTracing, DisableUnsubscriberTracing},
			want: &options{
				tracerProvider:      otel.GetTracerProvider(),
				meterProvider:       otel.GetMeterProvider(),
				propagator:          otel.GetTextMapPropagator(),
				tracePaths:          traceCallback,
				histogramBoundaries: defBucketBoundaries,
			},
		},
		{
			name:    "DisableTwoTracers",
			options: []Option{DisablePublisherTracing, DisableSubscriberTracing},
			want: &options{
				tracerProvider:      otel.GetTracerProvider(),
				meterProvider:       otel.GetMeterProvider(),
				propagator:          otel.GetTextMapPropagator(),
				tracePaths:          traceUnsubscriber + traceCallback,
				histogramBoundaries: defBucketBoundaries,
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()

			for _, opt := range tt.options {
				opt.apply(o)
			}

			// Ignore topicTransformer
			o.topicTransformer = nil

			assert.Equal(t, tt.want, o)
		})
	}

	extractorFn := func(_ context.Context) propagation.TextMapCarrier { return &propagation.MapCarrier{} }

	t.Run("TextMapCarrierExtractor", func(t *testing.T) {
		o := defaultOptions()

		WithTextMapCarrierExtractFunc(extractorFn).apply(o)

		assert.Equal(t, fmt.Sprintf("%p", extractorFn), fmt.Sprintf("%p", o.textMapCarrierExtractor))
	})

	t.Run("DefaultTopicAttributeTransformer", func(t *testing.T) {
		assert.Equal(t, "topic", DefaultTopicAttributeTransformer(context.Background(), "topic"))
	})

	customTransformer := func(_ context.Context, topic string) string { return topic + "custom" }

	t.Run("CustomTopicAttributeTransformer", func(t *testing.T) {
		o := defaultOptions()

		TopicAttributeTransformer(customTransformer).apply(o)

		assert.Equal(t, "topiccustom", o.topicTransformer(context.Background(), "topic"))
	})

	t.Run("BucketBoundaries", func(t *testing.T) {
		o := defaultOptions()

		bb := BucketBoundaries{
			Publisher:    []float64{1, 2, 3},
			Subscriber:   []float64{4, 5, 6},
			Unsubscriber: []float64{7, 8, 9},
			Callback:     []float64{10, 11, 12},
		}

		bb.apply(o)

		assert.Equal(t, []float64{1, 2, 3}, o.histogramBoundaries[tracePublisher])
		assert.Equal(t, []float64{4, 5, 6}, o.histogramBoundaries[traceSubscriber])
		assert.Equal(t, []float64{7, 8, 9}, o.histogramBoundaries[traceUnsubscriber])
		assert.Equal(t, []float64{10, 11, 12}, o.histogramBoundaries[traceCallback])
	})
}
