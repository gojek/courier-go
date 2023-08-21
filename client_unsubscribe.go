package courier

import (
	"context"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Unsubscribe removes any subscription to messages from an MQTT broker
func (c *Client) Unsubscribe(ctx context.Context, topics ...string) error {
	if err := c.unsubscriber.Unsubscribe(ctx, topics...); err != nil {
		return err
	}

	c.subMu.Lock()
	for _, topic := range topics {
		delete(c.subscriptions, topic)
	}
	c.subMu.Unlock()

	return nil
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
	return UnsubscriberFunc(func(ctx context.Context, topics ...string) (err error) {
		if e := c.execute(func(cc mqtt.Client) {
			t := cc.Unsubscribe(topics...)
			err = c.handleToken(ctx, t, ErrUnsubscribeTimeout)
		}); e != nil {
			err = e
		}

		return
	})
}
