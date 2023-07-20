package courier

import (
	"context"
	"time"
)

// StartOption can be used to customise behaviour of ExponentialStartStrategy
type StartOption func(*startOptions)

// WithMaxInterval sets the maximum interval the retry logic will wait
// before attempting another Client.Start, Default is 30 seconds
func WithMaxInterval(interval time.Duration) StartOption {
	return func(o *startOptions) {
		if interval > 0 {
			o.maxInterval = interval
		}
	}
}

// WithOnRetry sets the func which is called when there is an error
// in the previous Client.Start attempt
func WithOnRetry(retryFunc func(error)) StartOption {
	return func(o *startOptions) {
		o.onRetry = retryFunc
	}
}

// ExponentialStartStrategy will keep attempting to call Client.Start in the background and retry on error,
// it will never exit unless the context used to invoke is cancelled.
// This will NOT stop the client, that is the responsibility of caller.
func ExponentialStartStrategy(ctx context.Context, c interface{ Start() error }, opts ...StartOption) {
	so := defaultStartOptions()

	for _, opt := range opts {
		opt(so)
	}

	exponentialStartStrategy(ctx, c, so)
}

type startOptions struct {
	onRetry     func(error)
	maxInterval time.Duration
}

func defaultStartOptions() *startOptions {
	return &startOptions{maxInterval: 30 * time.Second}
}

func exponentialStartStrategy(ctx context.Context, c interface{ Start() error }, so *startOptions) {
	nextRetryInterval := 100 * time.Millisecond
	errCh := make(chan error, 1)

	startFn := func() {
		errCh <- c.Start()
	}
	go startFn()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errCh:
				if err == nil {
					return
				}

				if so.onRetry != nil {
					so.onRetry(err)
				}

				go startFn()
				time.Sleep(nextRetryInterval)
				nextRetryInterval = min(nextRetryInterval*2, so.maxInterval)
			}
		}
	}()
}

func min(x time.Duration, y time.Duration) time.Duration {
	if x < y {
		return x
	}

	return y
}
