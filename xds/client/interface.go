package client

import (
	"context"
	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
)

type Client interface {
	// Start returns a new xDS client stream specific to the underlying
	// transport protocol version.
	Start(ctx context.Context) error
	// SendRequest constructs and sends out a DiscoveryRequest message specific
	// to the underlying transport protocol version.
	SendRequest(resourceNames []string, version, nonce, errMsg string) error
	// ParseResponse type asserts message to the versioned response, and
	// retrieves the fields.
	ParseResponse(r proto.Message) ([]*anypb.Any, string, string, error)
	// Receive receives a response uses the provided eDS stream. It blocks till
	// a response is received from the management server.
	Receive() []*v3endpointpb.ClusterLoadAssignment
}

type restartFunc func(context.Context, grpc.ClientConnInterface) error

type sendFunc func(resourceNames []string, version string, nonce string, errMsg string) error

type parseFunc func(r proto.Message) ([]*anypb.Any, string, string, error)

type receiveFunc func() (proto.Message, error)
