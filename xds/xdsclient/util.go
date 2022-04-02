package xdsclient

import (
	"net"
	"strconv"

	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
)

func parseEndpoints(lbEndpoints []*v3endpointpb.LbEndpoint) []string {
	endpoints := make([]string, 0, len(lbEndpoints))
	for _, lbEndpoint := range lbEndpoints {
		endpoints = append(endpoints, parseAddress(lbEndpoint.GetEndpoint().GetAddress().GetSocketAddress()))
	}
	return endpoints
}

func parseAddress(socketAddress *v3corepb.SocketAddress) string {
	return net.JoinHostPort(socketAddress.GetAddress(), strconv.Itoa(int(socketAddress.GetPortValue())))
}

