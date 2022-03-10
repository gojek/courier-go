package courier

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_composeOptions(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want *options
	}{
		{
			name: "QoS",
			opts: []Option{QOSOne},
			want: &options{qos: 1},
		},
		{
			name: "Retained",
			opts: []Option{Retained(true)},
			want: &options{retained: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, composeOptions(tt.opts))
		})
	}
}
