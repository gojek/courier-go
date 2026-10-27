package courier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gojek/courier-go/internal/driver"
	mqtt "github.com/gojek/paho.mqtt.golang"
)

type v5Message struct {
	mqtt.Message
	props *driver.PublishProps
}

func (m v5Message) Topic() string               { return "t" }
func (m v5Message) Qos() byte                   { return 1 }
func (m v5Message) Duplicate() bool             { return false }
func (m v5Message) Retained() bool              { return false }
func (m v5Message) MessageID() uint16           { return 7 }
func (m v5Message) Payload() []byte             { return []byte(`"body"`) }
func (m v5Message) Props() *driver.PublishProps { return m.props }

func TestMessage_V5Fields(t *testing.T) {
	c, err := NewClient(WithAddress("127.0.0.1", 1883))
	require.NoError(t, err)

	var got *Message

	cb := callbackWrapper(c, func(_ context.Context, _ PubSub, m *Message) { got = m })

	cb(nil, v5Message{props: &driver.PublishProps{
		UserProperties:  map[string]string{"a": "b"},
		ContentType:     "application/json",
		ResponseTopic:   "reply",
		CorrelationData: []byte("id"),
	}})

	assert.Equal(t, map[string]string{"a": "b"}, got.UserProperties)
	assert.Equal(t, "application/json", got.ContentType)
	assert.Equal(t, "reply", got.ResponseTopic)
	assert.Equal(t, []byte("id"), got.CorrelationData)

	var body string
	require.NoError(t, got.DecodePayload(&body))
	assert.Equal(t, "body", body)

	// v3.1.1 style message or no properties: v5 fields stay empty and lookups do not panic
	cb(nil, v5Message{})
	assert.Nil(t, got.UserProperties)
	assert.Empty(t, got.UserProperties["missing"])
	assert.Empty(t, got.ContentType)
}
