package courier

import (
	"fmt"

	mqtt "github.com/gojek/paho.mqtt.golang"
	"github.com/gojekfarm/xtools/generic"
)

type pooledConnection struct {
	client mqtt.Client
	id     string
	state  *internalState
}

func newPooledConnection(client mqtt.Client, id string) *pooledConnection {
	pc := &pooledConnection{
		client: client,
		id:     id,
		state: &internalState{
			client:     client,
			subsCalled: make(generic.Set[string]),
		},
	}

	return pc
}

func (c *Client) initializeConnectionPool() {
	for i := 0; i < c.options.poolSize; i++ {
		mqttOpts := toClientOptions(c, c.options, fmt.Sprintf("-%d", i))

		mqttClient := newClientFunc.Load().(func(*mqtt.ClientOptions) mqtt.Client)(mqttOpts)
		pooledConn := newPooledConnection(mqttClient, fmt.Sprintf("%d", i))

		c.connectionPool = append(c.connectionPool, pooledConn)
	}
}

func (c *Client) getNextPoolConnection() *pooledConnection {
	if len(c.connectionPool) == 0 {
		return nil
	}

	index := int(c.poolIndex.next()) % len(c.connectionPool)

	return c.connectionPool[index]
}
