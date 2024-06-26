package otelcourier

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestPublishTraceSpan(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	reg := prom.NewRegistry()
	exporter, err := prometheus.New(prometheus.WithRegisterer(reg))
	require.NoError(t, err)
	mp := metric.NewMeterProvider(metric.WithReader(exporter))

	mtmc := newMockTextMapCarrier(t, "hello-world")

	mwf := New("test-service", WithTracerProvider(tp), WithMeterProvider(mp),
		WithTextMapPropagator(propagation.NewCompositeTextMapPropagator(&propagation.TraceContext{})))
	uErr := errors.New("error_from_upstream")

	p := mwf.PublisherMiddleware(courier.PublisherFunc(func(ctx context.Context, topic string, message any, opts ...courier.Option) error {
		return uErr
	}))

	traceParentRegex := regexp.MustCompile(`^\d{2}-\w{32}-\w{16}-\d{2}$`)
	mtmc.On("Set", "traceparent", mock.MatchedBy(func(in string) bool {
		return traceParentRegex.MatchString(in)
	}))

	assert.EqualError(t, p.Publish(context.Background(), "test-topic", mtmc, courier.QOSOne, courier.Retained(false)), uErr.Error())

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		got := attribute.NewSet(span.Attributes()...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ServiceNameKey.String("test-service"),
			MQTTTopic.String("test-topic"),
			MQTTQoS.Int(1),
			MQTTRetained.Bool(false),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, publishSpanName, span.Name())
		assert.Equal(t, oteltrace.SpanKindProducer, span.SpanKind())
	})

	t.Run("Events", func(t *testing.T) {
		got := attribute.NewSet(span.Events()[0].Attributes...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ExceptionMessageKey.String(uErr.Error()),
			semconv.ExceptionTypeKey.String(reflect.TypeOf(uErr).String()),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, codes.Error, span.Status().Code)
		assert.Equal(t, publishErrMessage, span.Status().Description)
	})

	t.Run("Metrics", func(t *testing.T) {
		vsn := courier.Version()
		buf := bytes.NewBufferString(fmt.Sprintf(`# HELP courier_publish_attempts_total Number of publish attempts
# TYPE courier_publish_attempts_total counter
courier_publish_attempts_total{mqtt_qos="1",mqtt_retained="false",mqtt_topic="test-topic",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
# HELP courier_publish_failures_total Number of publish failures
# TYPE courier_publish_failures_total counter
courier_publish_failures_total{mqtt_qos="1",mqtt_retained="false",mqtt_topic="test-topic",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
`, vsn, vsn))
		assert.NoError(t, testutil.GatherAndCompare(reg, buf,
			"courier_publish_attempts_total",
			"courier_publish_failures_total",
		))

		metrics, err := reg.Gather()
		assert.NoError(t, err)

		var found bool
		for _, metricFamily := range metrics {
			if metricFamily.GetName() != "courier_publish_latency_seconds" {
				continue
			}
			found = true

			for _, m := range metricFamily.GetMetric() {
				assert.EqualValues(t, 1, m.GetHistogram().GetSampleCount())
			}
		}

		assert.True(t, found)
	})

	mtmc.AssertExpectations(t)
}

func TestPublishSpanNotInstrumented(t *testing.T) {
	p := courier.PublisherFunc(func(ctx context.Context, _ string, _ any, _ ...courier.Option) error {
		span := oteltrace.SpanFromContext(ctx)
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		return nil
	})

	err := p.Publish(context.Background(), "test-topic", "hello-world", courier.QOSOne)
	assert.NoError(t, err)
}

func newMockTextMapCarrier(t *testing.T, payload string) *mockTextMapCarrier {
	m := &mockTextMapCarrier{Payload: payload}
	m.Test(t)
	return m
}

type mockTextMapCarrier struct {
	Payload string `json:"payload"`
	mock.Mock
}

func (m *mockTextMapCarrier) Get(key string) string {
	return m.Called(key).String(0)
}

func (m *mockTextMapCarrier) Set(key string, value string) {
	m.Called(key, value)
}

func (m *mockTextMapCarrier) Keys() []string {
	if v, ok := m.Called().Get(0).([]string); ok {
		return v
	}

	return nil
}
