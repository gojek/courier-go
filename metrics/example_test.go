package metrics_test

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	courier "***REMOVED***"
	"***REMOVED***/metrics"
)

func ExampleNewPrometheus() {
	reg := prometheus.NewRegistry()
	m := metrics.NewPrometheus()
	if err := m.AddToRegistry(reg); err != nil {
		panic(err)
	}

	c, err := courier.NewClient(
		courier.WithCustomMetrics(m),
	)

	if err != nil {
		panic(err)
	}

	metricsServer := http.Server{
		Addr:    ":9090",
		Handler: promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
	}
	go func() {
		_ = metricsServer.ListenAndServe()
	}()

	if err := c.Start(); err != nil {
		panic(err)
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, []os.Signal{os.Interrupt, syscall.SIGTERM}...)

	<-stopCh

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = metricsServer.Shutdown(stopCtx)
	c.Stop()
}
