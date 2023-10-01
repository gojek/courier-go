package courier

import (
	"math/rand"
	"sync/atomic"

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

type atomicCounter struct {
	value uint64
}

func (ac *atomicCounter) next() uint64 {
	current := atomic.LoadUint64(&ac.value)
	atomic.AddUint64(&ac.value, 1)

	return current
}

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
	ccs := xmap.Values(c.mqttClients)

	if eo == execOneRandom {
		// nolint:errcheck
		p := c.rndPool.Get().(*rand.Rand)
		defer c.rndPool.Put(p)

		return f(ccs[p.Intn(len(ccs))])
	}

	if eo == execOneRoundRobin {
		return f(ccs[int(c.rrCounter.next())%len(ccs)])
	}

	return slice.Reduce(slice.MapConcurrent(ccs, func(cc mqtt.Client) error {
		return f(cc)
	}), accumulateErrors)
}
