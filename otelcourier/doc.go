/*
Package otelcourier instruments the ***REMOVED*** package.

Example:

	package main

	import (
		"context"
		"fmt"
		"os"
		"os/signal"
		"syscall"
		"time"

		"***REMOVED***"
		"***REMOVED***/otelcourier"
	)

	func main() {
		c, err := courier.NewClient(
				courier.WithUsername("username"),
				courier.WithPassword("password"),
				courier.WithTCPAddress("localhost", 1883),
				courier.WithKeepAlive(10*time.Second),
				courier.WithConnectTimeout(10*time.Second),
				courier.WithWriteTimeout(2*time.Second),
				courier.WithMaxReconnectInterval(5*time.Minute),
				courier.WithGracefulShutdownPeriod(time.Minute),
				// courier.WithPersistence(s),   // persistence for qos > 0 use-cases
			)

		if err != nil {
			panic(err)
		}

		otelcourier.InstrumentClient(c, otelcourier.NewMiddleware("service-name"))

		if err := c.Start(); err != nil {
			panic(err)
		}

		stopCh := make(chan os.Signal)
		signal.Notify(stopCh, []os.Signal{os.Interrupt, syscall.SIGTERM}...)

		go func() {
			tick := time.NewTicker(time.Second)
			for {
				select {
				case t := <-tick.C:
					msg := map[string]interface{}{
						"time": t.UnixNano(),
					}
					if err := c.Publish(context.Background(), "topic", courier.QOSOne, false, msg); err != nil {
						fmt.Printf("Publish() error = %s\n", err)
					} else {
						fmt.Println("Publish() success")
					}
				case <-stopCh:
					tick.Stop()
					return
				}
			}
		}()

		<-stopCh

		c.Stop()
	}

*/
package otelcourier // import "***REMOVED***/otelcourier"
