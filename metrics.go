package courier

import (
	"time"

	"***REMOVED***/metrics"
)

const (
	attemptEvent eventType = 1 << iota
	successEvent
	timeoutEvent
	errorEvent
)

type eventWrapper struct {
	types eventType
}

type eventType uint

func (et eventType) match(e eventType) bool { return et&e != 0 }

func (c *Client) reportEvents(op metrics.Operation, w *eventWrapper, duration time.Duration) {
	go func() {
		c.metricsExchange.updates <- updateEvent{
			op:       op,
			types:    w.types,
			duration: duration,
		}
	}()
}

type updateEvent struct {
	op       metrics.Operation
	types    eventType
	duration time.Duration
}

type metricsExchange struct {
	updates   chan updateEvent
	collector metrics.Collector
}

func (m *metricsExchange) monitor() {
	for u := range m.updates {
		go m.pushMetrics(m.collector, u)
	}
}

func (m *metricsExchange) pushMetrics(c metrics.Collector, e updateEvent) {
	r := metrics.Result{
		OpType:      e.op,
		Attempts:    1,
		RunDuration: e.duration,
	}

	if e.types.match(successEvent) {
		r.Successes = 1
	}

	if e.types.match(errorEvent) {
		r.Errors = 1
	}

	if e.types.match(timeoutEvent) {
		r.Timeouts = 1
	}

	c.Update(r)
}
