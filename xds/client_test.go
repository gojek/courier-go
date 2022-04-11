package xds

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	v3discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/mock"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/gojekfarm/courier-go/xds/backoff"
	"github.com/gojekfarm/courier-go/xds/log"
)

func TestNewClient(t *testing.T) {
	opts := Options{
		XDSTarget: "cluster",
		NodeProto: &v3corepb.Node{
			Id: "id",
		},
		ClientConn: &mockConnection{},
	}

	client := NewClient(opts)

	switch client.logger.(type) {
	case *log.NoOpLogger:
		if !proto.Equal(client.nodeProto, opts.NodeProto) ||
			opts.XDSTarget != client.xdsTarget ||
			opts.ClientConn != client.cc ||
			reflect.DeepEqual(opts.BackoffStrategy, backoff.DefaultExponential) {
			t.Errorf("NewClient() init error")
		}
	default:
		t.Errorf("NewClient() init error")
	}
}

func TestClient_Done(t *testing.T) {
	done := make(chan struct{})

	c := &Client{
		done: done,
	}

	got := c.Done()

	go func() {
		done <- struct{}{}
	}()

	received := <-got

	if received != struct{}{} {
		t.Errorf("Receiving on done channel failed")
	}
}

func TestClient_Receive(t *testing.T) {
	receiveChan := make(chan []*v3endpointpb.ClusterLoadAssignment)

	c := &Client{
		receiveChan: receiveChan,
		logger:      &log.NoOpLogger{},
	}

	got := c.Receive()
	go func() {
		receiveChan <- []*v3endpointpb.ClusterLoadAssignment{}
	}()

	received := <-got

	if !reflect.DeepEqual(received, []*v3endpointpb.ClusterLoadAssignment{}) {
		t.Errorf("Receiving on receive channel failed")
	}
}

func TestClient_Start(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name           string
		mockEds        func() *mockEds
		mockConnection func(*mockEds) *mockConnection
		wantConnErr    bool
	}{
		{
			name: "success",
			mockEds: func() *mockEds {
				eds := &mockEds{}
				eds.On("RecvMsg", mock.AnythingOfType("*envoy_service_discovery_v3.DiscoveryResponse")).Return(errors.New("some error"))

				eds.On("SendMsg", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "",
						ResponseNonce: "",
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil)

				return eds
			},
			mockConnection: func(eds *mockEds) *mockConnection {
				conn := &mockConnection{}
				conn.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.endpoint.v3.EndpointDiscoveryService/StreamEndpoints",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(eds, nil)

				return conn
			},
		},
		{
			name: "error_initialising_client_stream",
			mockConnection: func(_ *mockEds) *mockConnection {
				conn := &mockConnection{}
				conn.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.endpoint.v3.EndpointDiscoveryService/StreamEndpoints",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(nil, errors.New("some error"))

				return conn
			},
			wantConnErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var eds *mockEds
			if tt.mockEds != nil {
				eds = tt.mockEds()
			}

			cc := tt.mockConnection(eds)

			c := Client{
				cc:        cc,
				strategy:  backoff.DefaultExponential,
				xdsTarget: targets[0],
				vsn:       "",
				nonce:     "",
				logger:    &log.NoOpLogger{},
			}

			err := c.Start(ctx)
			time.Sleep(1 * time.Millisecond)

			if (tt.wantConnErr && err == nil) || (!tt.wantConnErr && err != nil) {
				t.Errorf("Start() returned error: %v when error expected: %v", err, tt.wantConnErr)
			}

			cc.AssertExpectations(t)
			if tt.mockEds != nil {
				eds.AssertExpectations(t)
			}
		})
	}
}

