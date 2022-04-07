package xdsclient

//import (
//	"github.com/gojekfarm/courier-go/xds/types"
//	"net"
//	"strconv"
//
//	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
//	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
//)
//
//func mapToSlice(m map[string]struct{}) []string {
//	ret := make([]string, 0, len(m))
//	for i := range m {
//		ret = append(ret, i)
//	}
//	return ret
//}
//
//func endpointWatcherSliceToMap(ep []types.EndpointWatcher) map[string]struct{} {
//	ret := make(map[string]struct{}, len(ep))
//	for _, v := range ep {
//		ret[v.Endpoint] = struct{}{}
//	}
//	return ret
//}
//
//func parseEndpoints(lbEndpoints []*v3endpointpb.LbEndpoint) []string {
//	endpoints := make([]string, 0, len(lbEndpoints))
//	for _, lbEndpoint := range lbEndpoints {
//		endpoints = append(endpoints, parseAddress(lbEndpoint.GetEndpoint().GetAddress().GetSocketAddress()))
//	}
//	return endpoints
//}
//
//func parseAddress(socketAddress *v3corepb.SocketAddress) string {
//	return net.JoinHostPort(socketAddress.GetAddress(), strconv.Itoa(int(socketAddress.GetPortValue())))
//}

