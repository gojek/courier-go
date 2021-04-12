package metrics

import (
	gokit "github.com/go-kit/kit/metrics"
)

// Counter describes a metric that accumulates values monotonically.
// An example of a counter is the number of received HTTP requests.
type Counter = gokit.Counter

// Histogram describes a metric that takes repeated observations of the same
// kind of thing, and produces a statistical summary of those observations,
// typically expressed as quantiles or buckets. An example of a histogram is
// HTTP request latencies.
type Histogram = gokit.Histogram
