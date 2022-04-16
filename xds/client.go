package xds

import (
	"context"
	"fmt"
	"time"

	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	v3discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/proto"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/gojek/courier-go/xds/backoff"
	"github.com/gojek/courier-go/xds/log"
)

// stream is used to mock reloadingStream
type stream interface {
	Send(req *v3discoverypb.DiscoveryRequest) error
	Recv() (*v3discoverypb.DiscoveryResponse, error)
	startReloader(ctx context.Context)
}

// Options specifies options to be provided for initialising the xds client
type Options struct {
	XDSTarget       string
	NodeProto       *v3corepb.Node
	ClientConn      grpc.ClientConnInterface
	BackoffStrategy backoff.Strategy
	Logger          log.Logger
}

// NewClient returns a new ADS client stream using the *grpc.ClientConn provided.
func NewClient(opts Options) (*Client, error) {
	opts = setDefaultOpts(opts)

	c := &Client{
		xdsTarget:   opts.XDSTarget,
		cc:          opts.ClientConn,
		nodeProto:   opts.NodeProto,
		done:        make(chan struct{}),
		receiveChan: make(chan []*v3endpointpb.ClusterLoadAssignment),
		log:         opts.Logger,
	}

	cc := opts.ClientConn
	rs := &reloadingStream{
		reloadCh:    make(chan error, 1),
		onReConnect: c.onStreamReConnect,
		connTimeout: 10 * time.Second,
		log:         opts.Logger,
		strategy:    opts.BackoffStrategy,
		ns: func() v3discoverypb.AggregatedDiscoveryServiceClient {
			return v3discoverypb.NewAggregatedDiscoveryServiceClient(cc)
		},
	}

	c.stream = rs

	return c, rs.createStream()
}

// Client performs the actual ADS RPCs using the ADS v3 API. It creates an
// ADS stream on which the xdsTarget resources are received.
type Client struct {
	nodeProto *v3corepb.Node
	cc        grpc.ClientConnInterface
	stream    stream
	log       log.Logger

	xdsTarget  string
	vsn, nonce string

	done        chan struct{}
	receiveChan chan []*v3endpointpb.ClusterLoadAssignment
}

// Receive returns a channel where ClusterLoadAssignment resource updates can be received
func (c *Client) Receive() <-chan []*v3endpointpb.ClusterLoadAssignment {
	return c.receiveChan
}

// Done returns a channel which is closed when the run loop stops due to context expiry
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// Start will wait updates from control plane, it is non-blocking
func (c *Client) Start(ctx context.Context) error {
	c.log.Info("xds: Starting ads client", "node", c.nodeProto, "target", c.xdsTarget)

	go c.stream.startReloader(ctx)
	go c.run(ctx)

	return c.sendRequest(nil)
}

func (c *Client) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(c.done)
			close(c.receiveChan)

			return
		default:
			resp, err := c.stream.Recv()
			if err != nil {
				c.log.Error(err, "xds: error stream.Recv()")

				break
			}

			if url := resp.GetTypeUrl(); url != resource.EndpointType {
				c.nack(fmt.Errorf("xds: resource type (%s) is not EndpointResource in server response", url))

				break
			}

			c.vsn, c.nonce = resp.GetVersionInfo(), resp.GetNonce()

			c.ack()

			c.processResources(resp.GetResources())
		}
	}
}

func (c *Client) nack(err error) {
	if e := c.sendRequest(err); e != nil {
		c.log.Error(e, "xds: Nack: SendRequest error", "version", c.vsn, "nonce", c.nonce)
	}
}

func (c *Client) ack() {
	if err := c.sendRequest(nil); err != nil {
		c.log.Error(err, "xds: Ack: SendRequest error", "version", c.vsn, "nonce", c.nonce)
	}
}

func (c *Client) onStreamReConnect(err error) {
	c.log.Error(err, "reconnected ADS stream")
	c.vsn, c.nonce = "", ""

	if err := c.sendRequest(nil); err != nil {
		c.log.Error(err, "unable to send initial request")
	}
}

func (c *Client) sendRequest(err error) error {
	req := &v3discoverypb.DiscoveryRequest{
		Node:          c.nodeProto,
		TypeUrl:       resource.EndpointType,
		ResourceNames: []string{c.xdsTarget},
		VersionInfo:   c.vsn,
		ResponseNonce: c.nonce,
	}

	if err != nil {
		req.ErrorDetail = &statuspb.Status{
			Code: int32(codes.InvalidArgument), Message: err.Error(),
		}
	}

	if err := c.stream.Send(req); err != nil {
		return fmt.Errorf("xds: stream.Send(%+v) failed: %v", req, err)
	}

	return nil
}

func (c *Client) processResources(resources []*anypb.Any) {
	clusterLoadAssignments := make([]*v3endpointpb.ClusterLoadAssignment, 0, len(resources))

	for _, any := range resources {
		cla := new(v3endpointpb.ClusterLoadAssignment)
		_ = proto.Unmarshal(any.GetValue(), cla)
		clusterLoadAssignments = append(clusterLoadAssignments, cla)
	}

	c.receiveChan <- clusterLoadAssignments
}

func setDefaultOpts(opts Options) Options {
	if opts.Logger == nil {
		opts.Logger = &log.NoOpLogger{}
	}

	if opts.BackoffStrategy == nil {
		opts.BackoffStrategy = &backoff.DefaultExponential
	}

	return opts
}
