package otelcourier

import (
	"context"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gojek/courier-go"
)

const (
	publishSpanName   = "otelcourier.Publish"
	publishErrMessage = "publish error"
)

// PublisherMiddleware is a courier.PublisherMiddlewareFunc for tracing publish calls.
func (t *OTel) PublisherMiddleware(next courier.Publisher) courier.Publisher {
	return courier.PublisherFunc(func(
		ctx context.Context,
		topic string,
		message any,
		opts ...courier.Option,
	) error {
		attrs := append([]attribute.KeyValue{
			semconv.ServiceNameKey.String(t.service),
		}, mapAttributes(opts)...)

		attrs = append(attrs, t.attributes...)

		var clientID atomic.Value

		ctx = courier.WithClientIDCallback(ctx, func(id string) {
			clientID.Store(id)
		})

		metricAttrsFunc := func() metric.MeasurementOption {
			baseAttrs := append(attrs, MQTTTopic.String(t.topicTransformer(ctx, topic)))
			if id, ok := clientID.Load().(string); ok && id != "" {
				baseAttrs = append(baseAttrs, MQTTClientID.String(id))
			}

			return metric.WithAttributes(baseAttrs...)
		}

		defer func(ctx context.Context, now time.Time) {
			t.rc.recordLatency(ctx, tracePublisher, time.Since(now), metricAttrsFunc())
		}(ctx, t.tnow())

		ctx, span := t.tracer.Start(ctx, publishSpanName,
			trace.WithAttributes(append(attrs, MQTTTopic.String(topic))...),
			trace.WithSpanKind(trace.SpanKindProducer),
		)
		defer span.End()

		if tmc, ok := message.(propagation.TextMapCarrier); ok {
			t.propagator.Inject(ctx, tmc)
		}

		err := next.Publish(ctx, topic, message, opts...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, publishErrMessage)

			t.rc.incFailure(ctx, tracePublisher, metricAttrsFunc())
		}

		t.rc.incAttempt(ctx, tracePublisher, metricAttrsFunc())

		return err
	})
}

func mapAttributes(opts []courier.Option) []attribute.KeyValue {
	res := make([]attribute.KeyValue, 0, len(opts))

	for _, opt := range opts {
		switch opt := opt.(type) {
		case courier.QOSLevel:
			res = append(res, MQTTQoS.Int(int(opt)))
		case courier.Retained:
			res = append(res, MQTTRetained.Bool(bool(opt)))
		}
	}

	return res
}
