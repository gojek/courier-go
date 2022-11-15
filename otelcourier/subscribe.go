package otelcourier

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sort"

	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	courier "github.com/gojek/courier-go"
)

const (
	subscribeSpanName           = "otelcourier.Subscribe"
	subscribeMultipleSpanName   = "otelcourier.SubscribeMultiple"
	subscribeErrMessage         = "subscribe error"
	subscribeMultipleErrMessage = "subscribe multiple error"
)

// SubscriberMiddleware is a courier.SubscriberMiddlewareFunc for tracing subscribe calls.
func (t *Tracer) SubscriberMiddleware(next courier.Subscriber) courier.Subscriber {
	return courier.NewSubscriberFuncs(
		func(ctx context.Context, topic string, callback courier.MessageHandler, opts ...courier.Option) error {
			traceOpts := []trace.SpanStartOption{
				trace.WithAttributes(MQTTTopic.String(topic)),
				trace.WithAttributes(semconv.ServiceNameKey.String(t.service)),
				trace.WithSpanKind(trace.SpanKindClient),
			}
			traceOpts = append(traceOpts, mapOptions(opts)...)

			ctx, span := t.tracer.Start(ctx, subscribeSpanName, traceOpts...)
			defer span.End()

			err := next.Subscribe(ctx, topic, t.instrumentCallback(callback))
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, subscribeErrMessage)
			}

			return err
		},
		func(ctx context.Context, topicsWithQos map[string]courier.QOSLevel, callback courier.MessageHandler) error {
			opts := []trace.SpanStartOption{
				trace.WithAttributes(MQTTTopicWithQoS.StringSlice(mapToArray(topicsWithQos))),
				trace.WithAttributes(semconv.ServiceNameKey.String(t.service)),
				trace.WithSpanKind(trace.SpanKindClient),
			}
			ctx, span := t.tracer.Start(ctx, subscribeMultipleSpanName, opts...)
			defer span.End()

			err := next.SubscribeMultiple(ctx, topicsWithQos, t.instrumentCallback(callback))
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, subscribeMultipleErrMessage)
			}

			return err
		},
	)
}

func (t *Tracer) instrumentCallback(in courier.MessageHandler) courier.MessageHandler {
	if !t.tracePaths.match(traceCallback) {
		return in
	}

	return func(ctx context.Context, pubSub courier.PubSub, msg *courier.Message) {
		spanName := "UnknownSubscribeCallback"
		if fnPtr := runtime.FuncForPC(reflect.ValueOf(in).Pointer()); fnPtr != nil {
			spanName = fnPtr.Name()
		}

		if t.textMapCarrierFunc != nil {
			ctx = t.propagator.Extract(ctx, t.textMapCarrierFunc(ctx))
		}

		ctx, span := t.tracer.Start(ctx, spanName)
		defer span.End()

		in(ctx, pubSub, msg)
	}
}

func mapToArray(topicsWithQos map[string]courier.QOSLevel) []string {
	result := make([]string, 0, len(topicsWithQos))
	for k, v := range topicsWithQos {
		result = append(result, fmt.Sprintf("%s | qos[%d]", k, v))
	}

	sort.Strings(result)

	return result
}
