package courier

// Option changes behaviour of Publisher.Publish, Subscriber.Subscribe calls.
type Option interface {
	apply(*options)
}

// QOSLevel is an agreement between the sender of a message and the receiver of a message
// that defines the guarantee of delivery for a specific message
type QOSLevel uint8

const (
	// QOSZero denotes at most once message delivery
	QOSZero QOSLevel = 0
	// QOSOne denotes at least once message delivery
	QOSOne QOSLevel = 1
	// QOSTwo denotes exactly once message delivery
	QOSTwo QOSLevel = 2
)

func (q QOSLevel) apply(opts *options) {
	opts.qos = byte(q)
}

// Retained is an option used with Publisher.Publish call
type Retained bool

func (r Retained) apply(opts *options) {
	opts.retained = bool(r)
}

type options struct {
	qos      byte
	retained bool
}

func composeOptions(opts []Option) *options {
	res := &options{}

	for _, opt := range opts {
		opt.apply(res)
	}

	return res
}
