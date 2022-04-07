package courier

import (
	"bytes"
	"context"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Publish allows to publish messages to an MQTT broker
func (c *Client) Publish(ctx context.Context, topic string, message interface{}, opts ...Option) error {
	return c.publisher.Publish(ctx, topic, message, opts...)
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
	return PublisherFunc(func(ctx context.Context, topic string, message interface{}, opts ...Option) (err error) {
		buf := bytes.Buffer{}

		err = c.options.newEncoder(&buf).Encode(message)
		if err != nil {
			return
		}

		o := composeOptions(opts)
		c.execute(func(cc mqtt.Client) {
			t := cc.Publish(topic, o.qos, o.retained, buf.Bytes())
			err = c.handleToken(t, ErrPublishTimeout)
		})

		return
	})
}
