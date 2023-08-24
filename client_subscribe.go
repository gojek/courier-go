package courier

import (
	"bytes"
	"context"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/gojekfarm/xtools/generic/slice"
)

// Subscribe allows to subscribe to messages from an MQTT broker
func (c *Client) Subscribe(ctx context.Context, topic string, callback MessageHandler, opts ...Option) error {
	if err := c.subscriber.Subscribe(ctx, topic, callback, opts...); err != nil {
		return err
	}

	c.subMu.Lock()
	c.subscriptions[topic] = &subscriptionMeta{
		topic:    topic,
		options:  opts,
		callback: callback,
	}
	c.subMu.Unlock()

	return nil
}

// SubscribeMultiple allows to subscribe to messages on multiple topics from an MQTT broker
func (c *Client) SubscribeMultiple(
	ctx context.Context,
	topicsWithQos map[string]QOSLevel,
	callback MessageHandler,
) error {
	if err := c.subscriber.SubscribeMultiple(ctx, topicsWithQos, callback); err != nil {
		return err
	}

	c.subMu.Lock()

	for topic := range topicsWithQos {
		c.subscriptions[topic] = &subscriptionMeta{
			topic:    topic,
			options:  []Option{topicsWithQos[topic]},
			callback: callback,
		}
	}

	c.subMu.Unlock()

	return nil
}

// UseSubscriberMiddleware appends a SubscriberMiddlewareFunc to the chain.
// Middleware can be used to intercept or otherwise modify, process or skip subscriptions.
// They are executed in the order that they are applied to the Client.
func (c *Client) UseSubscriberMiddleware(mwf ...SubscriberMiddlewareFunc) {
	for _, fn := range mwf {
		c.sMiddlewares = append(c.sMiddlewares, fn)
	}

	c.subscriber = subscriberFuncs(c)

	for i := len(c.sMiddlewares) - 1; i >= 0; i-- {
		c.subscriber = c.sMiddlewares[i].Middleware(c.subscriber)
	}
}

type subscriptionMeta struct {
	topic    string
	options  []Option
	callback MessageHandler
}

func subscriberFuncs(c *Client) Subscriber {
	return NewSubscriberFuncs(
		func(ctx context.Context, topic string, callback MessageHandler, opts ...Option) error {
			o := composeOptions(opts)

			eo := execOneRandom
			if c.options.sharedSubscriptionPredicate(topic) {
				eo = execAll
			}

			return c.execute(func(cc mqtt.Client) error {
				return c.handleToken(ctx, cc.Subscribe(topic, o.qos, callbackWrapper(c, callback)), ErrSubscribeTimeout)
			}, eo)
		},
		func(ctx context.Context, topicsWithQos map[string]QOSLevel, callback MessageHandler) error {
			sharedSubs, normalSubs := filterSubs(topicsWithQos, c.options.sharedSubscriptionPredicate)

			execs := make([]func(context.Context) error, 0, 2)

			if len(sharedSubs) > 0 {
				execs = append(execs, func(ctx context.Context) error {
					return c.execute(func(cc mqtt.Client) error {
						return c.handleToken(ctx, cc.SubscribeMultiple(
							sharedSubs,
							callbackWrapper(c, callback),
						), ErrSubscribeMultipleTimeout)
					}, execAll)
				})
			}

			if len(normalSubs) > 0 {
				execs = append(execs, func(ctx context.Context) error {
					return c.execute(func(cc mqtt.Client) error {
						return c.handleToken(ctx, cc.SubscribeMultiple(
							normalSubs,
							callbackWrapper(c, callback),
						), ErrSubscribeMultipleTimeout)
					}, execOneRandom)
				})
			}

			return slice.Reduce(slice.MapConcurrentWithContext(ctx, execs,
				func(ctx context.Context, f func(context.Context) error) error { return f(ctx) },
			), accumulateErrors)
		},
	)
}

func callbackWrapper(c *Client, callback MessageHandler) mqtt.MessageHandler {
	return func(_ mqtt.Client, m mqtt.Message) {
		ctx := context.Background()

		msg := NewMessageWithDecoder(
			c.options.newDecoder(ctx, bytes.NewReader(m.Payload())),
		)
		msg.ID = int(m.MessageID())
		msg.Topic = m.Topic()
		msg.Duplicate = m.Duplicate()
		msg.Retained = m.Retained()
		msg.QoS = QOSLevel(m.Qos())

		callback(ctx, c, msg)
	}
}

func filterSubs(topicsWithQos map[string]QOSLevel, predicate func(string) bool) (map[string]byte, map[string]byte) {
	sharedSubs, normalSubs := make(map[string]byte), make(map[string]byte)

	for topic, qosLevel := range topicsWithQos {
		if predicate(topic) {
			sharedSubs[topic] = byte(qosLevel)

			continue
		}

		normalSubs[topic] = byte(qosLevel)
	}

	return sharedSubs, normalSubs
}

func routeFilters(topicsWithQos map[string]QOSLevel) map[string]byte {
	m := make(map[string]byte)

	for topic, q := range topicsWithQos {
		m[topic] = byte(q)
	}

	return m
}
