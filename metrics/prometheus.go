package metrics

import (
	gokitprom "github.com/go-kit/kit/metrics/prometheus"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	subsystemPrefix         = "courier"
	subsystemPublish        = subsystemPrefix + "_publish"
	subsystemUnsubscribe    = subsystemPrefix + "_unsubscribe"
	subsystemSubscribe      = subsystemPrefix + "_subscribe"
	subsystemSubscribeMulti = subsystemPrefix + "_subscribe_multiple"

	metricSuccesses           = "successes"
	metricAttempts            = "attempts"
	metricErrors              = "errors"
	metricTimeouts            = "timeouts"
	metricRunDuration         = "run_duration"
	metricCallbackRunDuration = "callback_run_duration"
)

var (
	opMaps = map[string]*Op{
		subsystemPublish:        {},
		subsystemUnsubscribe:    {},
		subsystemSubscribe:      {},
		subsystemSubscribeMulti: {},
	}

	counters   = []string{metricSuccesses, metricAttempts, metricErrors, metricTimeouts}
	histograms = []string{metricRunDuration}
)

// NewPrometheus creates a PrometheusMetrics instance which implements the Metrics interface
func NewPrometheus() *PrometheusMetrics {
	var cls []prometheus.Collector
	for s, op := range opMaps {
		for _, c := range counters {
			v, cv := newCounterRefFrom(prometheus.CounterOpts{
				Name:      c,
				Subsystem: s,
			}, nil)
			cls = append(cls, cv)
			addCounterRefToOp(c, v, op)
		}
	}
	for s, op := range opMaps {
		for _, c := range histograms {
			v, cv := newHistogramRefFrom(prometheus.HistogramOpts{
				Name:      c,
				Subsystem: s,
			}, nil)
			cls = append(cls, cv)
			addHistogramRefToOp(c, v, op)
		}
	}
	crd, crdhv := newHistogramRefFrom(prometheus.HistogramOpts{
		Name:      metricCallbackRunDuration,
		Subsystem: subsystemPrefix,
	}, nil)
	cls = append(cls, crdhv)

	return &PrometheusMetrics{
		collectors:     cls,
		publish:        opMaps[subsystemPublish],
		unsubscribe:    opMaps[subsystemUnsubscribe],
		subscribe:      opMaps[subsystemSubscribe],
		subscribeMulti: opMaps[subsystemSubscribeMulti],
		callbackOp:     &CallbackOp{RunDuration: crd},
	}
}

// PrometheusMetrics is a prometheus collector for courier.Client operations
type PrometheusMetrics struct {
	collectors     []prometheus.Collector
	publish        *Op
	unsubscribe    *Op
	subscribe      *Op
	subscribeMulti *Op
	callbackOp     *CallbackOp
}

func (p *PrometheusMetrics) Publish() *Op {
	return p.publish
}

func (p *PrometheusMetrics) Unsubscribe() *Op {
	return p.unsubscribe
}

func (p *PrometheusMetrics) Subscribe() *Op {
	return p.subscribe
}

func (p *PrometheusMetrics) SubscribeMultiple() *Op {
	return p.subscribeMulti
}

func (p *PrometheusMetrics) CallbackOp() *CallbackOp {
	return p.callbackOp
}

// AddToRegistry is used to register the collectors with a prometheus.Registerer
func (p *PrometheusMetrics) AddToRegistry(registerer prometheus.Registerer) error {
	for _, cl := range p.collectors {
		if err := registerer.Register(cl); err != nil {
			return err
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

func addCounterRefToOp(name string, c *gokitprom.Counter, op *Op) {
	switch name {
	case metricSuccesses:
		op.Successes = c
	case metricAttempts:
		op.Attempts = c
	case metricErrors:
		op.Errors = c
	case metricTimeouts:
		op.Timeouts = c
	}
}

func addHistogramRefToOp(name string, h *gokitprom.Histogram, op *Op) {
	switch name {
	case metricRunDuration:
		op.RunDuration = h
	}
}
