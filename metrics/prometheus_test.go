package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewPrometheus(t *testing.T) {
	reg := prometheus.NewRegistry()
	p := NewPrometheus()
	assert.NotNil(t, p)
	err := p.AddToRegistry(reg)
	assert.NoError(t, err)
}

func TestPrometheusMetrics_AddToRegistry_error(t *testing.T) {
	reg := prometheus.NewRegistry()
	p := NewPrometheus()
	err := p.AddToRegistry(reg)
	assert.NoError(t, err)
	err = p.AddToRegistry(reg)
	assert.Error(t, err)

	// unregister counters to test histogram duplicate registration
	for s := range p.operationMap {
		for _, c := range counters {
			_, cv := newCounterRefFrom(prometheus.CounterOpts{
				Name:      c,
				Subsystem: opSubsystemMap[s],
			}, nil)
			reg.Unregister(cv)
		}
	}
	err = p.AddToRegistry(reg)
	assert.Error(t, err)
}

func TestPrometheusMetrics_Update(t *testing.T) {
	reg := prometheus.NewRegistry()
	p := NewPrometheus()
	err := p.AddToRegistry(reg)
	assert.NoError(t, err)

	testcases := []struct {
		name     string
		res      Result
		expected string
		metric   string
	}{
		{
			name: "Attempts",
			res: Result{
				OpType:   PublishOp,
				Attempts: 1,
			},
			expected: `
# HELP courier_publish_attempts attempts counter
# TYPE courier_publish_attempts counter
courier_publish_attempts 1
`,
			metric: metricName(metricAttempts),
		},
		{
			name: "Successes",
			res: Result{
				OpType:    PublishOp,
				Successes: 1,
			},
			expected: `
# HELP courier_publish_successes successes counter
# TYPE courier_publish_successes counter
courier_publish_successes 1
`,
			metric: metricName(metricSuccesses),
		},
		{
			name: "Timeouts",
			res: Result{
				OpType:   PublishOp,
				Timeouts: 1,
			},
			expected: `
# HELP courier_publish_timeouts timeouts counter
# TYPE courier_publish_timeouts counter
courier_publish_timeouts 1
`,
			metric: metricName(metricTimeouts),
		},
		{
			name: "Errors",
			res: Result{
				OpType: PublishOp,
				Errors: 1,
			},
			expected: `
# HELP courier_publish_errors errors counter
# TYPE courier_publish_errors counter
courier_publish_errors 1
`,
			metric: metricName(metricErrors),
		},
		{
			name: "RunDuration",
			res: Result{
				OpType:      PublishOp,
				RunDuration: time.Millisecond,
			},
			expected: `
# HELP courier_publish_run_duration run_duration histogram
# TYPE courier_publish_run_duration histogram
courier_publish_run_duration_bucket{le="0.005"} 1
courier_publish_run_duration_bucket{le="0.01"} 1
courier_publish_run_duration_bucket{le="0.025"} 1
courier_publish_run_duration_bucket{le="0.05"} 1
courier_publish_run_duration_bucket{le="0.1"} 1
courier_publish_run_duration_bucket{le="0.25"} 1
courier_publish_run_duration_bucket{le="0.5"} 1
courier_publish_run_duration_bucket{le="1"} 1
courier_publish_run_duration_bucket{le="2.5"} 1
courier_publish_run_duration_bucket{le="5"} 1
courier_publish_run_duration_bucket{le="10"} 1
courier_publish_run_duration_bucket{le="+Inf"} 1
courier_publish_run_duration_sum 0.001
courier_publish_run_duration_count 1
`,
			metric: metricName(metricRunDuration),
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			p.Update(tt.res)
			assert.NoError(t, testutil.GatherAndCompare(reg, strings.NewReader(tt.expected), tt.metric))
		})
	}
}

func metricName(name string) string {
	return subsystemPublish + "_" + name
}
