package backoff

import (
	grpcbackoff "google.golang.org/grpc/backoff"
	"testing"
	"time"
)

func TestExponential_Backoff(t *testing.T) {

	tests := []struct {
		name    string
		retries int
		want    time.Duration
	}{
		{
			name:    "zero_retries",
			retries: 0,
			want:    1 * time.Second,
		},
		{
			name:    "non_zero_retries",
			retries: 1,
			want:    2 * time.Second,
		},
		{
			name:    "max_delay",
			retries: 8,
			want:    16 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := Exponential{Config: grpcbackoff.Config{
				BaseDelay:  1 * time.Second,
				Multiplier: 2,
				Jitter:     0,
				MaxDelay:   16 * time.Second,
			}}

			if got := bc.Backoff(tt.retries); got != tt.want {
				t.Errorf("Backoff() = %v, want %v", got, tt.want)
			}
		})
	}
}
