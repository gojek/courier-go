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
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/gojek/courier-go"
)

func TestUnsubscriberTraceSpan(t *testing.T) {
	tp := trace.NewTracerProvider()
	sr := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(sr)

	reg := prom.NewRegistry()
	exporter, err := prometheus.New(prometheus.WithRegisterer(reg))
	require.NoError(t, err)
	mp := metric.NewMeterProvider(metric.WithReader(exporter))

	mwf := New("test-service", WithTracerProvider(tp), WithMeterProvider(mp))
	uErr := errors.New("error_from_upstream")

	u := mwf.UnsubscriberMiddleware(courier.UnsubscriberFunc(func(ctx context.Context, topics ...string) error {
		return uErr
	}))

	assert.EqualError(t, u.Unsubscribe(context.Background(), "test-topic-1", "test-topic-2"), uErr.Error())

	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]

	t.Run("Attributes", func(t *testing.T) {
		got := attribute.NewSet(span.Attributes()...)
		expected := attribute.NewSet([]attribute.KeyValue{
			semconv.ServiceNameKey.String("test-service"),
			MQTTTopic.StringSlice([]string{"test-topic-1", "test-topic-2"}),
		}...)

		assert.Equal(t, expected, got)
		assert.Equal(t, unsubscribeSpanName, span.Name())
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
		assert.Equal(t, unsubscribeErrMessage, span.Status().Description)
	})

	t.Run("Metrics", func(t *testing.T) {
		vsn := courier.Version()
		buf := bytes.NewBufferString(fmt.Sprintf(`# HELP courier_unsubscribe_attempts_total Number of unsubscribe attempts
# TYPE courier_unsubscribe_attempts_total counter
courier_unsubscribe_attempts_total{mqtt_topic="test-topic-1",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
courier_unsubscribe_attempts_total{mqtt_topic="test-topic-2",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
# HELP courier_unsubscribe_failures_total Number of unsubscribe failures
# TYPE courier_unsubscribe_failures_total counter
courier_unsubscribe_failures_total{mqtt_topic="test-topic-1",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
courier_unsubscribe_failures_total{mqtt_topic="test-topic-2",otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 1
`, vsn, vsn, vsn, vsn))
		assert.NoError(t, testutil.GatherAndCompare(reg, buf,
			"courier_unsubscribe_attempts_total",
			"courier_unsubscribe_failures_total",
		))

		metrics, err := reg.Gather()
		assert.NoError(t, err)

		var found bool
		for _, metricFamily := range metrics {
			if metricFamily.GetName() != "courier_unsubscribe_latency_seconds" {
				continue
			}
			found = true

			for _, m := range metricFamily.GetMetric() {
				assert.EqualValues(t, 1, m.GetHistogram().GetSampleCount())

				for _, labelPair := range m.GetLabel() {
					// assert that MQTTTopic is not a slice, i.e. MQTTTopic.StringSlice(topics)
					if lpName := strings.ReplaceAll(string(MQTTTopic), ".", "_"); labelPair.GetName() == lpName {
						assert.False(t, strings.HasPrefix(labelPair.GetValue(), "["))
						assert.False(t, strings.HasSuffix(labelPair.GetValue(), "]"))
					}
				}
			}
		}

		assert.True(t, found)
	})
}

func TestUnsubscriberSpanNotInstrumented(t *testing.T) {
	u := courier.UnsubscriberFunc(func(ctx context.Context, _ ...string) error {
		span := oteltrace.SpanFromContext(ctx)
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		return nil
	})

	err := u.Unsubscribe(context.Background(), "test-topic-1", "test-topic-2")
	assert.NoError(t, err)
}
