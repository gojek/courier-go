package otelcourier

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/gojek/courier-go"
)

type traceparent int

const traceparentKey traceparent = 0

func TestSubscribeTraceSpan(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	reg := prom.NewRegistry()
	exporter, err := prometheus.New(prometheus.WithRegisterer(reg))
	require.NoError(t, err)
	mp := metric.NewMeterProvider(metric.WithReader(exporter))

	mwf := New("test-service", WithTracerProvider(tp), WithMeterProvider(mp))
	uErr := errors.New("error_from_upstream")

	u := mwf.SubscriberMiddleware(courier.NewSubscriberFuncs(
		func(_ context.Context, _ string, _ courier.MessageHandler, _ ...courier.Option) error {
			return uErr
		},
		func(_ context.Context, _ map[string]courier.QOSLevel, _ courier.MessageHandler) error {
			return nil
		},
	))

	assert.EqualError(t, u.Subscribe(context.Background(), "test-topic",
		func(_ context.Context, _ courier.PubSub, _ *courier.Message) {},
		courier.QOSOne,
	), uErr.Error())

	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		assert.Equal(t, subscribeSpanName, span.Name())
		assert.Equal(t, oteltrace.SpanKindClient, span.SpanKind())

		got := attribute.NewSet(span.Attributes()...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ServiceNameKey.String("test-service"),
			MQTTTopic.String("test-topic"),
			MQTTQoS.Int(1),
		}...)

		assert.Equal(t, expected, got)
	})

	t.Run("Events", func(t *testing.T) {
		got := attribute.NewSet(span.Events()[0].Attributes...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ExceptionMessageKey.String(uErr.Error()),
			semconv.ExceptionTypeKey.String(reflect.TypeOf(uErr).String()),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, codes.Error, span.Status().Code)
		assert.Equal(t, subscribeErrMessage, span.Status().Description)
	})

	t.Run("Metrics", func(t *testing.T) {
		vsn := courier.Version()
		buf := bytes.NewBufferString(fmt.Sprintf(`# HELP courier_subscribe_attempts_total Number of subscribe attempts
# TYPE courier_subscribe_attempts_total counter
courier_subscribe_attempts_total{mqtt_qos="1",mqtt_topic="test-topic",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
# HELP courier_subscribe_failures_total Number of subscribe failures
# TYPE courier_subscribe_failures_total counter
courier_subscribe_failures_total{mqtt_qos="1",mqtt_topic="test-topic",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
`, vsn, vsn))
		assert.NoError(t, testutil.GatherAndCompare(reg, buf,
			"courier_subscribe_attempts_total",
			"courier_subscribe_failures_total",
		))

		metrics, err := reg.Gather()
		assert.NoError(t, err)

		var found bool
		for _, metricFamily := range metrics {
			if metricFamily.GetName() != "courier_subscribe_latency_seconds" {
				continue
			}
			found = true

			for _, m := range metricFamily.GetMetric() {
				assert.EqualValues(t, 1, m.GetHistogram().GetSampleCount())
			}
		}

		assert.True(t, found)
	})
}

func TestSubscribeMultipleTraceSpan(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	reg := prom.NewRegistry()
	exporter, err := prometheus.New(prometheus.WithRegisterer(reg))
	require.NoError(t, err)
	mp := metric.NewMeterProvider(metric.WithReader(exporter))

	mwf := New("test-service", WithTracerProvider(tp), WithMeterProvider(mp))
	uErr := errors.New("error_from_upstream")

	u := mwf.SubscriberMiddleware(courier.NewSubscriberFuncs(
		func(_ context.Context, _ string, _ courier.MessageHandler, _ ...courier.Option) error {
			return nil
		},
		func(_ context.Context, _ map[string]courier.QOSLevel, _ courier.MessageHandler) error {
			return uErr
		},
	))

	assert.EqualError(t, u.SubscribeMultiple(context.Background(), map[string]courier.QOSLevel{
		"test-topic-1": courier.QOSOne,
		"test-topic-2": courier.QOSTwo,
	}, func(_ context.Context, _ courier.PubSub, _ *courier.Message) {}), uErr.Error())

	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		got := attribute.NewSet(span.Attributes()...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ServiceNameKey.String("test-service"),
			MQTTTopicWithQoS.StringSlice([]string{
				"test-topic-1 | qos[1]",
				"test-topic-2 | qos[2]",
			}),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, subscribeMultipleSpanName, span.Name())
		assert.Equal(t, oteltrace.SpanKindClient, span.SpanKind())
	})

	t.Run("Events", func(t *testing.T) {
		got := attribute.NewSet(span.Events()[0].Attributes...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ExceptionMessageKey.String(uErr.Error()),
			semconv.ExceptionTypeKey.String(reflect.TypeOf(uErr).String()),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, codes.Error, span.Status().Code)
		assert.Equal(t, subscribeMultipleErrMessage, span.Status().Description)
	})

	t.Run("Metrics", func(t *testing.T) {
		vsn := courier.Version()
		buf := bytes.NewBufferString(fmt.Sprintf(`# HELP courier_subscribe_attempts_total Number of subscribe attempts
# TYPE courier_subscribe_attempts_total counter
courier_subscribe_attempts_total{mqtt_qos="1",mqtt_topic="test-topic-1",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
courier_subscribe_attempts_total{mqtt_qos="2",mqtt_topic="test-topic-2",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
# HELP courier_subscribe_failures_total Number of subscribe failures
# TYPE courier_subscribe_failures_total counter
courier_subscribe_failures_total{mqtt_qos="1",mqtt_topic="test-topic-1",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
courier_subscribe_failures_total{mqtt_qos="2",mqtt_topic="test-topic-2",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
`, vsn, vsn, vsn, vsn))
		assert.NoError(t, testutil.GatherAndCompare(reg, buf,
			"courier_subscribe_attempts_total",
			"courier_subscribe_failures_total",
		))

		metrics, err := reg.Gather()
		assert.NoError(t, err)

		var found bool
		for _, metricFamily := range metrics {
			if metricFamily.GetName() != "courier_subscribe_latency_seconds" {
				continue
			}
			found = true

			for _, m := range metricFamily.GetMetric() {
				assert.EqualValues(t, 1, m.GetHistogram().GetSampleCount())

				for _, labelPair := range m.GetLabel() {
					assert.NotEqual(t,
						strings.ReplaceAll(string(MQTTTopicWithQoS), ".", "_"),
						labelPair.GetName(),
					)
				}
			}
		}

		assert.True(t, found)
	})
}

