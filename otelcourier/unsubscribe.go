package otelcourier

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	courier "github.com/gojek/courier-go"
)

const (
	unsubscribeSpanName   = "otelcourier.Unsubscribe"
	unsubscribeErrMessage = "unsubscribe error"
)

// UnsubscriberMiddleware is a courier.UnsubscriberMiddlewareFunc for tracing unsubscribe calls.
func (t *OTel) UnsubscriberMiddleware(next courier.Unsubscriber) courier.Unsubscriber {
	return courier.UnsubscriberFunc(func(ctx context.Context, topics ...string) error {
		unnestMetricAttrs := make([]metric.MeasurementOption, 0, len(topics))
		for _, topic := range topics {
			unnestMetricAttrs = append(unnestMetricAttrs, metric.WithAttributes(append([]attribute.KeyValue{
				semconv.ServiceNameKey.String(t.service),
				MQTTTopic.String(t.topicTransformer(ctx, topic)),
			}, t.attributes...)...))
		}

		ctx = courier.WithClientIDCallback(ctx, func(id string) {
			if id != "" {
				for i, topic := range topics {
					unnestMetricAttrs[i] = metric.WithAttributes(append([]attribute.KeyValue{
						semconv.ServiceNameKey.String(t.service),
						MQTTTopic.String(t.topicTransformer(ctx, topic)),
						MQTTClientID.String(id),
					}, t.attributes...)...)
				}
			}
		})

		defer func(ctx context.Context, now time.Time, attrs ...metric.MeasurementOption) {
			for _, attr := range attrs {
				t.rc.recordLatency(ctx, traceUnsubscriber, time.Since(now), attr)
			}
		}(ctx, t.tnow(), unnestMetricAttrs...)

		ctx, span := t.tracer.Start(ctx, unsubscribeSpanName,
			trace.WithAttributes(append([]attribute.KeyValue{
				semconv.ServiceNameKey.String(t.service),
				MQTTTopic.StringSlice(topics),
			}, t.attributes...)...),
			trace.WithSpanKind(trace.SpanKindClient),
		)
		defer span.End()

		err := next.Unsubscribe(ctx, topics...)

		for _, attr := range unnestMetricAttrs {
			t.rc.incAttempt(ctx, traceUnsubscriber, attr)
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, unsubscribeErrMessage)

			for _, attr := range unnestMetricAttrs {
				t.rc.incFailure(ctx, traceUnsubscriber, attr)
			}
		}

		return err
	})
}
