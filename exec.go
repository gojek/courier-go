package courier

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/gojekfarm/xtools/generic"
	"github.com/gojekfarm/xtools/generic/slice"
	"github.com/gojekfarm/xtools/generic/xmap"
)

var errInvalidExecOpt = errors.New("courier: invalid exec option")

type internalState struct {
	subsCalled generic.Set[string]
	client     mqtt.Client
	mu         sync.Mutex
}

type execOpt interface{ isExecOpt() }

type execOptConst int

func (eoc execOptConst) isExecOpt() {}

type execOptWithState func(func(mqtt.Client) error, *internalState) error

func (eof execOptWithState) isExecOpt() {}

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
	case execOptWithState:
		return slice.Reduce(slice.MapConcurrent(ccs, func(s *internalState) error { return eo(f, s) }), accumulateErrors)
	default:
		return errInvalidExecOpt
	}
}
