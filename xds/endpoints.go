package xds

import (
	"context"
	"fmt"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc"
)

const (
	googleapiPrefix    = "type.googleapis.com/"
	V3EndpointsType    = "envoy.config.endpoint.v3.ClusterLoadAssignment"
	V3EndpointsTypeURL = googleapiPrefix + V3EndpointsType
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
		_ = ac.Send(&discoveryv3.DiscoveryRequest{
			Node:          c.node,
			TypeUrl:       V3EndpointsTypeURL,
			ResourceNames: resourceNames,
			VersionInfo:   version,
			ResponseNonce: nonce,
		})
	}()

	ch, errCh := c.recv(ctx, ac)

	for {
		select {
		case <-ctx.Done():
			return nil
		case dr := <-ch:
			fmt.Println("Received v3.DiscoveryResponse", dr.String())
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
