package xds

import (
	"context"
	"fmt"
	"log"
	"time"

	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	v3discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	v3edsgrpc "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/proto"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/gojekfarm/courier-go/xds/backoff"
)

// Options specifies options to be provided for initialising the xds client
type Options struct {
	XDSTarget       string
	NodeProto       *v3corepb.Node
	ClientConn      grpc.ClientConnInterface
	BackoffStrategy backoff.Strategy
}

// NewClient returns a new eDS client stream using the *grpc.ClientConn provided.
func NewClient(opts Options) *Client {
	return &Client{
		xdsTarget:   opts.XDSTarget,
		cc:          opts.ClientConn,
		nodeProto:   opts.NodeProto,
		strategy:    opts.BackoffStrategy,
		done:        make(chan struct{}),
		receiveChan: make(chan []*v3endpointpb.ClusterLoadAssignment),
	}
}

type edsStream v3edsgrpc.EndpointDiscoveryService_StreamEndpointsClient

// Client performs the actual eDS RPCs using the eDS v3 API. It creates an
// EDS stream on which the xdsTarget resources are received.
type Client struct {
	nodeProto *v3corepb.Node
	cc        grpc.ClientConnInterface
	strategy  backoff.Strategy
	stream    edsStream

	xdsTarget  string
	vsn, nonce string

	done        chan struct{}
	receiveChan chan []*v3endpointpb.ClusterLoadAssignment
}

// Start sends the first discoveryRequest to the management server and starts the receive loop
func (c *Client) Start(ctx context.Context) error {
	fmt.Println("starting stream")

	stream, err := c.startEDSStream(ctx)

	if err != nil {
		return err
	}

	c.stream = stream
	go c.run(ctx)

	return c.sendRequest([]string{c.xdsTarget}, c.vsn, c.nonce, "")
}

// Receive returns a channel where ClusterLoadAssignment resource updates can be received
func (c *Client) Receive() <-chan []*v3endpointpb.ClusterLoadAssignment {
	return c.receiveChan
}

// Done returns a channel which is closed when the run loop stops due to context expiry
func (c *Client) Done() <-chan struct{} {
	return c.done
}

func (c *Client) restart(ctx context.Context) error {
	fmt.Println("restarting stream")

	stream, err := c.startEDSStream(ctx)

	if err != nil {
		return err
	}

	c.stream = stream

	return c.sendRequest([]string{c.xdsTarget}, c.vsn, c.nonce, "")
}

func (c *Client) startEDSStream(ctx context.Context) (edsStream, error) {
	c.vsn, c.nonce = "", ""

	return v3edsgrpc.NewEndpointDiscoveryServiceClient(c.cc).StreamEndpoints(ctx, grpc.WaitForReady(true))
}

func (c *Client) sendRequest(resourceNames []string, version, nonce, errMsg string) error {
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

func (c *Client) run(ctx context.Context) {
	retries := 0
	streamRunning := true

	for {
		select {
		case <-ctx.Done():
			close(c.done)

			return
		default:
		}

		if retries != 0 {
			timer := time.NewTimer(c.strategy.Backoff(retries))
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

		//restart if xds stream is not already running
		if !streamRunning {
			//Do not use c.Start here as it calls run() again
			if err := c.restart(ctx); err != nil {
				//ToDo: Send metrics here and logger here
				log.Printf("xds: Error recovering from broken stream: %v", err)

				continue
			}

			retries = 0
		}

		if c.recv() {
			retries = 0
		}

		fmt.Println("Received from recv")

		streamRunning = false
	}
}

func (c *Client) parseResponse(r proto.Message) ([]*anypb.Any, string, string, error) {
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
		return nil, "", "", fmt.Errorf("resource type %v is not EndpointResource in response from server", resp.GetTypeUrl())
	}

	return resp.GetResources(), resp.GetVersionInfo(), resp.GetNonce(), err
}

func (c *Client) nack(err error) {
	if err := c.sendRequest([]string{c.xdsTarget}, c.vsn, c.nonce, err.Error()); err != nil {
		log.Printf("SendRequest: Nack err %v", err)
	}
}

func (c *Client) ack() {
	if err := c.sendRequest([]string{c.xdsTarget}, c.vsn, c.nonce, ""); err != nil {
		log.Printf("SendRequest: Ack err %v", err)
	}
}

func (c *Client) recv() bool {
	success := false

	for {
		fmt.Println("Restarting recv loop ")

		resp, err := c.stream.Recv()

		if err != nil {
			log.Printf("Recv err %v", err)

			return success
		}

		log.Printf("ADS response received, type: %v\n", resp.GetTypeUrl())
		log.Printf("ADS response received: %+v", resp)

		resources, vsn, nonce, err := c.parseResponse(resp)

		if err != nil {
			c.nack(err)

			success = true

			continue
		}

		c.vsn, c.nonce = vsn, nonce

		c.ack()

		success = true

		clusterLoadAssignments := make([]*v3endpointpb.ClusterLoadAssignment, 0, len(resources))

		for _, any := range resources {
			cla := new(v3endpointpb.ClusterLoadAssignment)
			_ = proto.Unmarshal(any.GetValue(), cla)
			clusterLoadAssignments = append(clusterLoadAssignments, cla)
		}

		fmt.Println("Pushing to receivechan, ", clusterLoadAssignments)
		c.receiveChan <- clusterLoadAssignments
		fmt.Println("Pushed to receivechan, ", clusterLoadAssignments)
	}
}
