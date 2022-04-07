package client

import (
	"context"
	v3discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockEds struct {
	mock.Mock
}

func (m *mockEds) Recv() (*v3discoverypb.DiscoveryResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v3discoverypb.DiscoveryResponse), args.Error(1)
}

func (m *mockEds) Send(req *v3discoverypb.DiscoveryRequest) error {
	return m.Called(req).Error(0)
}

func (m *mockEds) Header() (metadata.MD, error) {
	return nil, nil
}

func (m *mockEds) Trailer() metadata.MD {
	return nil
}

func (m *mockEds) CloseSend() error {
	return nil
}

func (m *mockEds) Context() context.Context {
	return nil
}

func (m *mockEds) SendMsg(_ interface{}) error {
	return nil
}

func (m *mockEds) RecvMsg(_ interface{}) error {
	return nil
}

type mockConnection struct {
	mock.Mock
}

func (c *mockConnection) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	return c.Called(ctx, method, args, reply, opts).Error(0)
}

func (c *mockConnection) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	args := c.Called(ctx, desc, method, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(grpc.ClientStream), args.Error(1)
}
