package courier

import (
	"time"
)

// WaitForConnection checks if the Client is connected, it calls ConnectionInformer.IsConnected
// after every tick and waitFor is the maximum duration it can block.
// Returns true only when ConnectionInformer.IsConnected returns true
func WaitForConnection(c ConnectionInformer, waitFor time.Duration, tick time.Duration) bool {
	ch := make(chan bool, 1)

	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for tick := ticker.C; ; {
		select {
		case <-timer.C:
			return false
		case <-tick:
			tick = nil

			go func() { ch <- c.IsConnected() }()
		case v := <-ch:
			if v {
				return true
			}

			tick = ticker.C
		}
	}
}
