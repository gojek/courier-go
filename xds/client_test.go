package xds

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/gojek/courier-go/xds/backoff"
	"github.com/gojek/courier-go/xds/log"
)

func TestNewClient(t *testing.T) {
	mc := newMockConnection(t)

	opts := Options{
		XDSTarget:  "cluster",
		NodeProto:  &v3corepb.Node{Id: "id"},
		ClientConn: mc,
	}

	client := NewClient(opts)

	switch client.log.(type) {
	case *log.NoOpLogger:
		assert.True(t, proto.Equal(opts.NodeProto, client.nodeProto))
		assert.Equal(t, opts.XDSTarget, client.xdsTarget)
		assert.Equal(t, opts.ClientConn, client.cc)
		assert.Equal(t, &backoff.DefaultExponential, client.stream.(*reloadingStream).strategy)
	default:
		assert.Fail(t, "NewClient() init error")
	}

	mc.AssertExpectations(t)
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
		log:         &log.NoOpLogger{},
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
		name                string
		mockReloadingStream func(*mock.Mock, *sync.WaitGroup)
		want                assert.ErrorAssertionFunc
	}{
		{
			name: "Success",
			mockReloadingStream: func(m *mock.Mock, wg *sync.WaitGroup) {
				m.On("createStream", mock.Anything).Return(nil)

				m.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil)

				wg.Add(1)
				m.On("startReloader", mock.Anything).After(time.Second).Run(func(_ mock.Arguments) {
					wg.Done()
				})
			},
			want: assert.NoError,
		},
		{
			name: "createStreamError",
			mockReloadingStream: func(m *mock.Mock, wg *sync.WaitGroup) {
				m.On("createStream", mock.Anything).Return(errors.New("some error"))
			},
			want: assert.Error,
		},
		{
			name: "Error",
			mockReloadingStream: func(m *mock.Mock, wg *sync.WaitGroup) {
				m.On("createStream", mock.Anything).Return(nil)

				m.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
					}
					return proto.Equal(expectedRequest, req)
				})).Return(errors.New("send error"))

				wg.Add(1)
				m.On("startReloader", mock.Anything).After(time.Second).Run(func(_ mock.Arguments) {
					wg.Done()
				})
			},
			want: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			mrs := newMockReloadingStream(t)
			wg := &sync.WaitGroup{}

			if tt.mockReloadingStream != nil {
				tt.mockReloadingStream(&mrs.Mock, wg)
			}

			c := Client{
				stream:      mrs,
				xdsTarget:   targets[0],
				log:         &log.NoOpLogger{},
				done:        make(chan struct{}, 1),
				receiveChan: make(chan []*v3endpointpb.ClusterLoadAssignment, 1),
			}

			tt.want(t, c.Start(ctx))

			wg.Wait()
			mrs.AssertExpectations(t)
		})
	}
}

func TestClient_ack(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name                string
		mockReloadingStream func(*mock.Mock)
	}{
		{
			name: "Success",
			mockReloadingStream: func(m *mock.Mock) {
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
			name: "Failure",
			mockReloadingStream: func(m *mock.Mock) {
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
			mrs := newMockReloadingStream(t)
			tt.mockReloadingStream(&mrs.Mock)

			c := Client{
				stream:    mrs,
				xdsTarget: targets[0],
				vsn:       "1",
				nonce:     "1",
				log:       &log.NoOpLogger{},
			}

			c.ack()

			mrs.AssertExpectations(t)
		})
	}
}

func TestClient_nack(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name                string
		mockReloadingStream func(*mock.Mock)
	}{
		{
			name: "Success",
			mockReloadingStream: func(m *mock.Mock) {
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
			name: "Failure",
			mockReloadingStream: func(m *mock.Mock) {
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
			mrs := newMockReloadingStream(t)
			tt.mockReloadingStream(&mrs.Mock)
			c := Client{
				stream:    mrs,
				xdsTarget: targets[0],
				vsn:       "1",
				nonce:     "1",
				log:       &log.NoOpLogger{},
			}

			c.nack(errors.New("nack_error"))

			mrs.AssertExpectations(t)
		})
	}
}

