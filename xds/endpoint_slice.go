package xds

import (
	"github.com/gojek/courier-go"
)

type weightedEp struct {
	weight uint32
	value  courier.TCPAddress
}

type endpointSlice []weightedEp

// Len is the number of elements in the collection.
func (es endpointSlice) Len() int { return len(es) }

// Less reports whether the element with index i must sort before the element with index j.
func (es endpointSlice) Less(i, j int) bool { return es[i].weight > es[j].weight }

// Swap swaps the elements with indexes i and j.
func (es endpointSlice) Swap(i, j int) { es[i], es[j] = es[j], es[i] }

func (es endpointSlice) values() []courier.TCPAddress {
	ret := make([]courier.TCPAddress, 0, len(es))

	for _, wep := range es {
		ret = append(ret, wep.value)
	}

	return ret
}
