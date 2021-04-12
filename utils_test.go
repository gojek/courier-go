package courier

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWaitForConnection(t *testing.T) {
	tests := []struct {
		name     string
		informer *testInformer
		want     bool
	}{
		{
			name:     "Success",
			informer: &testInformer{connected: true},
			want:     true,
		},
		{
			name:     "Failure",
			informer: &testInformer{connected: false},
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, WaitForConnection(tt.informer, 1*time.Second, 100*time.Millisecond))
		})
	}
}

type testInformer struct {
	connected bool
}

func (m *testInformer) IsConnected() bool {
	return m.connected
}
