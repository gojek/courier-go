package courier

import (
	"bytes"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func callbackWrapper(c *Client, callback MessageHandler) mqtt.MessageHandler {
	return func(_ mqtt.Client, message mqtt.Message) {
		defer func(begin time.Time) {
			c.options.metricsCollector.CallbackOp().
				RunDuration.Observe(time.Since(begin).Seconds())
		}(time.Now())

		d := c.options.newDecoder(bytes.NewReader(message.Payload()))
		callback(c, d)
	}
}

// Subscribe allows to subscribe to messages from an MQTT broker
func (c *Client) Subscribe(topic string, qos QOSLevel, callback MessageHandler) error {
	sm := c.options.metricsCollector.Subscribe()
	defer func(begin time.Time) { sm.RunDuration.Observe(time.Since(begin).Seconds()) }(time.Now())

	sm.Attempts.Add(1)
	t := c.mqttClient.Subscribe(topic, byte(qos), callbackWrapper(c, callback))
	return c.handleToken(t, sm, ErrSubscribeTimeout)
}

// SubscribeMultiple allows to subscribe to messages on multiple topics from an MQTT broker
func (c *Client) SubscribeMultiple(topicsWithQos map[string]QOSLevel, callback MessageHandler) error {
	smm := c.options.metricsCollector.SubscribeMultiple()
	defer func(begin time.Time) { smm.RunDuration.Observe(time.Since(begin).Seconds()) }(time.Now())

	smm.Attempts.Add(1)
	t := c.mqttClient.SubscribeMultiple(routeFilters(topicsWithQos), callbackWrapper(c, callback))
	return c.handleToken(t, smm, ErrSubscribeMultipleTimeout)
}

// Unsubscribe removes any subscription to messages from an MQTT broker
func (c *Client) Unsubscribe(topics ...string) error {
	um := c.options.metricsCollector.Unsubscribe()
	defer func(begin time.Time) { um.RunDuration.Observe(time.Since(begin).Seconds()) }(time.Now())

	um.Attempts.Add(1)
	t := c.mqttClient.Unsubscribe(topics...)
	return c.handleToken(t, um, ErrUnsubscribeTimeout)
}

func routeFilters(topicsWithQos map[string]QOSLevel) map[string]byte {
	m := make(map[string]byte)
	for topic, q := range topicsWithQos {
		m[topic] = byte(q)
	}
	return m
}
