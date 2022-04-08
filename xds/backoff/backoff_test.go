package backoff

import (
	grpcbackoff "google.golang.org/grpc/backoff"
	"testing"
	"time"
)

func TestExponential_Backoff(t *testing.T) {
	tests := []struct {
		name      string
		retries   int
		baseDelay time.Duration
		maxDelay  time.Duration
		want      time.Duration
	}{
		{
			name:      "zero_retries",
			retries:   0,
			baseDelay: 1 * time.Second,
			maxDelay:  16 * time.Second,
			want:      1 * time.Second,
		},
		{
			name:      "non_zero_retries",
			retries:   1,
			baseDelay: 1 * time.Second,
			maxDelay:  16 * time.Second,
			want:      2 * time.Second,
		},
		{
			name:      "max_delay",
			retries:   8,
			baseDelay: 1 * time.Second,
			maxDelay:  16 * time.Second,
			want:      16 * time.Second,
		},
		{
			name:      "more_than_max_delay",
			retries:   8,
			baseDelay: 16 * time.Second,
			maxDelay:  8 * time.Second,
			want:      8 * time.Second,
		},
		{
			name:      "negative_delay",
			retries:   1,
			baseDelay: -1 * time.Second,
			maxDelay:  8 * time.Second,
			want:      0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := Exponential{Config: grpcbackoff.Config{
				BaseDelay:  tt.baseDelay,
				Multiplier: 2,
				Jitter:     0,
				MaxDelay:   tt.maxDelay,
			}}

			if got := bc.Backoff(tt.retries); got != tt.want {
				t.Errorf("Backoff() = %v, want %v", got, tt.want)
			}
		})
	}
}
