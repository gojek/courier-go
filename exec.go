package courier

import (
	"context"
	"math/rand"
	"sync/atomic"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/gojekfarm/xtools/generic/slice"
	"github.com/gojekfarm/xtools/generic/xmap"
)

type execOpt interface {
	isExecOpt()
}

type execOptConst int

func (eoc execOptConst) isExecOpt() {}

type execOptFn struct {
	execWithState func(func(mqtt.Client) error, *internalState) error
}

func (eof *execOptFn) isExecOpt() {}

const (
	execAll execOptConst = iota
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

func (c *Client) filterStates(predicate func(*internalState) bool) []*internalState {
	return slice.Filter(
		xmap.Values(c.mqttClients),
		predicate,
	)
}

func (c *Client) filterClients(predicate func(*internalState) bool) []mqtt.Client {
	return slice.Map(c.filterStates(predicate), func(is *internalState) mqtt.Client { return is.client })
}

func (c *Client) execMultiConn(f func(mqtt.Client) error, eo execOpt) error {
	ccs := xmap.Values(c.mqttClients)

	switch eo := eo.(type) {
	case execOptConst:

		if eo == execOneRandom {
			// nolint:errcheck
			p := c.rndPool.Get().(*rand.Rand)
			defer c.rndPool.Put(p)

			return f(ccs[p.Intn(len(ccs))].client)
		}

		if eo == execOneRoundRobin {
			return f(ccs[int(c.rrCounter.next())%len(ccs)].client)
		}

		return slice.Reduce(slice.MapConcurrent(ccs, func(s *internalState) error {
			return f(s.client)
		}), accumulateErrors)
	case *execOptFn:
		return slice.Reduce(slice.MapConcurrent(ccs, func(is *internalState) error {
			c.options.logger.Info(context.Background(), "executing", map[string]any{
				"clientId": clientIDMapper(is.client),
				"flow":     "execute",
			})

			return eo.execWithState(f, is)
		}), accumulateErrors)
	}

	return nil
}
