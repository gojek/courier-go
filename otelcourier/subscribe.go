package otelcourier

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sort"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"

	courier "***REMOVED***"
)

const (
	subscribeSpanName           = "otelcourier.Subscribe"
	subscribeMultipleSpanName   = "otelcourier.SubscribeMultiple"
	subscribeErrMessage         = "subscribe error"
	subscribeMultipleErrMessage = "subscribe multiple error"
)

// Subscriber returns a courier.SubscriberMiddlewareFunc which can be used
// to add OpenTelemetry based tracing to courier.Subscriber
func (m *Middleware) Subscriber() courier.SubscriberMiddlewareFunc {
	return func(next courier.Subscriber) courier.Subscriber {
		// This function will wrap the base courier.Publisher and use trace.Tracer to start a trace.Span
		return courier.NewSubscriberFuncs(
			func(ctx context.Context, topic string, qos courier.QOSLevel, callback courier.MessageHandler) error {
				opts := []trace.SpanOption{
					trace.WithAttributes(MQTTTopic.String(topic)),
					trace.WithAttributes(MQTTQoS.Int(int(qos))),
					trace.WithAttributes(semconv.ServiceNameKey.String(m.service)),
					trace.WithSpanKind(trace.SpanKindClient),
				}
				ctx, span := m.tracer.Start(ctx, subscribeSpanName, opts...)
				defer span.End()

				err := next.Subscribe(ctx, topic, qos, m.instrumentCallback(callback))
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, subscribeErrMessage)
				}

				return err
			},
			func(ctx context.Context, topicsWithQos map[string]courier.QOSLevel, callback courier.MessageHandler) error {
				opts := []trace.SpanOption{
					trace.WithAttributes(MQTTTopicWithQoS.Array(mapToArray(topicsWithQos))),
					trace.WithAttributes(semconv.ServiceNameKey.String(m.service)),
					trace.WithSpanKind(trace.SpanKindClient),
				}
				ctx, span := m.tracer.Start(ctx, subscribeMultipleSpanName, opts...)
				defer span.End()

				err := next.SubscribeMultiple(ctx, topicsWithQos, m.instrumentCallback(callback))
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, subscribeMultipleErrMessage)
				}

				return err
			},
		)
	}
}

func (m *Middleware) instrumentCallback(in courier.MessageHandler) courier.MessageHandler {
	if m.disableCallbackTracing {
		return in
	}

	return func(ctx context.Context, pubSub courier.PubSub, decoder courier.Decoder) {
		spanName := "UnknownSubscribeCallback"
		if fnPtr := runtime.FuncForPC(reflect.ValueOf(in).Pointer()); fnPtr != nil {
			spanName = fnPtr.Name()
		}

		ctx, span := m.tracer.Start(ctx, spanName)
		defer span.End()

		in(ctx, pubSub, decoder)
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
