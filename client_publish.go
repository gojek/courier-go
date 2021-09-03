package courier

import (
	"bytes"
	"context"
	"time"

	"***REMOVED***/metrics"
)

// Publish allows to publish messages to an MQTT broker
func (c *Client) Publish(ctx context.Context, topic string, qos QOSLevel, retained bool, message interface{}) error {
	return c.publisher.Publish(ctx, topic, qos, retained, message)
}

// UsePublisherMiddleware appends a PublisherMiddlewareFunc to the chain.
// Middleware can be used to intercept or otherwise modify, process or skip messages.
// They are executed in the order that they are applied to the Client.
func (c *Client) UsePublisherMiddleware(mwf ...PublisherMiddlewareFunc) {
	for _, fn := range mwf {
		c.pMiddlewares = append(c.pMiddlewares, fn)
	}

	c.publisher = publishHandler(c)

	for i := len(c.pMiddlewares) - 1; i >= 0; i-- {
		c.publisher = c.pMiddlewares[i].Middleware(c.publisher)
	}
}

func publishHandler(c *Client) Publisher {
	return PublisherFunc(func(ctx context.Context, topic string, qos QOSLevel, retained bool, message interface{}) error {
		w := &eventWrapper{types: attemptEvent}
		defer func(begin time.Time) {
			c.reportEvents(metrics.PublishOp, w, time.Since(begin))
		}(time.Now())

		buf := bytes.Buffer{}
		err := c.options.newEncoder(&buf).Encode(message)
		if err != nil {
			w.types |= errorEvent

			return err
		}
		t := c.mqttClient.Publish(topic, byte(qos), retained, buf.Bytes())

		return c.handleToken(t, w, ErrPublishTimeout)
	})
}
