package otelcourier

import (
	"context"
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
		attrs := []attribute.KeyValue{
			MQTTTopic.String(t.topicTransformer(ctx, topic)),
			semconv.ServiceNameKey.String(t.service),
		}
		attrs = append(attrs, mapAttributes(opts)...)

		metricAttrs := metric.WithAttributes(attrs...)

		defer func(ctx context.Context, now time.Time, attrs metric.MeasurementOption) {
			t.rc.recordLatency(ctx, tracePublisher, time.Since(now), attrs)
		}(ctx, t.tnow(), metricAttrs)

		ctx, span := t.tracer.Start(ctx, publishSpanName,
			trace.WithAttributes(attrs...),
			trace.WithSpanKind(trace.SpanKindProducer),
		)
		defer span.End()

		t.rc.incAttempt(ctx, tracePublisher, metricAttrs)

		if tmc, ok := message.(propagation.TextMapCarrier); ok {
			t.propagator.Inject(ctx, tmc)
		}

		err := next.Publish(ctx, topic, message, opts...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, publishErrMessage)

			t.rc.incFailure(ctx, tracePublisher, metricAttrs)
		}

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
