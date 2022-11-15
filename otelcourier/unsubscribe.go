package otelcourier

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	courier "github.com/gojek/courier-go"
)

const (
	unsubscribeSpanName   = "otelcourier.Unsubscribe"
	unsubscribeErrMessage = "unsubscribe error"
)

func (t *Tracer) UnsubscriberMiddleware(next courier.Unsubscriber) courier.Unsubscriber {
	return courier.UnsubscriberFunc(func(ctx context.Context, topics ...string) error {
		opts := []trace.SpanStartOption{
			trace.WithAttributes(MQTTTopic.StringSlice(topics)),
			trace.WithAttributes(semconv.ServiceNameKey.String(t.service)),
			trace.WithSpanKind(trace.SpanKindClient),
		}
		ctx, span := t.tracer.Start(ctx, unsubscribeSpanName, opts...)
		defer span.End()

		err := next.Unsubscribe(ctx, topics...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, unsubscribeErrMessage)
		}

		return err
	})
}
