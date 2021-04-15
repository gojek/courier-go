package metrics

import (
	"fmt"
	"sync"

	gokitprom "github.com/go-kit/kit/metrics/prometheus"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	subsystemPrefix         = "courier"
	subsystemPublish        = subsystemPrefix + "_publish"
	subsystemUnsubscribe    = subsystemPrefix + "_unsubscribe"
	subsystemSubscribe      = subsystemPrefix + "_subscribe"
	subsystemSubscribeMulti = subsystemPrefix + "_subscribe_multiple"
	subsystemCallback       = subsystemPrefix + "_callback"

	metricSuccesses   = "successes"
	metricAttempts    = "attempts"
	metricErrors      = "errors"
	metricTimeouts    = "timeouts"
	metricRunDuration = "run_duration"
)

var (
	opSubsystemMap = map[Operation]string{
		PublishOp:           subsystemPublish,
		SubscribeOp:         subsystemSubscribe,
		SubscribeMultipleOp: subsystemSubscribeMulti,
		UnsubscribeOp:       subsystemUnsubscribe,
		CallbackOp:          subsystemCallback,
	}

	counters   = []string{metricSuccesses, metricAttempts, metricErrors, metricTimeouts}
	histograms = []string{metricRunDuration}
)

// NewPrometheus creates a PrometheusMetrics instance which implements the Metrics interface
func NewPrometheus() *PrometheusMetrics {
	return &PrometheusMetrics{
		operationMap: map[Operation]*Aggregator{
			PublishOp:           {},
			SubscribeOp:         {},
			SubscribeMultipleOp: {},
			UnsubscribeOp:       {},
			CallbackOp:          {},
		},
	}
}

// PrometheusMetrics is a prometheus collector for courier.Client operations
type PrometheusMetrics struct {
	sync.RWMutex
	operationMap map[Operation]*Aggregator
}

func (p *PrometheusMetrics) Update(r Result) {
	p.RWMutex.Lock()
	defer p.RWMutex.Unlock()

	a := p.operationMap[r.OpType]

	if r.Attempts > 0 && a.Attempts != nil {
		a.Attempts.Add(1)
	}
	if r.Timeouts > 0 && a.Timeouts != nil {
		a.Timeouts.Add(1)
	}
	if r.Errors > 0 && a.Errors != nil {
		a.Errors.Add(1)
	}
	if r.Successes > 0 && a.Successes != nil {
		a.Successes.Add(1)
	}
	if r.RunDuration > 0 && a.RunDuration != nil {
		a.RunDuration.Observe(r.RunDuration.Seconds())
	}
}

// AddToRegistry is used to register the collectors with a prometheus.Registerer
func (p *PrometheusMetrics) AddToRegistry(registerer prometheus.Registerer) error {
	for s, op := range p.operationMap {
		for _, c := range counters {
			v, cv := newCounterRefFrom(prometheus.CounterOpts{
				Name:      c,
				Help:      fmt.Sprintf("%s counter", c),
				Subsystem: opSubsystemMap[s],
			}, nil)
			if err := registerer.Register(cv); err != nil {
				return err
			}
			addCounterRefToOp(c, v, op)
		}
	}
	for s, op := range p.operationMap {
		for _, c := range histograms {
			v, cv := newHistogramRefFrom(prometheus.HistogramOpts{
				Name:      c,
				Help:      fmt.Sprintf("%s histogram", c),
				Subsystem: opSubsystemMap[s],
			}, nil)
			if err := registerer.Register(cv); err != nil {
				return err
			}
			addHistogramRefToOp(c, v, op)
		}
	}
	return nil
}

func newCounterRefFrom(opts prometheus.CounterOpts, labelNames []string) (*gokitprom.Counter, prometheus.Collector) {
	cv := prometheus.NewCounterVec(opts, labelNames)
	return gokitprom.NewCounter(cv), cv
}

func newHistogramRefFrom(opts prometheus.HistogramOpts, labelNames []string) (*gokitprom.Histogram, prometheus.Collector) {
	hv := prometheus.NewHistogramVec(opts, labelNames)
	return gokitprom.NewHistogram(hv), hv
}

func addCounterRefToOp(name string, c *gokitprom.Counter, a *Aggregator) {
	switch name {
	case metricSuccesses:
		a.Successes = c
	case metricAttempts:
		a.Attempts = c
	case metricErrors:
		a.Errors = c
	case metricTimeouts:
		a.Timeouts = c
	}
}

func addHistogramRefToOp(name string, h *gokitprom.Histogram, a *Aggregator) {
	switch name {
	case metricRunDuration:
		a.RunDuration = h
	}
}
