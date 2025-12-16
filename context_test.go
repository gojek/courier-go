package courier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithClientIDCallback(t *testing.T) {
	tests := []struct {
		name     string
		invokeID string
		wantID   string
		callback bool
	}{
		{
			name:     "Invokes Callback",
			invokeID: "test-client",
			wantID:   "test-client",
			callback: true,
		},
		{
			name:     "Returns Empty String on nil Callback fn",
			invokeID: "",
			wantID:   "",
			callback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedID string
			var ctx context.Context
			if tt.callback {
				ctx = WithClientIDCallback(context.Background(), func(clientID string) {
					capturedID = clientID
				})
			} else {
				ctx = WithClientIDCallback(context.Background(), nil)
			}

			invokeClientIDCallback(ctx, tt.invokeID)
			assert.Equal(t, tt.wantID, capturedID)
		})
	}
}

func TestClientIDFromContext(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		wantID string
	}{
		{
			name:   "Returns Client ID",
			ctx:    withClientID(context.Background(), "test-subscribe"),
			wantID: "test-subscribe",
		},
		{
			name:   "Returns Empty String on Missing Client ID",
			ctx:    context.Background(),
			wantID: "",
		},
		{
			name:   "Returns Empty String on Nil Client ID",
			ctx:    context.WithValue(context.Background(), clientIDSubscribedKey{}, nil),
			wantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantID, ClientIDFromContext(tt.ctx))
		})
	}
}
