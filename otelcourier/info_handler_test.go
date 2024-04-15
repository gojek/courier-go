package otelcourier

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"testing"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	pcm "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/gojek/courier-go"
)

var defOpts []courier.ClientOption

func init() {
	brokerAddress := os.Getenv("BROKER_ADDRESS") // host:port format
	if len(brokerAddress) == 0 {
		brokerAddress = "localhost:1883"
	}

	list := strings.Split(brokerAddress, ":")
	p, _ := strconv.Atoi(list[1])

	defOpts = append(defOpts, courier.WithAddress(list[0], uint16(p)), courier.WithClientID("clientID"))
}

func TestConnectedClientMetric(t *testing.T) {
	t.Parallel()

	reg := prom.NewRegistry()
	exporter, err := prometheus.New(prometheus.WithRegisterer(reg))
	require.NoError(t, err)
	mp := metric.NewMeterProvider(metric.WithReader(exporter))

	c, err := courier.NewClient(defOpts...)
	_ = New("test-service", WithMeterProvider(mp), WithInfoHandlerFrom(c))
	assert.NoError(t, err)

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	errCh := make(chan error, 1)

	go func() {
		errCh <- c.Run(ctx)
	}()

	if !courier.WaitForConnection(c, 3*time.Second, 100*time.Millisecond) {
		t.Fatal("client did not connect")
	}

	<-time.After(1500 * time.Millisecond)

	got, err := reg.Gather()
	assert.NoError(t, err)

	found := false
	var metricFamily *pcm.MetricFamily
	for _, mf := range got {
		if mf.GetName() == "courier_client_connected" {
			found = true
			metricFamily = mf
			break
		}
	}

	assert.True(t, found)
	assert.Len(t, metricFamily.GetMetric(), 1)

	m := metricFamily.GetMetric()[0]
	assert.EqualValues(t, 1, m.GetGauge().GetValue())

	labelFound := false
	for _, lp := range m.GetLabel() {
		if lp.GetName() == "mqtt_clientid" {
			assert.Equal(t, "clientID", lp.GetValue())
			labelFound = true
		}
	}

	assert.True(t, labelFound)
}

func Test_boolInt64(t *testing.T) {
	tests := []struct {
		name      string
		connected bool
		want      int64
	}{
		{
			name:      "connected",
			connected: true,
			want:      1,
		},
		{
			name:      "not connected",
			connected: false,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValuesf(t, tt.want, boolInt64(tt.connected), "boolInt64(%v)", tt.connected)
		})
	}
}
