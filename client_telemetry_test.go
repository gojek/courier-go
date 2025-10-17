package courier

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_readSubscriptionMeta(t *testing.T) {
	c := &Client{
		subMu: sync.RWMutex{},
		subscriptions: map[string]*subscriptionMeta{
			"topic1": {
				topic:   "topic1",
				options: []Option{QOSOne},
			},
			"topic2": {
				topic: "topic2",
			},
			"topic3": {
				topic:   "topic3",
				options: []Option{QOSTwo},
			},
		},
	}

	subs := c.readSubscriptionMeta()

	assert.Equal(t, map[string]QOSLevel{
		"topic1": 1,
		"topic2": 0,
		"topic3": 2,
	}, subs)
}

func TestClient_clientInfo(t *testing.T) {
	t.Run("single connection mode", func(t *testing.T) {
		c := &Client{options: &clientOptions{}}
		ci := c.clientInfo()
		assert.Equal(t, []MQTTClientInfo(nil), ci)
	})

	t.Run("multi connection mode", func(t *testing.T) {
		c := &Client{options: &clientOptions{multiConnectionMode: true}}
		ci := c.clientInfo()
		assert.Equal(t, []MQTTClientInfo(nil), ci)
	})

	t.Run("pool mode", func(t *testing.T) {
		c := &Client{options: &clientOptions{poolEnabled: true}}
		ci := c.clientInfo()
		assert.Equal(t, []MQTTClientInfo(nil), ci)
	})
}
