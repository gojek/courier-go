package metrics

// Op struct holds the metric collectors for an operation
type Op struct {
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

// CallbackOp struct holds the metric collectors for a callback operation
type CallbackOp struct {
	// RunDuration is a histogram which tracks run durations of a callback operation
	RunDuration Histogram
}

// Metrics allows access to different operation collector implementation
type Metrics interface {
	// Publish returns an Op struct to be used in courier.Client#Publish method
	Publish() *Op

	// Unsubscribe returns an Op struct to be used in courier.Client#Unsubscribe method
	Unsubscribe() *Op

	// Subscribe returns an Op struct to be used in courier.Client#Subscribe method
	Subscribe() *Op

	// SubscribeMultiple returns an Op struct to be used in courier.Client#SubscribeMultiple method
	SubscribeMultiple() *Op

	// CallbackOp returns a CallbackOp struct to be used while calling the
	// callback in courier.Client subscription methods
	CallbackOp() *CallbackOp
}
