package otelcourier

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/gojek/courier-go"
)

func Test_recorderOpsWithoutInitialization(t *testing.T) {
	r := make(recorder)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		r.incAttempt(ctx, tracePublisher)
		r.incFailure(ctx, tracePublisher)
		r.recordLatency(ctx, tracePublisher, 0)
	})
}

func Test_recorderPanicsWithInvalidFlowName(t *testing.T) {
	ot := &OTel{meter: metric.NewMeterProvider().Meter(tracerName)}

	assert.Panics(t, func() { _ = ot.newRecorder("invalid%flow", nil) })
}

func Test_courierConfigMetrics(t *testing.T) {
	reg := prom.NewRegistry()
	exporter, err := prometheus.New(prometheus.WithRegisterer(reg))
	require.NoError(t, err)
	mp := metric.NewMeterProvider(metric.WithReader(exporter))

	mw := New("test-service", WithMeterProvider(mp))
	client, err := courier.NewClient(
		courier.WithAddress("localhost", 1883),
		courier.WithAckTimeout(20),
		courier.WithConnectTimeout(30),
		courier.WithKeepAlive(60),
		courier.WithWriteTimeout(30),
	)
	require.NoError(t, err)

	mw.ApplyMiddlewares(client)

	t.Run("Metrics", func(t *testing.T) {
		vsn := courier.Version()
		buf := bytes.NewBufferString(fmt.Sprintf(`# HELP courier_client_ack_timeout_seconds MQTT ack timeout in seconds
# TYPE courier_client_ack_timeout_seconds gauge
courier_client_ack_timeout_seconds{otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 2e-08
# HELP courier_client_connection_timeout_seconds MQTT connection timeout in seconds
# TYPE courier_client_connection_timeout_seconds gauge
courier_client_connection_timeout_seconds{otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 3e-08
# HELP courier_client_keep_alive_seconds MQTT keep alive in seconds
# TYPE courier_client_keep_alive_seconds gauge
courier_client_keep_alive_seconds{otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 6e-08
# HELP courier_client_library_version Courier library version
# TYPE courier_client_library_version gauge
courier_client_library_version{otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service",version="%s"} 1
# HELP courier_client_write_timeout_seconds MQTT write timeout in seconds
# TYPE courier_client_write_timeout_seconds gauge
courier_client_write_timeout_seconds{otel_scope_name="github.com/gojek/courier-go/otelcourier",otel_scope_version="semver:%s",service_name="test-service"} 3e-08
`, vsn, vsn, vsn, vsn, vsn, vsn))

		assert.NoError(t, testutil.GatherAndCompare(reg, buf,
			"courier_client_connection_timeout_seconds",
			"courier_client_write_timeout_seconds",
			"courier_client_keep_alive_seconds",
			"courier_client_ack_timeout_seconds",
			"courier_client_library_version",
		))
	})
}

func Test_courierConfigMetrics_NonCourierConfigType(t *testing.T) {
	reg := prom.NewRegistry()
	exporter, err := prometheus.New(prometheus.WithRegisterer(reg))
	require.NoError(t, err)
	mp := metric.NewMeterProvider(metric.WithReader(exporter))

	mw := New("test-service", WithMeterProvider(mp))

	mockClient := &mockUseMiddleware{}

	mw.ApplyMiddlewares(mockClient)

	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		metricName := mf.GetName()
		assert.NotContains(t, metricName, "courier_client_connection_timeout")
		assert.NotContains(t, metricName, "courier_client_write_timeout")
		assert.NotContains(t, metricName, "courier_client_keep_alive")
		assert.NotContains(t, metricName, "courier_client_ack_timeout")
		assert.NotContains(t, metricName, "courier_client_library_version")
	}
}

type mockUseMiddleware struct{}

func (m *mockUseMiddleware) UsePublisherMiddleware(mwf ...courier.PublisherMiddlewareFunc)       {}
func (m *mockUseMiddleware) UseSubscriberMiddleware(mwf ...courier.SubscriberMiddlewareFunc)     {}
func (m *mockUseMiddleware) UseUnsubscriberMiddleware(mwf ...courier.UnsubscriberMiddlewareFunc) {}
func (m *mockUseMiddleware) UseStopMiddleware(mwf courier.StopMiddlewareFunc)                    {}
