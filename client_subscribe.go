package courier

import (
	"bytes"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"***REMOVED***/metrics"
)

func callbackWrapper(c *Client, callback MessageHandler) mqtt.MessageHandler {
	return func(_ mqtt.Client, message mqtt.Message) {
		w := &eventWrapper{types: attemptEvent}
		defer func(begin time.Time) {
			c.reportEvents(metrics.CallbackOp, w, time.Since(begin))
		}(time.Now())

		d := c.options.newDecoder(bytes.NewReader(message.Payload()))
		callback(c, d)
		w.types |= successEvent
	}
}

// Subscribe allows to subscribe to messages from an MQTT broker
func (c *Client) Subscribe(topic string, qos QOSLevel, callback MessageHandler) error {
	w := &eventWrapper{types: attemptEvent}
	defer func(begin time.Time) {
		c.reportEvents(metrics.SubscribeOp, w, time.Since(begin))
	}(time.Now())

	t := c.mqttClient.Subscribe(topic, byte(qos), callbackWrapper(c, callback))
	return c.handleToken(t, w, ErrSubscribeTimeout)
}

// SubscribeMultiple allows to subscribe to messages on multiple topics from an MQTT broker
func (c *Client) SubscribeMultiple(topicsWithQos map[string]QOSLevel, callback MessageHandler) error {
	w := &eventWrapper{types: attemptEvent}
	defer func(begin time.Time) {
		c.reportEvents(metrics.SubscribeMultipleOp, w, time.Since(begin))
	}(time.Now())

	t := c.mqttClient.SubscribeMultiple(routeFilters(topicsWithQos), callbackWrapper(c, callback))
	return c.handleToken(t, w, ErrSubscribeMultipleTimeout)
}

// Unsubscribe removes any subscription to messages from an MQTT broker
func (c *Client) Unsubscribe(topics ...string) error {
	w := &eventWrapper{types: attemptEvent}
	defer func(begin time.Time) {
		c.reportEvents(metrics.UnsubscribeOp, w, time.Since(begin))
	}(time.Now())

	t := c.mqttClient.Unsubscribe(topics...)
	return c.handleToken(t, w, ErrUnsubscribeTimeout)
}

func routeFilters(topicsWithQos map[string]QOSLevel) map[string]byte {
	m := make(map[string]byte)
	for topic, q := range topicsWithQos {
		m[topic] = byte(q)
	}
	return m
}
