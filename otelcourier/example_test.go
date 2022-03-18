package otelcourier_test

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"

	courier "github.com/gojekfarm/courier-go"
	"github.com/gojekfarm/courier-go/otelcourier"
)

func ExampleNewTracer() {
	tp := trace.NewTracerProvider()
	defer tp.Shutdown(context.Background())

	otel.SetTracerProvider(tp)

	c, _ := courier.NewClient()
	otelcourier.NewTracer("service-name").ApplyTraceMiddlewares(c)

	if err := c.Start(); err != nil {
		panic(err)
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, []os.Signal{os.Interrupt, syscall.SIGTERM}...)

	if err := c.Publish(
		context.Background(), "test-topic", "message", courier.QOSOne); err != nil {
		panic(err)
	}
	<-stopCh

	c.Stop()
}
