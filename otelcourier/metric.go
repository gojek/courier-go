package otelcourier

import (
	"context"
	"fmt"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"github.com/gojek/courier-go"
)

var (
	defaultBoundaries   = prom.ExponentialBucketsRange(0.001, 1, 7)
	defBucketBoundaries = map[tracePath][]float64{
		tracePublisher:    prom.ExponentialBucketsRange(0.0001, 1, 10),
		traceSubscriber:   defaultBoundaries,
		traceUnsubscriber: defaultBoundaries,
		traceCallback:     prom.ExponentialBucketsRange(0.001, 10, 15),
	}

	metricFlows = map[tracePath]string{
		tracePublisher:    "publish",
		traceSubscriber:   "subscribe",
		traceUnsubscriber: "unsubscribe",
		traceCallback:     "subscribe.callback",
	}
)

func (t *OTel) initRecorders(histogramBoundaries map[tracePath][]float64) {
	for path, flow := range metricFlows {
		if !t.tracePaths.match(path) {
			continue
		}

		t.rc[path] = t.newRecorder(flow, histogramBoundaries[path])
	}

	if t.infoHandler != nil {
		t.initInfoHandler()
	}
}

func (t *OTel) initInfoHandler() {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	metricName := fmt.Sprintf("courier.mqtt.%s.connected", id)

	observable, err := t.meter.Int64ObservableUpDownCounter(
		metricName,
		metric.WithDescription("Tells if a client is connected or not, partitioned by client ID."),
	)
	if err != nil {
		panic(err)
	}

	registration, err := t.meter.RegisterCallback(
		infoHandler(t.infoHandler.ServeHTTP).callback(observable, append([]attribute.KeyValue{
			semconv.ServiceNameKey.String(t.service),
		}, t.attributes...)...),
		observable,
	)
	if err != nil {
		panic(err)
	}

	t.infoHandlerRegistration = registration
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

type CourierConfig interface {
	ConnectTimeout() time.Duration
	WriteTimeout() time.Duration
	KeepAlive() time.Duration
	AckTimeout() time.Duration
	PoolSize() int
}

func (t *OTel) initCourierConfig(c UseMiddleware) {
	cCfg, ok := c.(CourierConfig)
	if !ok {
		return
	}

	t.setupCourierMetrics(cCfg)
}

func (t *OTel) setupCourierMetrics(cCfg CourierConfig) {
	baseAttrs := append([]attribute.KeyValue{
		attribute.String("service.name", t.service),
	}, t.attributes...)

	ctx := context.Background()

	timeoutMetrics := []struct {
		name        string
		description string
		value       float64
	}{
		{
			name:        "courier.client.connection_timeout",
			description: "MQTT connection timeout in seconds",
			value:       cCfg.ConnectTimeout().Seconds(),
		},
		{
			name:        "courier.client.write_timeout",
			description: "MQTT write timeout in seconds",
			value:       cCfg.WriteTimeout().Seconds(),
		},
		{
			name:        "courier.client.keep_alive",
			description: "MQTT keep alive in seconds",
			value:       cCfg.KeepAlive().Seconds(),
		},
		{
			name:        "courier.client.ack_timeout",
			description: "MQTT ack timeout in seconds",
			value:       cCfg.AckTimeout().Seconds(),
		},
	}

	for _, tm := range timeoutMetrics {
		gauge, err := t.meter.Float64UpDownCounter(
			tm.name,
			metric.WithDescription(tm.description),
			metric.WithUnit("s"),
		)
		if err != nil {
			panic(err)
		}

		gauge.Add(ctx, tm.value, metric.WithAttributes(baseAttrs...))
	}

	versionGauge, err := t.meter.Float64UpDownCounter(
		"courier.client.library_version",
		metric.WithDescription("Courier library version"),
	)
	if err != nil {
		panic(err)
	}

	versionAttrs := append(baseAttrs, attribute.String("version", courier.Version()))
	versionGauge.Add(ctx, 1.0, metric.WithAttributes(versionAttrs...))

	poolSizeGauge, err := t.meter.Float64UpDownCounter(
		"courier.client.pool_size",
		metric.WithDescription("Size of the MQTT connection pool"),
	)
	if err != nil {
		panic(err)
	}

	poolSizeGauge.Add(ctx, float64(cCfg.PoolSize()), metric.WithAttributes(baseAttrs...))
}
