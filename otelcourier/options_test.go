package otelcourier

import (
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
				tracePaths:     tracePublisher + traceSubscriber + traceUnsubsriber + traceCallback,
			},
		},
		{
			name:    "DisablePublisher",
			options: []Option{DisablePublisherTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				tracePaths:     traceSubscriber + traceUnsubsriber + traceCallback,
			},
		},
		{
			name:    "DisableSubscriber",
			options: []Option{DisableSubscriberTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				tracePaths:     tracePublisher + traceUnsubsriber + traceCallback,
			},
		},
		{
			name:    "DisableUnsubscriber",
			options: []Option{DisableUnsubscriberTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				tracePaths:     tracePublisher + traceSubscriber + traceCallback,
			},
		},
		{
			name:    "DisableCallback",
			options: []Option{DisableCallbackTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				tracePaths:     tracePublisher + traceSubscriber + traceUnsubsriber,
			},
		},
		{
			name:    "DisableCallbackTwice",
			options: []Option{DisableCallbackTracing, DisableCallbackTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				tracePaths:     tracePublisher + traceSubscriber + traceUnsubsriber,
			},
		},
		{
			name:    "DisableAllButCallback",
			options: []Option{DisablePublisherTracing, DisableSubscriberTracing, DisableUnsubscriberTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				tracePaths:     traceCallback,
			},
		},
		{
			name:    "DisableTwoTracers",
			options: []Option{DisablePublisherTracing, DisableSubscriberTracing},
			want: &traceOptions{
				tracerProvider: otel.GetTracerProvider(),
				tracePaths:     traceUnsubsriber + traceCallback,
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
}
