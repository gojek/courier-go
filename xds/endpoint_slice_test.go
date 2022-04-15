package xds

import (
	"github.com/gojek/courier-go"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func Test_endpointSlice_Sort(t *testing.T) {
	tests := []struct {
		name string
		es   endpointSlice
		want []courier.TCPAddress
	}{
		{
			name: "Case1",
			es: []weightedEp{
				{weight: 0, value: courier.TCPAddress{Port: 1883}},
				{weight: 1, value: courier.TCPAddress{Port: 1884}},
				{weight: 2, value: courier.TCPAddress{Port: 1885}},
				{weight: 3, value: courier.TCPAddress{Port: 1886}},
			},
			want: []courier.TCPAddress{
				{Port: 1886},
				{Port: 1885},
				{Port: 1884},
				{Port: 1883},
			},
		},
		{
			name: "Case2",
			es: []weightedEp{
				{weight: 0, value: courier.TCPAddress{Port: 1883}},
				{weight: 3, value: courier.TCPAddress{Port: 1884}},
				{weight: 1, value: courier.TCPAddress{Port: 1885}},
				{weight: 2, value: courier.TCPAddress{Port: 1886}},
			},
			want: []courier.TCPAddress{
				{Port: 1884},
				{Port: 1886},
				{Port: 1885},
				{Port: 1883},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := tt.es
			sort.Sort(es)
			assert.Equal(t, tt.want, es.values())
		})
	}
}
