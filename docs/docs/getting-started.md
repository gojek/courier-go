---
sidebar_position: 1
---

# Getting Started

## Introduction

Courier Golang client library provides an opinionated wrapper over paho MQTT library to add features on top of it.

### Usage

```bash
go get -u github.com/gojek/courier-go
```

Create a `main.go` file and add the following code to it.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/gojek/courier-go"
)

func main() {
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

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

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
		case <-ctx.Done():
			tick.Stop()
			c.Stop()
		}
	}
}
```
