package courier

import (
	"errors"
)

var (
	ErrConnectTimeout           = errors.New("client timed out while trying to connect to the broker")
	ErrPublishTimeout           = errors.New("publish timeout")
	ErrSubscribeTimeout         = errors.New("subscribe timeout")
	ErrUnsubscribeTimeout       = errors.New("unsubscribe timeout")
	ErrSubscribeMultipleTimeout = errors.New("subscribe multiple timeout")
)
