package xds

import (
	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/gojekfarm/courier-go"
	"sort"
)

type clusterUpdateReceiver interface {
	Receive() <-chan []*v3endpointpb.ClusterLoadAssignment
	Done() <-chan struct{}
}

type weightedEp struct {
	weight uint32
	value  courier.TCPAddress
}

type Resolver struct {
	rc clusterUpdateReceiver
	ch chan []courier.TCPAddress
}

func NewResolver(rc clusterUpdateReceiver) *Resolver {
	r := &Resolver{
		rc: rc,
		ch: make(chan []courier.TCPAddress),
	}

	go r.run()

	return r
}

func (r *Resolver) UpdateChan() <-chan []courier.TCPAddress {
	return r.ch
}

func (r *Resolver) Done() <-chan struct{} {
	return r.rc.Done()
}

func (r *Resolver) run() {
	for {
		select {
		case <-r.Done():
			return
		case resources := <-r.rc.Receive():
			var weightedEndpoints []weightedEp

			for _, cla := range resources {
				weightedEndpoints = append(weightedEndpoints, r.sliceEndpoints(cla)...)
			}

			sort.Slice(weightedEndpoints, func(i, j int) bool {
				return weightedEndpoints[i].weight > weightedEndpoints[j].weight
			})

			ret := make([]courier.TCPAddress, 0, len(weightedEndpoints))

			for _, wep := range weightedEndpoints {
				ret = append(ret, wep.value)
			}

			r.ch <- ret
		}
	}
}

func (r *Resolver) sliceEndpoints(resource *v3endpointpb.ClusterLoadAssignment) []weightedEp {
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
