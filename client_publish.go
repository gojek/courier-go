package courier

import (
	"bytes"
	"time"

	"***REMOVED***/metrics"
)

// Publish allows to publish messages to an MQTT broker
func (c *Client) Publish(topic string, qos QOSLevel, retained bool, message interface{}) error {
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
}
