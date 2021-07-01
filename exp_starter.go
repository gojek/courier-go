package courier

import (
	"context"
	"sync"
	"time"
)

type startOptions struct {
	maxInterval time.Duration
	onRetry     func(error)
}

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
func ExponentialStartStrategy(c *Client, opts ...StartOption) func(context.Context) error {
	so := startOptions{maxInterval: 30 * time.Second}
	for _, opt := range opts {
		opt(&so)
	}

	internalCtx, internalCancel := context.WithCancel(context.Background())
	internalErrCh := make(chan error, 1)

	wg := &sync.WaitGroup{}
	attemptStart := func(wg *sync.WaitGroup) {
		wg.Add(1)
		defer wg.Done()
		internalErrCh <- c.Start(internalCtx)
	}

	retryFunc := func(retryInterval time.Duration) error {
		err := <-internalErrCh
		if err != nil {
			if so.onRetry != nil {
				so.onRetry(err)
			}
			go attemptStart(wg)
			<-time.After(retryInterval)
			return err
		}
		return nil
	}

	return func(ctx context.Context) error {
		go attemptStart(wg)
		go func() {
			nextRetryInterval := 100 * time.Millisecond
			err := retryFunc(nextRetryInterval)
			for err != nil {
				nextRetryInterval = min(nextRetryInterval*2, so.maxInterval)
				err = retryFunc(nextRetryInterval)
			}
		}()

		<-ctx.Done()
		internalCancel()
		wg.Wait()
		return nil
	}
}

func min(x time.Duration, y time.Duration) time.Duration {
	if x < y {
		return x
	}
	return y
}
