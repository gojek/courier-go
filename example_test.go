package courier_test

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gojek/courier-go"
)

func ExampleNewClient() {
	c, err := courier.NewClient(
		courier.WithUsername("username"),
		courier.WithPassword("password"),
		courier.WithTCPAddress("localhost", 1883),
	)

	if err != nil {
		panic(err)
	}

	if err := c.Start(); err != nil {
		panic(err)
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, []os.Signal{os.Interrupt, syscall.SIGTERM}...)

	go func() {
		tick := time.NewTicker(time.Second)
		for {
			select {
			case t := <-tick.C:
				msg := map[string]interface{}{
					"time": t.UnixNano(),
				}
				if err := c.Publish(context.Background(), "topic", msg, courier.QOSOne); err != nil {
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
