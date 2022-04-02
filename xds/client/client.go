package client

import (
	"context"
	"errors"
	"fmt"
	"log"

	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	v3edsgrpc "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/proto"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/anypb"
)

func NewClient(ctx context.Context, nodeProto *v3corepb.Node, cc *grpc.ClientConn) (Client, error) {
	edsClient, err := v3edsgrpc.NewEndpointDiscoveryServiceClient(cc).StreamEndpoints(ctx, grpc.WaitForReady(true))
	if err != nil {
		return nil, err
	}

	v3c := &client{
		nodeProto: nodeProto,
		edsClient:    edsClient,
	}
	return v3c, nil
}

type edsStream v3edsgrpc.EndpointDiscoveryService_StreamEndpointsClient

// client performs the actual xDS RPCs using the xDS v3 API. It creates an
// EDS stream on which the different types of xDS requests and responses
// are multiplexed.
type client struct {
	nodeProto *v3corepb.Node
	edsClient edsStream
}

type Client interface {
	// RestartEDSClient returns a new xDS client stream specific to the underlying
	// transport protocol version.
	RestartEDSClient(ctx context.Context, cc *grpc.ClientConn) error
	// SendRequest constructs and sends out a DiscoveryRequest message specific
	// to the underlying transport protocol version.
	SendRequest(resourceNames []string, version, nonce, errMsg string) error
	// ParseResponse type asserts message to the versioned response, and
	// retrieves the fields.
	ParseResponse(r proto.Message) ([]*anypb.Any, string, string, error)

	// Receive uses the provided stream to receive a response.
	Receive() (proto.Message, error)
}

func (c *client) RestartEDSClient(ctx context.Context, cc *grpc.ClientConn) error {
	edsClient, err :=  v3edsgrpc.NewEndpointDiscoveryServiceClient(cc).StreamEndpoints(ctx, grpc.WaitForReady(true))
	c.edsClient = edsClient

	return err
}

func (c *client) SendRequest(resourceNames []string, version, nonce, errMsg string) error {
	req := &v3discoverypb.DiscoveryRequest{
		Node:          c.nodeProto,
		TypeUrl:       resource.EndpointType,
		ResourceNames: resourceNames,
		VersionInfo:   version,
		ResponseNonce: nonce,
	}
	if errMsg != "" {
		req.ErrorDetail = &statuspb.Status{
			Code: int32(codes.InvalidArgument), Message: errMsg,
		}
	}
	if err := c.edsClient.Send(req); err != nil {
		return fmt.Errorf("xds: stream.Send(%+v) failed: %v", req, err)
	}
	return nil
}

// Receive blocks on the receipt of one response message on the provided
// stream.
func (c *client) Receive() (proto.Message, error) {
	resp, err := c.edsClient.Recv()
	fmt.Println("OOOOOOOO Very many responses happening: ", resp)
	if err != nil {
		return nil, fmt.Errorf("xds: stream.Recv() failed: %v", err)
	}
	log.Printf("ADS response received, type: %v\n", resp.GetTypeUrl())
	log.Printf("ADS response received: %+v", resp)
	return resp, nil
}

func (c *client) ParseResponse(r proto.Message) ([]*anypb.Any, string, string, error) {
	resp, ok := r.(*v3discoverypb.DiscoveryResponse)
	if !ok {
		return nil, "", "", fmt.Errorf("xds: unsupported message type: %T", resp)
	}

	// Note that the xDS transport protocol is versioned independently of
	// the resource types, and it is supported to transfer older versions
	// of resource types using new versions of the transport protocol, or
	// vice-versa. Hence we need to handle v3 type_urls as well here.
	var err error
	url := resp.GetTypeUrl()
	if url != resource.EndpointType {
		return nil, "", "", errors.New(fmt.Sprintf("Resource type %v is not EndpointResource in response from server", resp.GetTypeUrl()))
	}
	return resp.GetResources(), resp.GetVersionInfo(), resp.GetNonce(), err
}
