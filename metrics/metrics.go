package metrics

import (
	"time"
)

type Operation int

const (
	PublishOp Operation = iota
	SubscribeOp
	SubscribeMultipleOp
	UnsubscribeOp
	CallbackOp
)

type Result struct {
	OpType      Operation
	Attempts    float64
	Timeouts    float64
	Errors      float64
	Successes   float64
	RunDuration time.Duration
}

// Aggregator struct holds the metric collectors for an operation
type Aggregator struct {
	// Attempts is a counter which tracks number of attempts for an operation
	Attempts Counter

	// Timeouts is a counter which tracks number of timeouts for an operation
	Timeouts Counter

	// Errors is a counter which tracks number of errors for an operation
	Errors Counter

	// Successes is a counter which tracks number of successes for an operation
	Successes Counter

	// RunDuration is a histogram which tracks run durations of an operation
	RunDuration Histogram
}

// Collector handles the metric updates.
//
// Deprecated: Use Middlewares (courier.PublisherMiddlewareFunc, courier.SubscriberMiddlewareFunc
// and/or courier.UnsubscriberMiddlewareFunc) to instrument calls.
type Collector interface {
	// Update is called on every Operation performed,
	// this method should handle any read/write in a concurrency safe manner
	Update(Result)
}