func TestClient_restart(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name           string
		mockEds        func() *mockEds
		mockConnection func(*mockEds) *mockConnection
		wantConnErr    bool
	}{
		{
			name: "success",
			mockEds: func() *mockEds {
				eds := &mockEds{}
				eds.On("SendMsg", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "",
						ResponseNonce: "",
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil)

				return eds
			},
			mockConnection: func(eds *mockEds) *mockConnection {
				conn := &mockConnection{}
				conn.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.endpoint.v3.EndpointDiscoveryService/StreamEndpoints",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(eds, nil)

				return conn
			},
		},
		{
			name: "error_initialising_client_stream",
			mockConnection: func(_ *mockEds) *mockConnection {
				conn := &mockConnection{}
				conn.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.endpoint.v3.EndpointDiscoveryService/StreamEndpoints",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(nil, errors.New("some error"))

				return conn
			},
			wantConnErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var eds *mockEds
			if tt.mockEds != nil {
				eds = tt.mockEds()
			}

			cc := tt.mockConnection(eds)

			c := Client{
				cc:        cc,
				strategy:  backoff.DefaultExponential,
				xdsTarget: targets[0],
				vsn:       "",
				nonce:     "",
				logger:    &log.NoOpLogger{},
			}

			err := c.restart(ctx)
			time.Sleep(1 * time.Millisecond)

			if (tt.wantConnErr && err == nil) || (!tt.wantConnErr && err != nil) {
				t.Errorf("Start() returned error: %v when error expected: %v", err, tt.wantConnErr)
			}

			cc.AssertExpectations(t)
			if tt.mockEds != nil {
				eds.AssertExpectations(t)
			}
		})
	}
}

func TestClient_ack(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name    string
		mockEds func() *mockEds
	}{
		{
			name: "success",
			mockEds: func() *mockEds {
				eds := &mockEds{}
				eds.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "1",
						ResponseNonce: "1",
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil)

				return eds
			},
		},
		{
			name: "failure",
			mockEds: func() *mockEds {
				eds := &mockEds{}
				eds.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "1",
						ResponseNonce: "1",
					}
					return proto.Equal(expectedRequest, req)
				})).Return(errors.New("some_error"))

				return eds
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eds := tt.mockEds()

			c := Client{
				stream:    eds,
				xdsTarget: targets[0],
				vsn:       "1",
				nonce:     "1",
				logger:    &log.NoOpLogger{},
			}

			c.ack()

			eds.AssertExpectations(t)
		})
	}
}

func TestClient_nack(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name    string
		mockEds func() *mockEds
	}{
		{
			name: "success",
			mockEds: func() *mockEds {
				eds := &mockEds{}
				eds.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "1",
						ResponseNonce: "1",
						ErrorDetail: &statuspb.Status{
							Code: int32(codes.InvalidArgument), Message: "nack_error",
						},
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil)

				return eds
			},
		},
		{
			name: "failure",
			mockEds: func() *mockEds {
				eds := &mockEds{}
				eds.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "1",
						ResponseNonce: "1",
						ErrorDetail: &statuspb.Status{
							Code: int32(codes.InvalidArgument), Message: "nack_error",
						},
					}
					return proto.Equal(expectedRequest, req)
				})).Return(errors.New("some_error"))

				return eds
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eds := tt.mockEds()
			c := Client{
				stream:    eds,
				xdsTarget: targets[0],
				vsn:       "1",
				nonce:     "1",
				logger:    &log.NoOpLogger{},
			}

			c.nack(errors.New("nack_error"))

			eds.AssertExpectations(t)
		})
	}
}

