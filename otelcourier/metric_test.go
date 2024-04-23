package otelcourier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/metric"
)

func Test_recorderOpsWithoutInitialization(t *testing.T) {
	r := make(recorder)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		r.incAttempt(ctx, tracePublisher)
		r.incFailure(ctx, tracePublisher)
		r.recordLatency(ctx, tracePublisher, 0)
	})
}

func Test_recorderPanicsWithInvalidFlowName(t *testing.T) {
	ot := &OTel{meter: metric.NewMeterProvider().Meter(tracerName)}

	assert.Panics(t, func() { _ = ot.newRecorder("invalid%flow", nil) })
}