func TestSubscriberSpanNotInstrumented(t *testing.T) {
	u := courier.UnsubscriberFunc(func(ctx context.Context, _ ...string) error {
		span := oteltrace.SpanFromContext(ctx)
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		return nil
	})

	err := u.Unsubscribe(context.Background(), "test-topic-1", "test-topic-2")
	assert.NoError(t, err)
}

func Test_instrumentCallback(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	reg := prom.NewRegistry()
	exporter, err := prometheus.New(prometheus.WithRegisterer(reg))
	require.NoError(t, err)
	mp := metric.NewMeterProvider(metric.WithReader(exporter))

	extFn := func(ctx context.Context) propagation.TextMapCarrier {
		return ctx.Value(traceparentKey).(propagation.TextMapCarrier)
	}

	m := New("test-service",
		WithTracerProvider(tp),
		WithMeterProvider(mp),
		WithTextMapPropagator(propagation.NewCompositeTextMapPropagator(&propagation.TraceContext{})),
		WithTextMapCarrierExtractFunc(extFn),
	)

	callback := m.instrumentCallback(func(ctx context.Context, _ courier.PubSub, _ *courier.Message) {
		span := oteltrace.SpanFromContext(ctx)

		span.SetAttributes(attribute.String("test-attr", "test-value"))
		span.RecordError(errors.New("test-error"))
		span.SetStatus(codes.Error, "test-error-msg")
	})

	c, _ := courier.NewClient()
	callback(context.WithValue(context.Background(), traceparentKey, &propagation.MapCarrier{
		"traceparent": "00-c8e801456e8232f618c49c6f65f101db-1986c136102242cd-01",
	}), c, &courier.Message{
		Topic:    "test-topic",
		QoS:      courier.QOSOne,
		Retained: true,
	})

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]

	t.Run("Span", func(t *testing.T) {
		assert.Equal(t, "c8e801456e8232f618c49c6f65f101db", span.SpanContext().TraceID().String())
		assert.Equal(t, "otelcourier.Test_instrumentCallback.func2", span.Name())
	})

	t.Run("Attributes", func(t *testing.T) {
		got := attribute.NewSet(span.Attributes()...)
		expected := attribute.NewSet([]attribute.KeyValue{
			MQTTTopic.String("test-topic"),
			MQTTQoS.Int(int(courier.QOSOne)),
			MQTTRetained.Bool(true),
			semconv.ServiceNameKey.String("test-service"),
			attribute.String("test-attr", "test-value"),
		}...)

		assert.Equal(t, expected, got)
	})

	t.Run("Events", func(t *testing.T) {
		got := attribute.NewSet(span.Events()[0].Attributes...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ExceptionMessageKey.String("test-error"),
			semconv.ExceptionTypeKey.String(reflect.TypeOf(errors.New("test-error")).String()),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, codes.Error, span.Status().Code)
		assert.Equal(t, "test-error-msg", span.Status().Description)
	})

	t.Run("Metrics", func(t *testing.T) {
		vsn := courier.Version()
		buf := bytes.NewBufferString(fmt.Sprintf(`# HELP courier_subscribe_callback_attempts_total Number of subscribe.callback attempts
# TYPE courier_subscribe_callback_attempts_total counter
courier_subscribe_callback_attempts_total{callback_name="otelcourier.Test_instrumentCallback.func2",mqtt_qos="1",mqtt_retained="true",mqtt_topic="test-topic",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
# HELP courier_subscribe_callback_failures_total Number of subscribe.callback failures
# TYPE courier_subscribe_callback_failures_total counter
courier_subscribe_callback_failures_total{callback_name="otelcourier.Test_instrumentCallback.func2",mqtt_qos="1",mqtt_retained="true",mqtt_topic="test-topic",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
`, vsn, vsn))

		assert.NoError(t, testutil.GatherAndCompare(reg, buf,
			"courier_subscribe_callback_attempts_total",
			"courier_subscribe_callback_failures_total",
		))
	})
}

func Test_instrumentCallbackDisabled(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	m := New("test-service", WithTracerProvider(tp), DisableCallbackTracing)

	callback := m.instrumentCallback(func(_ context.Context, _ courier.PubSub, _ *courier.Message) {})

	c, _ := courier.NewClient()
	callback(context.Background(), c, &courier.Message{})

	spans := sr.Ended()
	require.Len(t, spans, 0)
}
