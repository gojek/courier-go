package otelcourier

import (
	"context"
	"sync/atomic"
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
		var clientID atomic.Value

		ctx = courier.WithClientIDCallback(ctx, func(id string) {
			clientID.Store(id)
		})

		getClientIDAttr := func() []attribute.KeyValue {
			if id, ok := clientID.Load().(string); ok && id != "" {
				return []attribute.KeyValue{MQTTClientID.String(id)}
			}

			return nil
		}

		unnestMetricAttrsFunc := func() []metric.MeasurementOption {
			result := make([]metric.MeasurementOption, 0, len(topics))

			for _, topic := range topics {
				attrs := append([]attribute.KeyValue{
					semconv.ServiceNameKey.String(t.service),
					MQTTTopic.String(t.topicTransformer(ctx, topic)),
				}, t.attributes...)
				attrs = append(attrs, getClientIDAttr()...)
				result = append(result, metric.WithAttributes(attrs...))
			}

			return result
		}

		defer func(ctx context.Context, now time.Time) {
			for _, attr := range unnestMetricAttrsFunc() {
				t.rc.recordLatency(ctx, traceUnsubscriber, time.Since(now), attr)
			}
		}(ctx, t.tnow())

		ctx, span := t.tracer.Start(ctx, unsubscribeSpanName,
			trace.WithAttributes(append([]attribute.KeyValue{
				semconv.ServiceNameKey.String(t.service),
				MQTTTopic.StringSlice(topics),
			}, t.attributes...)...),
			trace.WithSpanKind(trace.SpanKindClient),
		)
		defer span.End()

		for _, attr := range unnestMetricAttrsFunc() {
			t.rc.incAttempt(ctx, traceUnsubscriber, attr)
		}

		err := next.Unsubscribe(ctx, topics...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, unsubscribeErrMessage)

			for _, attr := range unnestMetricAttrsFunc() {
				t.rc.incFailure(ctx, traceUnsubscriber, attr)
			}
		}

		return err
	})
}
