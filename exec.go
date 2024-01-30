package courier

import (
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
	onExec    func(*internalState, error)
	predicate func(*internalState) bool
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
	switch eo := eo.(type) {
	case execOptConst:
		ccs := c.filterClients(func(is *internalState) bool { return true })

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
	case *execOptFn:
		return slice.Reduce(slice.MapConcurrent(c.filterStates(eo.predicate), func(is *internalState) error {
			err := f(is.client)
			eo.onExec(is, err)
			return err
		}), accumulateErrors)
	}

	return nil
}
