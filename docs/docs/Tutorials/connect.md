---
title: Connect to Broker
description: Tutorial on connecting to an MQTT broker
---

Create a new courier client and provide broker address to it. Upon calling `.Start()` the client will attempt to connect to the broker.

You can verify the connection by calling `.IsConnected()` and it should return true.

```go title="connect.go" {2,11,15}
c, err := courier.NewClient(
    courier.WithAddress("broker.emqx.io", 1883),
    // courier.WithUsername("username"),
    // courier.WithPassword("password"),
)

if err != nil {
    panic(err)
}

if err := c.Start(); err != nil {
    panic(err)
}

fmt.Println(c.IsConnected())
```
