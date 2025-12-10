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
		func(r *recorders) error {
			if flow != "subscribe.callback" {
				return nil
			}
			im, err := t.meter.Int64Counter(
				"courier.subscribe.callback.incoming_messages",
				metric.WithDescription("Number of incoming messages received"),
			)
			r.incomingMessages = im

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

func (r recorder) incIncomingMessages(ctx context.Context, path tracePath, opts ...metric.AddOption) {
	c, ok := r[path]
	if !ok || c.incomingMessages == nil {
		return
	}

	c.incomingMessages.Add(ctx, 1, opts...)
}

type recorders struct {
	attempts         metric.Int64Counter
	failures         metric.Int64Counter
	latency          metric.Float64Histogram
	incomingMessages metric.Int64Counter
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

	configMetrics := []struct {
		name        string
		description string
		unit        string
		value       float64
		attrs       []attribute.KeyValue
	}{
		{
			name:        "courier.client.connection_timeout",
			description: "MQTT connection timeout in seconds",
			unit:        "s",
			value:       cCfg.ConnectTimeout().Seconds(),
			attrs:       baseAttrs,
		},
		{
			name:        "courier.client.write_timeout",
			description: "MQTT write timeout in seconds",
			unit:        "s",
			value:       cCfg.WriteTimeout().Seconds(),
			attrs:       baseAttrs,
		},
		{
			name:        "courier.client.keep_alive",
			description: "MQTT keep alive in seconds",
			unit:        "s",
			value:       cCfg.KeepAlive().Seconds(),
			attrs:       baseAttrs,
		},
		{
			name:        "courier.client.ack_timeout",
			description: "MQTT ack timeout in seconds",
			unit:        "s",
			value:       cCfg.AckTimeout().Seconds(),
			attrs:       baseAttrs,
		},
		{
			name:        "courier.client.library_version",
			description: "Courier library version",
			value:       1.0,
			attrs:       append(baseAttrs, attribute.String("version", courier.Version())),
		},
		{
			name:        "courier.client.pool_size",
			description: "Size of the MQTT connection pool",
			value:       float64(cCfg.PoolSize()),
			attrs:       baseAttrs,
		},
	}

	for _, cm := range configMetrics {
		opts := []metric.Float64ObservableGaugeOption{metric.WithDescription(cm.description)}
		if cm.unit != "" {
			opts = append(opts, metric.WithUnit(cm.unit))
		}

		gauge, err := t.meter.Float64ObservableGauge(cm.name, opts...)
		if err != nil {
			panic(err)
		}

		value, attrs := cm.value, cm.attrs

		_, err = t.meter.RegisterCallback(
			func(ctx context.Context, o metric.Observer) error {
				o.ObserveFloat64(gauge, value, metric.WithAttributes(attrs...))

				return nil
			},
			gauge,
		)
		if err != nil {
			panic(err)
		}
	}
}
