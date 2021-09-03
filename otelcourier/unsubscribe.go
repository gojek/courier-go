package otelcourier

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"

	courier "***REMOVED***"
)

const (
	unsubscribeSpanName   = "otelcourier.Unsubscribe"
	unsubscribeErrMessage = "unsubscribe error"
)

// Unsubscriber returns a courier.UnsubscriberMiddlewareFunc which can be used
// to add OpenTelemetry based tracing to courier.Unsubscriber
func (m *Middleware) Unsubscriber() courier.UnsubscriberMiddlewareFunc {
	return func(next courier.Unsubscriber) courier.Unsubscriber {
		// This function will wrap the base courier.Publisher and use trace.Tracer to start a trace.Span
		return courier.UnsubscriberFunc(func(ctx context.Context, topics ...string) error {
			opts := []trace.SpanOption{
				trace.WithAttributes(MQTTTopic.Array(topics)),
				trace.WithAttributes(semconv.ServiceNameKey.String(m.service)),
				trace.WithSpanKind(trace.SpanKindClient),
			}
			ctx, span := m.tracer.Start(ctx, unsubscribeSpanName, opts...)
			defer span.End()

			err := next.Unsubscribe(ctx, topics...)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, unsubscribeErrMessage)
			}

			return err
		})
	}
}
