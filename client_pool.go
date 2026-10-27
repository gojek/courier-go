package courier

import (
	"fmt"

	"github.com/gojekfarm/xtools/generic"
)

func (c *Client) initializeConnectionPool() {
	c.mqttClients = make(map[string]*internalState, c.options.poolSize)
	c.orderedClients = make([]*internalState, 0, c.options.poolSize)

	for i := 0; i < c.options.poolSize; i++ {
		mqttOpts := toClientOptions(c, c.options, fmt.Sprintf("-%d", i))
		mqttClient := newMQTTClient(c.options, mqttOpts)

		poolID := fmt.Sprintf("%s-%d", c.options.clientID, i)
		ic := &internalState{
			client:     mqttClient,
			subsCalled: make(generic.Set[string]),
		}
		c.mqttClients[poolID] = ic
		c.orderedClients = append(c.orderedClients, ic)
	}
}
