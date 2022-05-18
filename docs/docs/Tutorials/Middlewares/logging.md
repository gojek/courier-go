---
title: Logging
description: Tutorial on writing a logging middleware
---

A logging middleware is helpful to give information whenever you invoke `client.Publish()`, the call is first passed through the chain of middlewares and then published to broker.

```go title="publish_logger.go" {12,15-16}
type chatMessage struct {
    From string      `json:"from"`
    To   string      `json:"to"`
    Data interface{} `json:"data"`
}

var client *courier.Client

client.UsePublisherMiddleware(func(next courier.Publisher) courier.Publisher {
    return courier.PublisherFunc(func(ctx context.Context, topic string, data interface{}, opts ...courier.Option) error {
        if msg, ok := data.(*chatMessage); ok {
            log.Printf("Sending message from %s to %s", msg.From, msg.To)
        }

        if err := next.Publish(ctx, topic, data, opts...); err != nil {
            log.Printf("err sending message: %s", err)

            return err
        }

        return nil
    })
})
```
