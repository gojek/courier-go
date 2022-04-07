package xds

import (
	"context"
	"errors"
	"fmt"
	"github.com/gojekfarm/courier-go/xds/backoff"
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
)

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
		receiveChan: make(chan []*v3endpointpb.ClusterLoadAssignment),
	}
}

type edsStream v3edsgrpc.EndpointDiscoveryService_StreamEndpointsClient

// Client performs the actual eDS RPCs using the eDS v3 API. It creates an
// EDS stream on which the different types of eDS requests and responses
// are multiplexed.
type Client struct {
	nodeProto  *v3corepb.Node
	cc         grpc.ClientConnInterface
	stream     edsStream
	vsn, nonce string
	xdsTarget  string
	strategy   backoff.Strategy

	done        chan struct{}
	receiveChan chan []*v3endpointpb.ClusterLoadAssignment
}

func (c *Client) startEDSStream(ctx context.Context) (edsStream, error) {
	c.vsn, c.nonce = "", ""
	return v3edsgrpc.NewEndpointDiscoveryServiceClient(c.cc).StreamEndpoints(ctx, grpc.WaitForReady(true))
}

func (c *Client) Start(ctx context.Context) error {
	fmt.Println("restarting stream")
	edsStream, err := c.startEDSStream(ctx)
	if err != nil {
		return err
	}

	c.stream = edsStream
	go c.run(ctx)
	return c.sendRequest([]string{c.xdsTarget}, c.vsn, c.nonce, "")
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

func (c *Client) Receive() <-chan []*v3endpointpb.ClusterLoadAssignment {
	return c.receiveChan
}

func (c *Client) run(ctx context.Context) {
	retries := 0
	streamRunning := true

	//go c.startReceive(ctx)

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

		success := func() bool {
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

				clas := make([]*v3endpointpb.ClusterLoadAssignment, 0, len(resources))
				for _, any := range resources {
					cla := new(v3endpointpb.ClusterLoadAssignment)
					_ = proto.Unmarshal(any.GetValue(), cla)
					clas = append(clas, cla)
				}

				fmt.Println("Pushing to receivechan, ", clas)
				c.receiveChan <- clas
				fmt.Println("Pushed to receivechan, ", clas)
			}
		}()

		if success {
			retries = 0
		}
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
		return nil, "", "", errors.New(fmt.Sprintf("Resource type %v is not EndpointResource in response from server", resp.GetTypeUrl()))
	}
	return resp.GetResources(), resp.GetVersionInfo(), resp.GetNonce(), err
}

func (c *Client) startReceive(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(c.receiveChan)
			return
		default:
			fmt.Println("Restarting recv loop ")
			resp, err := c.stream.Recv()
			if err != nil {
				log.Printf("Recv err %v", err)
				continue
			}
			log.Printf("ADS response received, type: %v\n", resp.GetTypeUrl())
			log.Printf("ADS response received: %+v", resp)

			resources, vsn, nonce, err := c.parseResponse(resp)
			if err != nil {
				c.nack(err)
				continue
			}

			c.vsn, c.nonce = vsn, nonce

			c.ack()

			clas := make([]*v3endpointpb.ClusterLoadAssignment, 0, len(resources))
			for _, any := range resources {
				cla := new(v3endpointpb.ClusterLoadAssignment)
				_ = proto.Unmarshal(any.GetValue(), cla)
				clas = append(clas, cla)
			}

			fmt.Println("Pushing to receivechan, ", clas)
			c.receiveChan <- clas
			fmt.Println("Pushed to receivechan, ", clas)
		}
	}
}

func (c *Client) Done() <-chan struct{} {
	return c.done
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
