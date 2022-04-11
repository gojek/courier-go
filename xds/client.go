package xds

import (
	"context"
	"fmt"
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
	"github.com/gojekfarm/courier-go/xds/log"
)

// Options specifies options to be provided for initialising the xds client
type Options struct {
	XDSTarget       string
	NodeProto       *v3corepb.Node
	ClientConn      grpc.ClientConnInterface
	BackoffStrategy backoff.Strategy
	Logger          log.Logger
}

// NewClient returns a new eDS client stream using the *grpc.ClientConn provided.
func NewClient(opts Options) *Client {
	opts = setOpts(opts)

	return &Client{
		xdsTarget:   opts.XDSTarget,
		cc:          opts.ClientConn,
		nodeProto:   opts.NodeProto,
		strategy:    opts.BackoffStrategy,
		done:        make(chan struct{}),
		receiveChan: make(chan []*v3endpointpb.ClusterLoadAssignment),
		logger:      opts.Logger,
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
	logger    log.Logger

	xdsTarget  string
	vsn, nonce string

	done        chan struct{}
	receiveChan chan []*v3endpointpb.ClusterLoadAssignment
}

// Start sends the first discoveryRequest to the management server and starts the receive loop
func (c *Client) Start(ctx context.Context) error {
	c.logger.Info("xds: Starting eds stream for:", "node", c.nodeProto, "target", c.xdsTarget)

	stream, err := c.startEDSStream(ctx)

	if err != nil {
		return err
	}

	c.stream = stream
	go c.run(ctx)

	if err = c.sendRequest([]string{c.xdsTarget}, c.vsn, c.nonce, ""); err != nil {
		return fmt.Errorf("xds: client.Start: failed to sendRequest: %v", err)
	}

	return nil
}

// Receive returns a channel where ClusterLoadAssignment resource updates can be received
func (c *Client) Receive() <-chan []*v3endpointpb.ClusterLoadAssignment {
	return c.receiveChan
}

// Done returns a channel which is closed when the run loop stops due to context expiry
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// restart invokes the StreamEndpoints method on the EDS client and sends subscription request
func (c *Client) restart(ctx context.Context) error {
	c.logger.Info("xds: Restarting eds stream for:", "node", c.nodeProto, "target", c.xdsTarget)

	stream, err := c.startEDSStream(ctx)

	if err != nil {
		return err
	}

	c.stream = stream

	if err = c.sendRequest([]string{c.xdsTarget}, c.vsn, c.nonce, ""); err != nil {
		return fmt.Errorf("xds: client.restart: failed to sendRequest: %v", err)
	}

	return nil
}

// restart invokes the StreamEndpoints method on the EDS client
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
				//ToDo: Consider exposing metrics to track this
				c.logger.Error(err, "xds: Failure to recover from broken stream:", "retries", retries)

				continue
			}

			retries = 0
		}

		if c.recv() {
			retries = 0
		}

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
		return nil, "", "", fmt.Errorf("xds: resource type (%v) is not EndpointResource in server response",
			resp.GetTypeUrl())
	}

	return resp.GetResources(), resp.GetVersionInfo(), resp.GetNonce(), err
}

func (c *Client) nack(err error) {
	if e := c.sendRequest([]string{c.xdsTarget}, c.vsn, c.nonce, err.Error()); e != nil {
		c.logger.Error(e, "xds: Nack: SendRequest error", "version", c.vsn, "nonce", c.nonce)
	}
}

func (c *Client) ack() {
	if err := c.sendRequest([]string{c.xdsTarget}, c.vsn, c.nonce, ""); err != nil {
		c.logger.Error(err, "xds: Ack: SendRequest error", "version", c.vsn, "nonce", c.nonce)
	}
}

func (c *Client) recv() bool {
	success := false

	for {
		c.logger.Debug("xds: Restarting recv loop ")

		resp, err := c.stream.Recv()

		if err != nil {
			c.logger.Error(err, "xds: error while recv()")

			return success
		}

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

		c.receiveChan <- clusterLoadAssignments
	}
}

func setOpts(opts Options) Options {
	if opts.Logger == nil {
		opts.Logger = &log.NoOpLogger{}
	}

	if opts.BackoffStrategy == nil {
		opts.BackoffStrategy = &backoff.DefaultExponential
	}

	return opts
}
