package xds

import (
	"context"
	"errors"
	"testing"
	"time"

	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	v3discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/gojek/courier-go/xds/backoff"
	"github.com/gojek/courier-go/xds/log"
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
		assert.True(t, proto.Equal(opts.NodeProto, client.nodeProto))
		assert.Equal(t, opts.XDSTarget, client.xdsTarget)
		assert.Equal(t, opts.ClientConn, client.cc)
		assert.Equal(t, &backoff.DefaultExponential, client.strategy)
	default:
		assert.Fail(t, "NewClient() init error")
	}
}

func TestClient_Done(t *testing.T) {
	done := make(chan struct{}, 1)

	c := &Client{
		done: done,
	}

	got := c.Done()

	go func() {
		done <- struct{}{}
	}()

	select {
	case received := <-got:
		assert.NotNil(t, received)
	case <-time.After(3 * time.Second):
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

	select {
	case received := <-got:
		assert.Equal(t, []*v3endpointpb.ClusterLoadAssignment{}, received)
	case <-time.After(3 * time.Second):
		t.Errorf("Receiving on receive channel failed")
	}
}

func TestClient_Start(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name           string
		mockAds        func(*mock.Mock)
		mockConnection func(*mock.Mock, *mockAds)
		wantConnErr    bool
	}{
		{
			name: "success",
			mockAds: func(m *mock.Mock) {
				m.On("RecvMsg", mock.AnythingOfType("*envoy_service_discovery_v3.DiscoveryResponse")).Return(errors.New("some error"))

				m.On("SendMsg", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "",
						ResponseNonce: "",
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil)
			},
			mockConnection: func(m *mock.Mock, ads *mockAds) {
				m.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.discovery.v3.AggregatedDiscoveryService/StreamAggregatedResources",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(ads, nil)
			},
		},
		{
			name: "error_initialising_client_stream",
			mockConnection: func(m *mock.Mock, _ *mockAds) {
				m.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.discovery.v3.AggregatedDiscoveryService/StreamAggregatedResources",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(nil, errors.New("some error"))
			},
			wantConnErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ads := newMockAds(t)
			if tt.mockAds != nil {
				tt.mockAds(&ads.Mock)
			}

			cc := newMockConnection(t)
			tt.mockConnection(&cc.Mock, ads)

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
			ads.AssertExpectations(t)
		})
	}
}

func TestClient_restart(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name           string
		mockAds        func(*mock.Mock)
		mockConnection func(*mock.Mock, *mockAds)
		wantConnErr    bool
	}{
		{
			name: "success",
			mockAds: func(m *mock.Mock) {
				m.On("SendMsg", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "",
						ResponseNonce: "",
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil)
			},
			mockConnection: func(m *mock.Mock, ads *mockAds) {
				m.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.discovery.v3.AggregatedDiscoveryService/StreamAggregatedResources",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(ads, nil)
			},
		},
		{
			name: "error_initialising_client_stream",
			mockConnection: func(m *mock.Mock, _ *mockAds) {
				m.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.discovery.v3.AggregatedDiscoveryService/StreamAggregatedResources",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(nil, errors.New("some error"))
			},
			wantConnErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ads := newMockAds(t)
			if tt.mockAds != nil {
				tt.mockAds(&ads.Mock)
			}

			cc := newMockConnection(t)
			tt.mockConnection(&cc.Mock, ads)

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
			ads.AssertExpectations(t)
		})
	}
}

func TestClient_ack(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name    string
		mockAds func(*mock.Mock)
	}{
		{
			name: "success",
			mockAds: func(m *mock.Mock) {
				m.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "1",
						ResponseNonce: "1",
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil)
			},
		},
		{
			name: "failure",
			mockAds: func(m *mock.Mock) {
				m.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "1",
						ResponseNonce: "1",
					}
					return proto.Equal(expectedRequest, req)
				})).Return(errors.New("some_error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ads := newMockAds(t)
			tt.mockAds(&ads.Mock)

			c := Client{
				stream:    ads,
				xdsTarget: targets[0],
				vsn:       "1",
				nonce:     "1",
				logger:    &log.NoOpLogger{},
			}

			c.ack()

			ads.AssertExpectations(t)
		})
	}
}

