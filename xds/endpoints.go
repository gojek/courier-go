package xds

import (
	"context"
	"fmt"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	googleapiPrefix = "type.googleapis.com/"

	V3ClusterType    = "envoy.config.cluster.v3.Cluster"
	V3ClusterTypeURL = googleapiPrefix + V3ClusterType
)

type adsClient discoveryv3.AggregatedDiscoveryService_StreamAggregatedResourcesClient

type Client struct {
	cc   *grpc.ClientConn
	node *corev3.Node
}

func (c *Client) streamEndpoints(ctx context.Context, resourceNames []string, version, nonce string) error {
	ac, err := c.adsClient(ctx)
	if err != nil {
		return err
	}

	go func() {
		if err := ac.Send(&discoveryv3.DiscoveryRequest{
			Node:          c.node,
			TypeUrl:       V3ClusterTypeURL,
			ResourceNames: resourceNames,
			VersionInfo:   version,
			ResponseNonce: nonce,
		}); err != nil {
			panic(err)
		}
	}()

	ch, errCh := c.recv(ctx, ac)

	for {
		select {
		case <-ctx.Done():
			return nil
		case dr := <-ch:
			cls := new(cluster.Cluster)

			if err := anypb.UnmarshalTo(&anypb.Any{TypeUrl: dr.GetResources()[0].GetTypeUrl(), Value: dr.GetResources()[0].GetValue()}, cls, proto.UnmarshalOptions{}); err != nil {
				errCh <- err
			}

			fmt.Println("Received v3.Cluster", cls)
		case err := <-errCh:
			close(errCh)
			return err
		}
	}
}

func (c *Client) adsClient(ctx context.Context) (adsClient, error) {
	v, err := c.newStream(ctx, c.cc)
	if err != nil {
		return nil, err
	}
	str, ok := v.(discoveryv3.AggregatedDiscoveryService_StreamAggregatedResourcesClient)
	if !ok {
		return nil, fmt.Errorf("xds: Attempt to send request on unsupported stream type: %T", v)
	}

	return str, nil
}

func (c *Client) newStream(ctx context.Context, cc *grpc.ClientConn) (grpc.ClientStream, error) {
	return discoveryv3.NewAggregatedDiscoveryServiceClient(cc).StreamAggregatedResources(ctx, grpc.WaitForReady(true))
}

func (c *Client) recv(ctx context.Context, ac adsClient) (chan *discoveryv3.DiscoveryResponse, chan error) {
	ch := make(chan *discoveryv3.DiscoveryResponse)
	errCh := make(chan error, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			default:
				dr, err := ac.Recv()
				if err != nil {
					errCh <- err
					continue
				}
				ch <- dr
			}
		}
	}()

	return ch, errCh
}