func TestClient_run(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name                string
		ctx                 func() (context.Context, context.CancelFunc)
		mockReloadingStream func(*mock.Mock)
	}{
		{
			name: "CancelledContext",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
		},
		{
			name: "StreamRecvError",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 15*time.Millisecond)
			},
			mockReloadingStream: func(m *mock.Mock) {
				m.On("Recv").Return(nil, errors.New("recv error")).
					After(10 * time.Millisecond).
					Twice()
			},
		},
		{
			name: "IncorrectURLType",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 15*time.Millisecond)
			},
			mockReloadingStream: func(m *mock.Mock) {
				m.On("Recv").Return(&v3discoverypb.DiscoveryResponse{TypeUrl: resource.RouteType}, nil).
					After(10 * time.Millisecond).
					Once()

				m.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
						ErrorDetail: &statuspb.Status{
							Code: int32(codes.InvalidArgument),
							Message: fmt.Sprintf(
								"xds: resource type (%s) is not EndpointResource in server response",
								resource.RouteType,
							),
						},
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil).
					After(10 * time.Millisecond).
					Once()
			},
		},
		{
			name: "SuccessStreamReceivesClusterLoadAssignment",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 15*time.Millisecond)
			},
			mockReloadingStream: func(m *mock.Mock) {
				m.On("Recv").Return(&v3discoverypb.DiscoveryResponse{TypeUrl: resource.EndpointType}, nil).
					After(10 * time.Millisecond).
					Once()

				m.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil).
					After(10 * time.Millisecond).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.ctx()
			defer cancel()

			mrs := newMockReloadingStream(t)
			if tt.mockReloadingStream != nil {
				tt.mockReloadingStream(&mrs.Mock)
			}

			c := Client{
				stream:      mrs,
				xdsTarget:   targets[0],
				done:        make(chan struct{}),
				receiveChan: make(chan []*v3endpointpb.ClusterLoadAssignment),
				log:         &log.NoOpLogger{},
			}

			go func() {
				<-c.Receive()
				cancel()
			}()

			c.run(ctx)

			mrs.AssertExpectations(t)
		})
	}
}

func TestClient_processResources(t *testing.T) {
	in1 := &v3endpointpb.ClusterLoadAssignment{
		ClusterName: "cluster-1",
		Endpoints: []*v3endpointpb.LocalityLbEndpoints{
			{Locality: &v3corepb.Locality{Region: "asia-east1"}},
		},
	}
	in2 := &v3endpointpb.ClusterLoadAssignment{
		ClusterName: "cluster-1",
		Endpoints: []*v3endpointpb.LocalityLbEndpoints{
			{Locality: &v3corepb.Locality{Region: "asia-east1"}},
		},
	}

	a1, _ := anypb.New(in1)
	a2, _ := anypb.New(in2)

	tests := []struct {
		name      string
		resources []*anypb.Any
		want      []*v3endpointpb.ClusterLoadAssignment
	}{
		{
			name: "Single",
			resources: []*anypb.Any{
				a1,
			},
			want: []*v3endpointpb.ClusterLoadAssignment{in1},
		},
		{
			name: "Double",
			resources: []*anypb.Any{
				a1, a2,
			},
			want: []*v3endpointpb.ClusterLoadAssignment{in1, in2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{receiveChan: make(chan []*v3endpointpb.ClusterLoadAssignment, 1)}

			go c.processResources(tt.resources)

			for i, cla := range <-c.receiveChan {
				assert.True(t, proto.Equal(tt.want[i], cla))
			}
		})
	}
}

func TestClient_onStreamReConnect(t *testing.T) {
	targets := []string{"cluster"}

	tests := []struct {
		name                string
		mockReloadingStream func(m *mock.Mock)
	}{
		{
			name: "SuccessInitialSend",
			mockReloadingStream: func(m *mock.Mock) {
				m.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
					}
					return proto.Equal(expectedRequest, req)
				})).Return(nil).
					After(10 * time.Millisecond).
					Once()
			},
		},
		{
			name: "FailureInitialSend",
			mockReloadingStream: func(m *mock.Mock) {
				m.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						TypeUrl:       resource.EndpointType,
						ResourceNames: targets,
					}
					return proto.Equal(expectedRequest, req)
				})).Return(errors.New("send error")).
					After(10 * time.Millisecond).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mrs := newMockReloadingStream(t)
			tt.mockReloadingStream(&mrs.Mock)

			c := &Client{xdsTarget: targets[0], stream: mrs, log: &log.NoOpLogger{}}

			c.onStreamReConnect(errors.New("stream closed"))

			mrs.AssertExpectations(t)
		})
	}
}

// Mocks

func newMockReloadingStream(t *testing.T) *mockReloadingStream {
	m := &mockReloadingStream{}
	m.Test(t)
	return m
}

type mockReloadingStream struct {
	mock.Mock
}

func (m *mockReloadingStream) Send(req *v3discoverypb.DiscoveryRequest) error {
	return m.Called(req).Error(0)
}

func (m *mockReloadingStream) Recv() (*v3discoverypb.DiscoveryResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v3discoverypb.DiscoveryResponse), args.Error(1)
}

func (m *mockReloadingStream) startReloader(ctx context.Context) {
	m.Called(ctx)
}

func (m *mockReloadingStream) createStream(ctx context.Context) error {
	return m.Called(ctx).Error(0)
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
