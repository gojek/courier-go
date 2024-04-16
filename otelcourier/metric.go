package otelcourier

import (
	"context"
	"fmt"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var metricFlows = map[tracePath]struct {
	name       string
	boundaries []float64
}{
	tracePublisher:    {name: "publish", boundaries: prom.ExponentialBucketsRange(0.0001, 1, 10)},
	traceSubscriber:   {name: "subscribe", boundaries: prom.ExponentialBucketsRange(0.001, 1, 7)},
	traceUnsubscriber: {name: "unsubscribe", boundaries: prom.ExponentialBucketsRange(0.001, 1, 7)},
	traceCallback:     {name: "subscribe.callback", boundaries: prom.ExponentialBucketsRange(0.001, 10, 15)},
}

func (t *OTel) initRecorders() {
	for path, flow := range metricFlows {
		if !t.tracePaths.match(path) {
			continue
		}

		t.rc[path] = t.newRecorder(flow.name, flow.boundaries)
	}

	if t.infoHandler != nil {
		t.initInfoHandler()
	}
}

func (t *OTel) initInfoHandler() {
	if _, err := t.meter.Int64ObservableUpDownCounter(
		"courier.client.connected",
		metric.WithDescription("Tells if a client is connected or not, partitioned by client ID."),
		metric.WithInt64Callback(infoHandler(t.infoHandler.ServeHTTP).
			callback(semconv.ServiceNameKey.String(t.service))),
	); err != nil {
		panic(err)
	}
}

type recordersOp func(*recorders) error

func (t *OTel) newRecorder(flow string, boundaries []float64) *recorders {
	var rs recorders

	for _, op := range []recordersOp{
		func(r *recorders) error {
			ac, err := t.meter.Int64Counter(
				fmt.Sprintf("courier.%s.attempts", flow),
				metric.WithDescription(fmt.Sprintf("Number of %s attempts", flow)),
			)
			r.attempts = ac

			return err
		},
		func(r *recorders) error {
			fc, err := t.meter.Int64Counter(
				fmt.Sprintf("courier.%s.failures", flow),
				metric.WithDescription(fmt.Sprintf("Number of %s failures", flow)),
			)
			r.failures = fc

			return err
		},
		func(r *recorders) error {
			lt, err := t.meter.Float64Histogram(
				fmt.Sprintf("courier.%s.latency", flow),
				metric.WithDescription(fmt.Sprintf("Latency of %s calls", flow)),
				metric.WithUnit("s"),
				metric.WithExplicitBucketBoundaries(boundaries...),
			)
			r.latency = lt

			return err
		},
	} {
		if err := op(&rs); err != nil {
			panic(err)
		}
	}

	return &rs
}

type recorder map[tracePath]*recorders

func (r recorder) incAttempt(ctx context.Context, path tracePath, opts ...metric.AddOption) {
	c, ok := r[path]
	if !ok {
		return
	}

	c.attempts.Add(ctx, 1, opts...)
}

func (r recorder) incFailure(ctx context.Context, path tracePath, opts ...metric.AddOption) {
	c, ok := r[path]
	if !ok {
		return
	}

	c.failures.Add(ctx, 1, opts...)
}

func (r recorder) recordLatency(
	ctx context.Context, path tracePath, latency time.Duration, opts ...metric.RecordOption,
) {
	c, ok := r[path]
	if !ok {
		return
	}

	c.latency.Record(ctx, latency.Seconds(), opts...)
}

type recorders struct {
	attempts metric.Int64Counter
	failures metric.Int64Counter
	latency  metric.Float64Histogram
}