func TestClient_run(t *testing.T) {
	targets := []string{"cluster"}
	resources := v3endpointpb.ClusterLoadAssignment{
		ClusterName: "cluster",
	}

	resourceBytes, _ := proto.Marshal(&resources)

	tests := []struct {
		name           string
		mockEds        func() *mockEds
		mockConnection func(*mockEds) *mockConnection
		wantConnErr    bool
	}{
		{
			name: "success_stream_receives_cla",
			mockEds: func() *mockEds {
				eds := &mockEds{}
				eds.On("Recv").Return(&v3discoverypb.DiscoveryResponse{
					TypeUrl:     resource.EndpointType,
					VersionInfo: "",
				}, nil).Once()
				eds.On("Recv").Return(nil, errors.New("some error")).Once()

				eds.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "",
						ResponseNonce: "",
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil).Once()

				eds.On("SendMsg", mock.Anything).Return(nil).Maybe()
				eds.On("RecvMsg", mock.AnythingOfType("*envoy_service_discovery_v3.DiscoveryResponse")).Return(errors.New("some error")).Maybe()

				return eds
			},
			mockConnection: func(eds *mockEds) *mockConnection {
				conn := &mockConnection{}
				conn.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.endpoint.v3.EndpointDiscoveryService/StreamEndpoints",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(eds, nil).Maybe()

				return conn
			},
		},
		{
			name: "when_stream_stopped",
			mockEds: func() *mockEds {
				eds := &mockEds{}
				eds.On("Recv").Return(&v3discoverypb.DiscoveryResponse{
					TypeUrl:     resource.EndpointType,
					VersionInfo: "",
					Resources:   []*anypb.Any{{Value: resourceBytes}},
				}, nil).Once()
				eds.On("Recv").Return(nil, errors.New("some error")).Once()

				eds.On("Send", mock.Anything).Return(nil)

				eds.On("RecvMsg", mock.AnythingOfType("*envoy_service_discovery_v3.DiscoveryResponse")).Return(errors.New("some error")).Maybe()
				eds.On("SendMsg", mock.Anything).Return(nil).Maybe()

				return eds
			},
			mockConnection: func(eds *mockEds) *mockConnection {
				conn := &mockConnection{}
				conn.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.endpoint.v3.EndpointDiscoveryService/StreamEndpoints",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(eds, nil).Maybe()

				return conn
			},
		},
		{
			name: "parse_response_error",
			mockEds: func() *mockEds {
				eds := &mockEds{}
				eds.On("Recv").Return(&v3discoverypb.DiscoveryResponse{
					TypeUrl:     resource.ClusterType,
					VersionInfo: "",
				}, nil).Once()
				eds.On("Recv").Return(&v3discoverypb.DiscoveryResponse{
					TypeUrl:     resource.EndpointType,
					VersionInfo: "",
				}, nil).Once()
				eds.On("Recv").Return(nil, errors.New("somer error")).Once()

				eds.On("Send", mock.Anything).Return(nil)

				eds.On("RecvMsg", mock.AnythingOfType("*envoy_service_discovery_v3.DiscoveryResponse")).Return(errors.New("some error")).Maybe()

				eds.On("SendMsg", mock.Anything).Return(nil).Maybe()

				return eds
			},
			mockConnection: func(eds *mockEds) *mockConnection {
				conn := &mockConnection{}
				conn.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.endpoint.v3.EndpointDiscoveryService/StreamEndpoints",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(eds, nil).Maybe()

				return conn
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())

			var eds *mockEds
			if tt.mockEds != nil {
				eds = tt.mockEds()
			}

			var mc *mockConnection
			if eds != nil && tt.mockConnection != nil {
				mc = tt.mockConnection(eds)
			}

			c := Client{
				stream:      eds,
				strategy:    backoff.DefaultExponential,
				xdsTarget:   targets[0],
				vsn:         "",
				nonce:       "",
				done:        make(chan struct{}),
				receiveChan: make(chan []*v3endpointpb.ClusterLoadAssignment),
				cc:          mc,
				logger:      &log.NoOpLogger{},
			}

			go func() {
				<-c.Receive()
				fmt.Println("channel closed")
				cancel()
			}()

			c.run(ctx)

			if eds != nil {
				eds.AssertExpectations(t)
			}

			if mc != nil {
				mc.AssertExpectations(t)
			}
		})
	}
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

func (m *mockEds) SendMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}

func (m *mockEds) RecvMsg(msg interface{}) error {
	args := m.Called(msg)
	if len(args) > 1 {
		resp := args.Get(1).(*v3discoverypb.DiscoveryResponse)
		v, _ := msg.(*v3discoverypb.DiscoveryResponse)
		*v = v3discoverypb.DiscoveryResponse{
			VersionInfo: resp.VersionInfo,
			Resources:   resp.Resources,
			TypeUrl:     resp.TypeUrl,
			Nonce:       resp.Nonce,
		}
	}

	return args.Error(0)
}
