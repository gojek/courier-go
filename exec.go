package courier

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/gojekfarm/xtools/generic/slice"
	"github.com/gojekfarm/xtools/generic/xmap"
)

type execOpt int

const (
	execAll execOpt = iota
	execOneRandom
	execOneRoundRobin
)

func (c *Client) execute(f func(mqtt.Client) error, eo execOpt) error {
	c.clientMu.RLock()
	defer c.clientMu.RUnlock()

	if c.mqttClient == nil && len(c.mqttClients) == 0 {
		return ErrClientNotInitialized
	}

	if c.options.multiConnectionMode {
		return c.execMultiConn(f, eo)
	}

	return f(c.mqttClient)
}

func (c *Client) execMultiConn(f func(mqtt.Client) error, eo execOpt) error {
	if eo == execOneRandom {
		return f(xmap.Values(c.mqttClients)[c.rnd.Intn(len(c.mqttClients))])
	}

	// TODO: Choose client in round robin fashion and execute
	// if eo == execOneRoundRobin {
	// }

	return slice.Reduce(slice.MapConcurrent(xmap.Values(c.mqttClients), func(cc mqtt.Client) error {
		return f(cc)
	}), accumulateErrors)
}
