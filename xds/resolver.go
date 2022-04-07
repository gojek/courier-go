package xds

import (
	"context"
	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/gojekfarm/courier-go"
	"sort"
)

type clusterUpdateReceiver interface {
	Receive() <-chan []*v3endpointpb.ClusterLoadAssignment
}

type weightedEp struct {
	weight uint32
	value  courier.TCPAddress
}

type Resolver struct {
	rc clusterUpdateReceiver
	ch chan []courier.TCPAddress
}

var _ courier.Resolver = (*Resolver)(nil)

func NewResolver(rc clusterUpdateReceiver) *Resolver {
	return &Resolver{
		rc: rc,
		ch: make(chan []courier.TCPAddress),
	}
}

func (r *Resolver) UpdateChan() <-chan []courier.TCPAddress {
	return r.ch
}

func (r *Resolver) Done() <-chan struct{} {
	close(r.ch)
	done := make(chan struct{}, 1)
	return done
}

func (r *Resolver) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
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
