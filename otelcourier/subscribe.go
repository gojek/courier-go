package otelcourier

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gojek/courier-go"
)

const (
	subscribeSpanName           = "otelcourier.Subscribe"
	subscribeMultipleSpanName   = "otelcourier.SubscribeMultiple"
	subscribeErrMessage         = "subscribe error"
	subscribeMultipleErrMessage = "subscribe multiple error"

	moduleNamedGroup = "module"
	pkgNamedGroup    = "pkg"
	fnNamedGroup     = "fn"
)

var (
	runtimeCallbackExtractor = regexp.MustCompile(
		fmt.Sprintf(`^(?P<%s>.+?)\/(?P<%s>[^\/.]+)\.(?P<%s>.+)$`, moduleNamedGroup, pkgNamedGroup, fnNamedGroup),
	)
	moduleIndex = runtimeCallbackExtractor.SubexpIndex(moduleNamedGroup)
	pkgIndex    = runtimeCallbackExtractor.SubexpIndex(pkgNamedGroup)
	fnIndex     = runtimeCallbackExtractor.SubexpIndex(fnNamedGroup)
)

// SubscriberMiddleware is a courier.SubscriberMiddlewareFunc for tracing subscribe calls.
func (t *OTel) SubscriberMiddleware(next courier.Subscriber) courier.Subscriber {
	return courier.NewSubscriberFuncs(
		func(ctx context.Context, topic string, callback courier.MessageHandler, opts ...courier.Option) error {
			attrs := append([]attribute.KeyValue{
				semconv.ServiceNameKey.String(t.service),
			}, mapAttributes(opts)...)
			metricAttrs := metric.WithAttributes(append(attrs, MQTTTopic.String(t.topicTransformer(ctx, topic)))...)

			defer func(ctx context.Context, now time.Time, attrs metric.MeasurementOption) {
				t.rc.recordLatency(ctx, traceSubscriber, time.Since(now), attrs)
			}(ctx, t.tnow(), metricAttrs)

			ctx, span := t.tracer.Start(ctx, subscribeSpanName,
				trace.WithAttributes(append(attrs, MQTTTopic.String(topic))...),
				trace.WithSpanKind(trace.SpanKindClient),
			)
			defer span.End()

			t.rc.incAttempt(ctx, traceSubscriber, metricAttrs)

			err := next.Subscribe(ctx, topic, t.instrumentCallback(callback))
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, subscribeErrMessage)

				t.rc.incFailure(ctx, traceSubscriber, metricAttrs)
			}

			return err
		},
		func(ctx context.Context, topicsWithQos map[string]courier.QOSLevel, callback courier.MessageHandler) error {
			unnestMetricAttrs := make([]metric.MeasurementOption, 0, len(topicsWithQos))
			for topic, qos := range topicsWithQos {
				unnestMetricAttrs = append(unnestMetricAttrs, metric.WithAttributes(
					semconv.ServiceNameKey.String(t.service),
					MQTTTopic.String(t.topicTransformer(ctx, topic)),
					MQTTQoS.Int(int(qos)),
				))
			}

			defer func(ctx context.Context, now time.Time, attrs ...metric.MeasurementOption) {
				for _, attr := range attrs {
					t.rc.recordLatency(ctx, traceSubscriber, time.Since(now), attr)
				}
			}(ctx, t.tnow(), unnestMetricAttrs...)

			ctx, span := t.tracer.Start(ctx, subscribeMultipleSpanName,
				trace.WithAttributes(
					semconv.ServiceNameKey.String(t.service),
					MQTTTopicWithQoS.StringSlice(mapToArray(topicsWithQos)),
				),
				trace.WithSpanKind(trace.SpanKindClient),
			)
			defer span.End()

			for _, attr := range unnestMetricAttrs {
				t.rc.incAttempt(ctx, traceSubscriber, attr)
			}

			err := next.SubscribeMultiple(ctx, topicsWithQos, t.instrumentCallback(callback))
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, subscribeMultipleErrMessage)

				for _, attr := range unnestMetricAttrs {
					t.rc.incFailure(ctx, traceSubscriber, attr)
				}
			}

			return err
		},
	)
}

func (t *OTel) instrumentCallback(in courier.MessageHandler) courier.MessageHandler {
	if !t.tracePaths.match(traceCallback) {
		return in
	}

	return func(ctx context.Context, pubSub courier.PubSub, msg *courier.Message) {
		spanName := "UnknownSubscribeCallback"
		pkgName := "Unknown"
		fnName := "UnknownFunc"

		if fnPtr := runtime.FuncForPC(reflect.ValueOf(in).Pointer()); fnPtr != nil {
			fullName := fnPtr.Name()

			if matches := runtimeCallbackExtractor.FindStringSubmatch(fullName); len(matches) > 0 {
				spanName = matches[fnIndex]
				pkgName = fmt.Sprintf("%s/%s", matches[moduleIndex], matches[pkgIndex])
				fnName = matches[fnIndex]
			}
		}

		attrs := []attribute.KeyValue{
			semconv.ServiceNameKey.String(t.service),
			semconv.CodeNamespace(pkgName),
			semconv.CodeFunction(fnName),
			MQTTQoS.Int(int(msg.QoS)),
			MQTTRetained.Bool(msg.Retained),
		}

		if t.textMapCarrierFunc != nil {
			ctx = t.propagator.Extract(ctx, t.textMapCarrierFunc(ctx))
		}

		metricAttrs := metric.WithAttributes(append(attrs,
			MQTTTopic.String(t.topicTransformer(ctx, msg.Topic)),
		)...)

		defer func(ctx context.Context, now time.Time, attrs metric.MeasurementOption) {
			t.rc.recordLatency(ctx, traceCallback, time.Since(now), attrs)
		}(ctx, t.tnow(), metricAttrs)

		ctx, span := t.tracer.Start(ctx, spanName, trace.WithAttributes(append(attrs, MQTTTopic.String(msg.Topic))...))
		defer span.End()

		t.rc.incAttempt(ctx, traceCallback, metricAttrs)

		in(ctx, pubSub, msg)

		if ros, ok := span.(tracesdk.ReadOnlySpan); ok && ros.Status().Code == codes.Error {
			t.rc.incFailure(ctx, traceCallback, metricAttrs)
		}
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
