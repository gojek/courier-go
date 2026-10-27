// Package otelcourier instruments the github.com/gojek/courier-go package.
//
// With MQTT v5 (courier.WithMQTTVersion(courier.MQTTv5)) the W3C trace context is propagated transparently in
// native MQTT User Properties: the publisher middleware injects traceparent/tracestate with courier.WithUserProperties
// and subscriber callbacks continue the trace from courier.Message.UserProperties, the payload is never modified.
// With MQTT v3.1.1 there are no headers, use WithTextMapCarrierExtractFunc or a propagation.TextMapCarrier message.
package otelcourier // import "github.com/gojek/courier-go/otelcourier"
