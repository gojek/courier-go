package xds

import (
	"sort"

	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"

	"github.com/gojek/courier-go"
)

type clusterUpdateReceiver interface {
	Receive() <-chan []*v3endpointpb.ClusterLoadAssignment
	Done() <-chan struct{}
}

// Resolver sends updates to via the channel returned by UpdateChan()
type Resolver struct {
	rc clusterUpdateReceiver
	ch chan []courier.TCPAddress
}

// NewResolver returns a *Resolver that uses rc to receive cluster updates
func NewResolver(rc clusterUpdateReceiver) *Resolver {
	r := &Resolver{
		rc: rc,
		ch: make(chan []courier.TCPAddress),
	}

	go r.run()

	return r
}

// UpdateChan returns a channel where []courier.TCPAddress can be received
func (r *Resolver) UpdateChan() <-chan []courier.TCPAddress {
	return r.ch
}

// Done returns a channel which is closed when the underlying clusterUpdateReceiver is marked as done
func (r *Resolver) Done() <-chan struct{} {
	return r.rc.Done()
}

func (r *Resolver) run() {
	for {
		select {
		case <-r.Done():
			return
		case resources := <-r.rc.Receive():
			es := endpointSlice{}
			for _, cla := range resources {
				es = append(es, r.weightedEndpoints(cla)...)
			}

			sort.Sort(es)

			r.ch <- es.values()
		}
	}
}

func (r *Resolver) weightedEndpoints(resource *v3endpointpb.ClusterLoadAssignment) []weightedEp {
	var endpoints []weightedEp

	for _, locality := range resource.GetEndpoints() {
		for _, ep := range locality.GetLbEndpoints() {
			sockAddr := ep.GetEndpoint().GetAddress().GetSocketAddress()

			endpoints = append(endpoints, weightedEp{
				weight: ep.GetLoadBalancingWeight().GetValue(),
				value: courier.TCPAddress{
					Host: sockAddr.GetAddress(),
					Port: uint16(sockAddr.GetPortValue()),
				},
			})
		}
	}

	return endpoints
}
