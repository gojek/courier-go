package courier

import (
	"context"
	"time"

	"***REMOVED***/metrics"
)

// Unsubscribe removes any subscription to messages from an MQTT broker
func (c *Client) Unsubscribe(ctx context.Context, topics ...string) error {
	return c.unsubscriber.Unsubscribe(ctx, topics...)
}

// UseUnsubscriberMiddleware appends a UnsubscriberMiddlewareFunc to the chain.
// Middleware can be used to intercept or otherwise modify, process or skip subscriptions.
// They are executed in the order that they are applied to the Client.
func (c *Client) UseUnsubscriberMiddleware(mwf ...UnsubscriberMiddlewareFunc) {
	for _, fn := range mwf {
		c.usMiddlewares = append(c.usMiddlewares, fn)
	}

	c.unsubscriber = unsubscriberHandler(c)

	for i := len(c.usMiddlewares) - 1; i >= 0; i-- {
		c.unsubscriber = c.usMiddlewares[i].Middleware(c.unsubscriber)
	}
}

func unsubscriberHandler(c *Client) Unsubscriber {
	return UnsubscriberFunc(func(ctx context.Context, topics ...string) error {
		w := &eventWrapper{types: attemptEvent}
		defer func(begin time.Time) {
			c.reportEvents(metrics.UnsubscribeOp, w, time.Since(begin))
		}(time.Now())

		t := c.mqttClient.Unsubscribe(topics...)

		return c.handleToken(t, w, ErrUnsubscribeTimeout)
	})
}
