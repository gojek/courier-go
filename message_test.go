package courier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_Ack(t *testing.T) {
	m := NewMessageWithDecoder(nil)
	m.Ack() // no-op when auto ack is on

	acked := false
	m.ack = func() { acked = true }
	m.Ack()

	assert.True(t, acked)
}
