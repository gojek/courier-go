package courier

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
