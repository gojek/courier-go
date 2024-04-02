package otelcourier_test

import (
	"context"
	"os"
	"os/signal"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/gojek/courier-go"
	"github.com/gojek/courier-go/otelcourier"
)

func ExampleNew() {
	tp := trace.NewTracerProvider()
	defer tp.Shutdown(context.Background())

	exporter, err := prometheus.New(
	/* Add a non-default prometheus registry here with `prometheus.WithRegisterer` option, if needed. */
	)
	if err != nil {
		panic(err)
	}
	mp := metric.NewMeterProvider(metric.WithReader(exporter))

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(&propagation.TraceContext{})

	metricLabelMapper := otelcourier.TopicAttributeTransformer(func(ctx context.Context, topic string) string {
		if strings.HasPrefix(topic, "test") {
			return "test"
		}

		return "other"
	})

	c, _ := courier.NewClient()
	otelcourier.New("service-name", metricLabelMapper).ApplyMiddlewares(c)

	if err := c.Start(); err != nil {
		panic(err)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	if err := c.Publish(
		context.Background(), "test-topic", "message", courier.QOSOne); err != nil {
		panic(err)
	}

	if err := c.Publish(
		context.Background(), "other-topic", "message", courier.QOSOne); err != nil {
		panic(err)
	}

	// Here, you can expose the metrics at /metrics endpoint for prometheus.DefaultRegisterer.

	<-ctx.Done()

	c.Stop()
}
