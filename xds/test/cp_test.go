package test

import (
	"context"
	"fmt"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/protobuf/types/known/durationpb"
	"net"
	"os"
	"os/signal"
	"testing"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimev3 "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretv3 "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"
)

func TestControlPlane(t *testing.T) {
	l, err := net.Listen("tcp", ":9100")
	if err != nil {
		t.Fail()
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)

	snapshotCache := cache.NewSnapshotCache(true, cache.IDHash{}, nil)
	if err := snapshotCache.SetSnapshot(context.Background(), "52fdfc07-2182-454f-963f-5f0f9a621d72", generateSnap()); err != nil {
		t.Error(err)
	}

	srv := grpc.NewServer()
	hSrv := serverv3.NewServer(ctx, snapshotCache, serverv3.CallbackFuncs{
		StreamOpenFunc: func(ctx context.Context, i int64, s string) error {
			fmt.Println("StreamOpenFunc", ctx, i, s)
			return nil
		},
		StreamClosedFunc: func(i int64) {
			fmt.Println("StreamClosedFunc", i)
		},
		DeltaStreamOpenFunc: func(ctx context.Context, i int64, s string) error {
			fmt.Println("DeltaStreamOpenFunc", ctx, i, s)
			return nil
		},
		DeltaStreamClosedFunc: func(i int64) {
			fmt.Println("DeltaStreamClosedFunc", i)
		},
		StreamRequestFunc: func(i int64, r *discoveryv3.DiscoveryRequest) error {
			fmt.Println("StreamRequestFunc", i, r)
			return nil
		},
		StreamResponseFunc: func(ctx context.Context, i int64, rq *discoveryv3.DiscoveryRequest, rp *discoveryv3.DiscoveryResponse) {
			fmt.Println("StreamResponseFunc", ctx, i, rq, rp)
		},
		StreamDeltaRequestFunc: func(i int64, r *discoveryv3.DeltaDiscoveryRequest) error {
			fmt.Println("StreamDeltaRequestFunc", i, r)
			return nil
		},
		StreamDeltaResponseFunc: func(i int64, rq *discoveryv3.DeltaDiscoveryRequest, rp *discoveryv3.DeltaDiscoveryResponse) {
			fmt.Println("StreamDeltaResponseFunc", i, rq, rp)
		},
		FetchRequestFunc: func(ctx context.Context, r *discoveryv3.DiscoveryRequest) error {
			fmt.Println("FetchRequestFunc", ctx, r)
			return nil
		},
		FetchResponseFunc: func(rq *discoveryv3.DiscoveryRequest, rp *discoveryv3.DiscoveryResponse) {
			fmt.Println("FetchResponseFunc", rq, rp)
		},
	})
	registerServer(srv, hSrv)

	go func() {
		if err := srv.Serve(l); err != grpc.ErrServerStopped {
			t.Error(err)
		}
	}()

	<-ctx.Done()

	srv.GracefulStop()
}

func generateSnap() cache.Snapshot {
	snap, _ := cache.NewSnapshot("1",
		map[resource.Type][]types.Resource{
			resource.ClusterType: {
				&cluster.Cluster{
					Name:                 "customer",
					ConnectTimeout:       durationpb.New(5 * time.Second),
					ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_LOGICAL_DNS},
					LbPolicy:             cluster.Cluster_ROUND_ROBIN,
					LoadAssignment: &endpoint.ClusterLoadAssignment{
						ClusterName: "customer",
						Endpoints: []*endpoint.LocalityLbEndpoints{{
							LbEndpoints: []*endpoint.LbEndpoint{{
								HostIdentifier: &endpoint.LbEndpoint_Endpoint{
									Endpoint: &endpoint.Endpoint{
										Address: &core.Address{
											Address: &core.Address_SocketAddress{
												SocketAddress: &core.SocketAddress{
													Protocol: core.SocketAddress_TCP,
													Address:  "customer-internal.golabs.io",
													PortSpecifier: &core.SocketAddress_PortValue{
														PortValue: 1883,
													},
												},
											},
										},
									},
								},
							}},
						}},
					},
					DnsLookupFamily: cluster.Cluster_V4_ONLY,
				},
			},
		},
	)
	return snap
}

func registerServer(grpcServer *grpc.Server, server serverv3.Server) {
	// register services
	discoveryv3.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	endpointv3.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	clusterv3.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	routev3.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	listenerv3.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	secretv3.RegisterSecretDiscoveryServiceServer(grpcServer, server)
	runtimev3.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)
}
