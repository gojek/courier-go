package courier

import (
	"bytes"
	"time"
)

// Publish allows to publish messages to an MQTT broker
func (c *Client) Publish(topic string, qos QOSLevel, retained bool, message interface{}) error {
	pm := c.options.metricsCollector.Publish()
	defer func(begin time.Time) { pm.RunDuration.Observe(time.Since(begin).Seconds()) }(time.Now())

	buf := bytes.Buffer{}
	err := c.options.newEncoder(&buf).Encode(message)
	if err != nil {
		return err
	}
	pm.Attempts.Add(1)
	t := c.mqttClient.Publish(topic, byte(qos), retained, buf.Bytes())
	return c.handleToken(t, pm, ErrPublishTimeout)
}