func TestClient_nack(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name    string
		mockAds func(*mock.Mock)
	}{
		{
			name: "success",
			mockAds: func(m *mock.Mock) {
				m.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
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
			},
		},
		{
			name: "failure",
			mockAds: func(m *mock.Mock) {
				m.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ads := newMockAds(t)
			tt.mockAds(&ads.Mock)
			c := Client{
				stream:    ads,
				xdsTarget: targets[0],
				vsn:       "1",
				nonce:     "1",
				logger:    &log.NoOpLogger{},
			}

			c.nack(errors.New("nack_error"))

			ads.AssertExpectations(t)
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
		mockAds        func(*mock.Mock)
		mockConnection func(*mock.Mock, *mockAds)
		wantConnErr    bool
	}{
		{
			name: "success_stream_receives_cla",
			mockAds: func(m *mock.Mock) {
				m.On("Recv").Return(&v3discoverypb.DiscoveryResponse{
					TypeUrl:     resource.EndpointType,
					VersionInfo: "",
				}, nil).Once()
				m.On("Recv").Return(nil, errors.New("some error")).Once()

				m.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						VersionInfo:   "",
						ResponseNonce: "",
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil).Once()

				m.On("SendMsg", mock.Anything).Return(nil).Maybe()
				m.On("RecvMsg", mock.AnythingOfType("*envoy_service_discovery_v3.DiscoveryResponse")).Return(errors.New("some error")).Maybe()
			},
			mockConnection: func(m *mock.Mock, ads *mockAds) {
				m.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.discovery.v3.AggregatedDiscoveryService/StreamAggregatedResources",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(ads, nil).Maybe()
			},
		},
		{
			name: "when_stream_stopped",
			mockAds: func(m *mock.Mock) {
				m.On("Recv").Return(&v3discoverypb.DiscoveryResponse{
					TypeUrl:     resource.EndpointType,
					VersionInfo: "",
					Resources:   []*anypb.Any{{Value: resourceBytes}},
				}, nil).Once()
				m.On("Recv").Return(nil, errors.New("some error")).Once()

				m.On("Send", mock.Anything).Return(nil)

				m.On("RecvMsg", mock.AnythingOfType("*envoy_service_discovery_v3.DiscoveryResponse")).Return(errors.New("some error")).Maybe()
				m.On("SendMsg", mock.Anything).Return(nil).Maybe()
			},
			mockConnection: func(m *mock.Mock, ads *mockAds) {
				m.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.discovery.v3.AggregatedDiscoveryService/StreamAggregatedResources",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(ads, nil).Maybe()
			},
		},
		{
			name: "parse_response_error",
			mockAds: func(m *mock.Mock) {
				m.On("Recv").Return(&v3discoverypb.DiscoveryResponse{
					TypeUrl:     resource.ClusterType,
					VersionInfo: "",
				}, nil).Once()
				m.On("Recv").Return(&v3discoverypb.DiscoveryResponse{
					TypeUrl:     resource.EndpointType,
					VersionInfo: "",
				}, nil).Once()
				m.On("Recv").Return(nil, errors.New("somer error")).Once()

				m.On("Send", mock.Anything).Return(nil)

				m.On("RecvMsg", mock.AnythingOfType("*envoy_service_discovery_v3.DiscoveryResponse")).Return(errors.New("some error")).Maybe()

				m.On("SendMsg", mock.Anything).Return(nil).Maybe()
			},
			mockConnection: func(m *mock.Mock, ads *mockAds) {
				m.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.discovery.v3.AggregatedDiscoveryService/StreamAggregatedResources",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(ads, nil).Maybe()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())

			ads := newMockAds(t)
			if tt.mockAds != nil {
				tt.mockAds(&ads.Mock)
			}

			mc := newMockConnection(t)
			if tt.mockConnection != nil {
				tt.mockConnection(&mc.Mock, ads)
			}

			c := Client{
				stream:      ads,
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
				cancel()
			}()

			c.run(ctx)

			ads.AssertExpectations(t)
			mc.AssertExpectations(t)
		})
	}
}

func newMockConnection(t *testing.T) *mockConnection {
	m := &mockConnection{}
	m.Test(t)
	return m
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

func newMockAds(t *testing.T) *mockAds {
	m := &mockAds{}
	m.Test(t)
	return m
}

type mockAds struct {
	mock.Mock
}

func (m *mockAds) Recv() (*v3discoverypb.DiscoveryResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v3discoverypb.DiscoveryResponse), args.Error(1)
}

func (m *mockAds) Send(req *v3discoverypb.DiscoveryRequest) error {
	return m.Called(req).Error(0)
}

func (m *mockAds) Header() (metadata.MD, error) {
	return nil, nil
}

func (m *mockAds) Trailer() metadata.MD {
	return nil
}

func (m *mockAds) CloseSend() error {
	return nil
}

func (m *mockAds) Context() context.Context {
	return nil
}

func (m *mockAds) SendMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}

func (m *mockAds) RecvMsg(msg interface{}) error {
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
