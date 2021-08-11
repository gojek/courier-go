package otelcourier_test

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/oteltest"

	courier "***REMOVED***"
	"***REMOVED***/otelcourier"
)

func ExampleNewMiddleware() {
	tp := oteltest.NewTracerProvider()

	// import "go.opentelemetry.io/otel/sdk/trace"
	//
	// tp := trace.NewTracerProvider()
	// defer tp.Shutdown(context.Background())

	otel.SetTracerProvider(tp)

	c, _ := courier.NewClient()
	otelcourier.InstrumentClient(c, otelcourier.NewMiddleware("service-name"))

	if err := c.Start(); err != nil {
		panic(err)
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, []os.Signal{os.Interrupt, syscall.SIGTERM}...)

	if err := c.Publish(
		context.Background(), "test-topic", courier.QOSOne, false, "message"); err != nil {
		panic(err)
	}
	<-stopCh

	c.Stop()
}
