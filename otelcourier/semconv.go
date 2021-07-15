package otelcourier

import (
	"go.opentelemetry.io/otel/attribute"
)

const (
	// MQTTTopic is the attribute key for tracing message topic
	MQTTTopic = attribute.Key("mqtt.topic")
	// MQTTQoS is the attribute key for tracing message qos
	MQTTQoS = attribute.Key("mqtt.qos")
	// MQTTTopicWithQoS is the attribute key for tracing message topic and qos together
	MQTTTopicWithQoS = attribute.Key("mqtt.topicwithqos")
	// MQTTRetained is the attribute key for tracing message retained flag
	MQTTRetained = attribute.Key("mqtt.retained")
)
