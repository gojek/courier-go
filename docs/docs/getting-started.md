---
sidebar_position: 1
---

# Getting Started

## Introduction

Courier Golang client library provides an opinionated wrapper over paho MQTT library to add features on top of it.

Long running connection is a persistent connection established between client & server for instant bi-directional communication. A long running connection is maintained for maximum possible duration with the help of keep alive packets. This helps in saving battery and data on mobile devices.

MQTT is an extremely lightweight protocol which works on publish/subscribe messaging model. It is designed for connections with remote locations where a "small code footprint" is required or the network bandwidth is limited.

The protocol usually runs over TCP/IP; however, any network protocol that provides ordered, lossless, bi-directional connections can support MQTT.

MQTT has 3 built-in QoS levels for Reliable Message Delivery:

- QoS 0(At most once) - the message is sent only once and the client and broker take no additional steps to acknowledge delivery (fire and forget).
- QoS 1(At least once) - the message is re-tried by the sender multiple times until acknowledgement is received (acknowledged delivery).
- QoS 2(Exactly once) - the sender and receiver engage in a two-level handshake to ensure only one copy of the message is received (assured delivery).

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
		courier.WithAddress("localhost", 1883),
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
