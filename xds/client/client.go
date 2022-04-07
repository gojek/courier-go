package client

import (
	"context"
	"errors"
	"fmt"
	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"log"
	"time"

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

// NewClient returns a new eDS client stream using the *grpc.ClientConn provided.
func NewClient(xdsTarget string, nodeProto *v3corepb.Node, cc grpc.ClientConnInterface) (Client, error) {
	v3c := &client{
		nodeProto: nodeProto,
		cc:        cc,
		xdsTarget: xdsTarget,
	}
	return v3c, nil
}

type edsStream v3edsgrpc.EndpointDiscoveryService_StreamEndpointsClient

// client performs the actual eDS RPCs using the eDS v3 API. It creates an
// EDS stream on which the different types of eDS requests and responses
// are multiplexed.
type client struct {
	nodeProto  *v3corepb.Node
	cc         grpc.ClientConnInterface
	stream     edsStream
	vsn, nonce string
	xdsTarget  string

	receiveChan chan []*v3endpointpb.ClusterLoadAssignment
}

func (c *client) startEDSStream(ctx context.Context) (edsStream, error) {
	c.vsn, c.nonce = "", ""
	return v3edsgrpc.NewEndpointDiscoveryServiceClient(c.cc).StreamEndpoints(ctx, grpc.WaitForReady(true))
}

func (c *client) Start(ctx context.Context) error {
	edsStream, err := c.startEDSStream(ctx)
	if err != nil {
		return err
	}

	c.stream = edsStream
	return c.SendRequest([]string{c.xdsTarget}, c.vsn, c.nonce, "")
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
	if err := c.stream.Send(req); err != nil {
		return fmt.Errorf("xds: stream.Send(%+v) failed: %v", req, err)
	}
	return nil
}

func (c *client) Receive() <-chan []*v3endpointpb.ClusterLoadAssignment {
	resp, err := c.stream.Recv()
	if err != nil {
		return nil
	}
	log.Printf("ADS response received, type: %v\n", resp.GetTypeUrl())
	log.Printf("ADS response received: %+v", resp)

	resources, vsn, nonce, err := c.ParseResponse(resp)
	if err != nil {
		//ToDo: errMsg
		c.SendRequest([]string{c.xdsTarget}, c.vsn, c.nonce, "")
		log.Printf("some error")
	}

	c.vsn, c.nonce = vsn, nonce

	clas := make([]*v3endpointpb.ClusterLoadAssignment, 0, len(resources))
	for _, any := range resources {
		cla := &v3endpointpb.ClusterLoadAssignment{}

		proto.Unmarshal(any.GetValue(), cla)

		clas = append(clas, cla)
	}

	return clas
}

func (c *client) run(ctx context.Context) {
	retries := 0
	streamRunning := true

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if retries != 0 {
			timer := time.NewTimer(c.backoff(retries))
			select {
			case <-timer.C:
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return
			}
		}

		retries++

		//restart if xds stream if not already running
		if !streamRunning {
			if err := c.Start(ctx); err != nil {
				//ToDo: Send metrics here and logger here
				log.Printf("xds: Error recovering from broken stream: %v", err)
				streamRunning = false
				continue
			}
			streamRunning = true

			retries = 0
		}
	}
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
