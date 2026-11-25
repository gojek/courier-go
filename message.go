package courier

// Message represents the entity that is being relayed via the courier MQTT brokers from Publisher(s) to Subscriber(s).
type Message struct {
	ID        int
	Topic     string
	Duplicate bool
	Retained  bool
	QoS       QOSLevel

	// MQTT v5 only, empty when the connection uses MQTT v3.1.1 or the publisher did not set them.
	// UserProperties keeps the last value for a repeated key.
	UserProperties  map[string]string
	ContentType     string
	ResponseTopic   string
	CorrelationData []byte

	payloadDecoder Decoder
}

// NewMessageWithDecoder is a helper to create Message, ideally payloadDecoder should not be mutated once created.
func NewMessageWithDecoder(
	payloadDecoder Decoder,
) *Message {
	return &Message{
		payloadDecoder: payloadDecoder,
	}
}

// DecodePayload can decode the message payload bytes into the desired object.
func (m *Message) DecodePayload(v interface{}) error {
	return m.payloadDecoder.Decode(v)
}
