package courier

import (
	"errors"
)

var (
	// ErrConnectTimeout indicates connection timeout while connecting to broker.
	ErrConnectTimeout = errors.New("client timed out while trying to connect to the broker")
	// ErrPublishTimeout indicates publish timeout.
	ErrPublishTimeout = errors.New("publish timeout")
	// ErrSubscribeTimeout indicates subscribe timeout.
	ErrSubscribeTimeout = errors.New("subscribe timeout")
	// ErrUnsubscribeTimeout indicates unsubscribe timeout.
	ErrUnsubscribeTimeout = errors.New("unsubscribe timeout")
	// ErrSubscribeMultipleTimeout indicates multiple subscribe timeout.
	ErrSubscribeMultipleTimeout = errors.New("subscribe multiple timeout")
)
