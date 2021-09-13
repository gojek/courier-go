package otelcourier

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"

	courier "***REMOVED***"
)

const (
	publishSpanName   = "otelcourier.Publish"
	publishErrMessage = "publish error"
)

// Publisher returns a courier.PublisherMiddlewareFunc which can be used
// to add OpenTelemetry based tracing to courier.Publisher
func (m *Middleware) Publisher() courier.PublisherMiddlewareFunc {
	return func(next courier.Publisher) courier.Publisher {
		// This function will wrap the base courier.Publisher and use trace.Tracer to start a trace.Span
		return courier.PublisherFunc(func(
			ctx context.Context,
			topic string,
			qos courier.QOSLevel,
			retained bool,
			message interface{},
		) error {
			opts := []trace.SpanOption{
				trace.WithAttributes(MQTTTopic.String(topic)),
				trace.WithAttributes(MQTTQoS.Int(int(qos))),
				trace.WithAttributes(MQTTRetained.Bool(retained)),
				trace.WithAttributes(semconv.ServiceNameKey.String(m.service)),
				trace.WithSpanKind(trace.SpanKindProducer),
			}
			ctx, span := m.tracer.Start(ctx, publishSpanName, opts...)
			defer span.End()

			err := next.Publish(ctx, topic, qos, retained, message)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, publishErrMessage)
			}

			return err
		})
	}
}
