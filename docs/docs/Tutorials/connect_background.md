---
title: Background Broker Connect
description: Tutorial on connecting to an MQTT broker
---

Create a new courier client and provide broker address to it.

You can start the client in background with `courier.ExponentialStartStrategy` which will keep trying to connect to the broker until the context is cancelled.

You can wait for the connection with `courier.WaitForConnection` and verify the connection by calling `.IsConnected()` and it should return true.

```go title="background_connect.go" {2,13,15}
c, err := courier.NewClient(
    courier.WithTCPAddress("broker.emqx.io", 1883),
    // courier.WithUsername("username"),
    // courier.WithPassword("password"),
)

if err != nil {
    panic(err)
}

ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

courier.ExponentialStartStrategy(ctx, c)

courier.WaitForConnection(c, 5*time.Second, 100*time.Millisecond)

fmt.Println(c.IsConnected())
```
