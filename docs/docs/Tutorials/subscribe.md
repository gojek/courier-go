---
title: Subscribe Messages
description: Tutorial on subscribing to messages via client
---

You can add a subscription to a topic and respond to message as well when you receive a message.

For example, you can relay the `received_at` time back to the sender by publishing a message. 

```go title="subscriber.go" {11-18,22}
type chatMessage struct {
    From string      `json:"from"`
    To   string      `json:"to"`
    Data interface{} `json:"data"`
}

type status struct {
    ReceivedAt time.Time `json:"received_at"`
}

cb := func(ctx context.Context, ps courier.PubSub, m *courier.Message) {
    msg := new(chatMessage)
    if err := m.DecodePayload(msg); err != nil {
        // Log Error or Panic
    }

    _ = ps.Publish(ctx, fmt.Sprintf("chat/%s/send", msg.From), &status{ReceivedAt: time.Now()})
}

var client courier.Subscriber

_ = client.Subscribe(context.Background(), "chat/test-username-2/send", cb)
```
