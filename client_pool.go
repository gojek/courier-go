package courier

import (
	"fmt"

	mqtt "github.com/gojek/paho.mqtt.golang"
	"github.com/gojekfarm/xtools/generic"
)

func (c *Client) initializeConnectionPool() {
	for i := 0; i < c.options.poolSize; i++ {
		mqttOpts := toClientOptions(c, c.options, fmt.Sprintf("-%d", i))
		mqttClient := newClientFunc.Load().(func(*mqtt.ClientOptions) mqtt.Client)(mqttOpts)

		poolID := fmt.Sprintf("%s-%d", c.options.clientID, i)
		c.mqttClients[poolID] = &internalState{
			client:     mqttClient,
			subsCalled: make(generic.Set[string]),
		}
	}
}
