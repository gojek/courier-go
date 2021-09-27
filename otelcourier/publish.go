package otelcourier

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	courier "***REMOVED***"
)

const (
	publishSpanName   = "otelcourier.Publish"
	publishErrMessage = "publish error"
)

func (t *Tracer) publisher(next courier.Publisher) courier.Publisher {
	return courier.PublisherFunc(func(
		ctx context.Context,
		topic string,
		qos courier.QOSLevel,
		retained bool,
		message interface{},
	) error {
		opts := []trace.SpanStartOption{
			trace.WithAttributes(MQTTTopic.String(topic)),
			trace.WithAttributes(MQTTQoS.Int(int(qos))),
			trace.WithAttributes(MQTTRetained.Bool(retained)),
			trace.WithAttributes(semconv.ServiceNameKey.String(t.service)),
			trace.WithSpanKind(trace.SpanKindProducer),
		}
		ctx, span := t.tracer.Start(ctx, publishSpanName, opts...)
		defer span.End()

		err := next.Publish(ctx, topic, qos, retained, message)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, publishErrMessage)
		}

		return err
	})
}
