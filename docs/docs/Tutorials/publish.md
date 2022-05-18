---
title: Publish Message
description: Tutorial on publishing messages via client
---

Once you have initialised the courier client and established a connection with the broker, you can publish message in the following way.

```go title="publisher.go" {7-13,17}
type chatMessage struct {
	From string      `json:"from"`
	To   string      `json:"to"`
	Data interface{} `json:"data"`
}

msg := &chatMessage{
	From: "test-username-1",
	To:   "test-username-2",
	Data: map[string]string{
		"message": "Hi, User 2!",
	},
}

var client courier.Publisher

_ = client.Publish(context.Background(), "chat/test-username-2/send", msg)
```
