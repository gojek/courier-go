package otelcourier

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/propagation"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
)

func TestOption(t *testing.T) {
	testcases := []struct {
		name    string
		options []Option
		want    *traceOptions
	}{
		{
			name: "DefaultOptions",
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				propagator:     otel.GetTextMapPropagator(),
				tracePaths:     tracePublisher + traceSubscriber + traceUnsubscriber + traceCallback,
			},
		},
		{
			name:    "DisablePublisher",
			options: []Option{DisablePublisherTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				propagator:     otel.GetTextMapPropagator(),
				tracePaths:     traceSubscriber + traceUnsubscriber + traceCallback,
			},
		},
		{
			name:    "DisableSubscriber",
			options: []Option{DisableSubscriberTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				propagator:     otel.GetTextMapPropagator(),
				tracePaths:     tracePublisher + traceUnsubscriber + traceCallback,
			},
		},
		{
			name:    "DisableUnsubscriber",
			options: []Option{DisableUnsubscriberTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				propagator:     otel.GetTextMapPropagator(),
				tracePaths:     tracePublisher + traceSubscriber + traceCallback,
			},
		},
		{
			name:    "DisableCallback",
			options: []Option{DisableCallbackTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				propagator:     otel.GetTextMapPropagator(),
				tracePaths:     tracePublisher + traceSubscriber + traceUnsubscriber,
			},
		},
		{
			name:    "DisableCallbackTwice",
			options: []Option{DisableCallbackTracing, DisableCallbackTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				propagator:     otel.GetTextMapPropagator(),
				tracePaths:     tracePublisher + traceSubscriber + traceUnsubscriber,
			},
		},
		{
			name:    "DisableAllButCallback",
			options: []Option{DisablePublisherTracing, DisableSubscriberTracing, DisableUnsubscriberTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				propagator:     otel.GetTextMapPropagator(),
				tracePaths:     traceCallback,
			},
		},
		{
			name:    "DisableTwoTracers",
			options: []Option{DisablePublisherTracing, DisableSubscriberTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				propagator:     otel.GetTextMapPropagator(),
				tracePaths:     traceUnsubscriber + traceCallback,
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			to := defaultOptions()

			for _, opt := range tt.options {
				opt(to)
			}

			assert.Equal(t, tt.want, to)
		})
	}

	extractorFn := func(_ context.Context) propagation.TextMapCarrier { return &propagation.MapCarrier{} }

	t.Run("TextMapCarrierExtractor", func(t *testing.T) {
		to := defaultOptions()

		WithTextMapCarrierExtractFunc(extractorFn)(to)

		assert.Equal(t, fmt.Sprintf("%p", extractorFn), fmt.Sprintf("%p", to.textMapCarrierExtractor))
	})
}
