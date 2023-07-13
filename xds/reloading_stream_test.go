package xds

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/gojek/courier-go/xds/backoff"
	"github.com/gojek/courier-go/xds/log"
)

func Test_reloadingStream_Send(t *testing.T) {
	tests := []struct {
		name         string
		req          *v3discoverypb.DiscoveryRequest
		newStream    func(*testing.T, *mock.Mock) func()
		wantStartErr assert.ErrorAssertionFunc
		wantSendErr  assert.ErrorAssertionFunc
	}{
		{
			name: "StreamCreateError",
			newStream: func(t *testing.T, m *mock.Mock) func() {
				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(nil, errors.New("can't create a stream"))
				return nil
			},
			wantStartErr: assert.Error,
		},
		{
			name: "SendSuccess",
			req: &v3discoverypb.DiscoveryRequest{
				Node:          &corev3.Node{Id: "node~127.0.0.1"},
				ResourceNames: []string{"cluster.domain"},
				TypeUrl:       resource.EndpointType,
			},
			newStream: func(t *testing.T, m *mock.Mock) func() {
				mads := newMockAdsStream(t)
				mads.On("Send", &v3discoverypb.DiscoveryRequest{
					Node:          &corev3.Node{Id: "node~127.0.0.1"},
					ResourceNames: []string{"cluster.domain"},
					TypeUrl:       resource.EndpointType,
				}).Return(nil)

				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(mads, nil)
				return func() { mads.AssertExpectations(t) }
			},
			wantStartErr: assert.NoError,
			wantSendErr:  assert.NoError,
		},
		{
			name: "SendError",
			req: &v3discoverypb.DiscoveryRequest{
				Node:          &corev3.Node{Id: "node~127.0.0.1"},
				ResourceNames: []string{"cluster.domain"},
				TypeUrl:       resource.EndpointType,
			},
			newStream: func(t *testing.T, m *mock.Mock) func() {
				mads := newMockAdsStream(t)
				mads.On("Send", &v3discoverypb.DiscoveryRequest{
					Node:          &corev3.Node{Id: "node~127.0.0.1"},
					ResourceNames: []string{"cluster.domain"},
					TypeUrl:       resource.EndpointType,
				}).Return(errors.New("send error"))

				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(mads, nil)
				return func() { mads.AssertExpectations(t) }
			},
			wantStartErr: assert.NoError,
			wantSendErr:  assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mads := newMockADSClient(t)
			if tt.newStream != nil {
				ae := tt.newStream(t, &mads.Mock)
				if ae != nil {
					defer ae()
				}
			}

			r := &reloadingStream{
				ns:          mads,
				strategy:    &backoff.DefaultExponential,
				reloadCh:    make(chan error, 1),
				connTimeout: time.Second,
				log:         &log.NoOpLogger{},
			}

			if tt.wantStartErr != nil {
				tt.wantStartErr(t, r.createStream(context.Background()))
			}

			if tt.wantSendErr != nil {
				tt.wantSendErr(t, r.Send(tt.req), fmt.Sprintf("Send(%v)", tt.req))
			}

			mads.AssertExpectations(t)
		})
	}
}

func Test_reloadingStream_Recv(t *testing.T) {
	tests := []struct {
		name         string
		reloadCtx    func() (context.Context, context.CancelFunc)
		newStream    func(*testing.T, *mock.Mock) func()
		onReConnect  func(assert.TestingT, error)
		want         *v3discoverypb.DiscoveryResponse
		wantRecv     func(assert.TestingT, interface{}, interface{}, error, ...interface{}) bool
		wantStartErr assert.ErrorAssertionFunc
	}{
		{
			name: "StreamCreateError",
			newStream: func(t *testing.T, m *mock.Mock) func() {
				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(nil, errors.New("can't create a stream"))
				return nil
			},
			wantStartErr: assert.Error,
		},
		{
			name: "RecvSuccess",
			newStream: func(t *testing.T, m *mock.Mock) func() {
				mads := newMockAdsStream(t)
				mads.On("Recv").Return(&v3discoverypb.DiscoveryResponse{
					TypeUrl: resource.EndpointType,
				}, nil)

				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(mads, nil)
				return func() { mads.AssertExpectations(t) }
			},
			wantStartErr: assert.NoError,
			wantRecv: func(t assert.TestingT, expected interface{}, actual interface{}, err error, args ...interface{}) bool {
				return assert.Equal(t, expected, actual, args...) && assert.NoError(t, err, args...)
			},
			want: &v3discoverypb.DiscoveryResponse{TypeUrl: resource.EndpointType},
		},
		{
			name: "RecvWithError",
			reloadCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Second)
			},
			onReConnect: func(t assert.TestingT, err error) {
				assert.EqualError(t, err, "stream broken")
			},
			newStream: func(t *testing.T, m *mock.Mock) func() {
				mads1 := newMockAdsStream(t)
				mads1.On("Recv").Return(nil, errors.New("stream broken")).Once()
				mads2 := newMockAdsStream(t)

				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(mads1, nil).Once()
				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(mads2, nil).Once()

				return func() { mads1.AssertExpectations(t); mads2.AssertExpectations(t) }
			},
			wantStartErr: assert.NoError,
			wantRecv: func(t assert.TestingT, expected interface{}, actual interface{}, err error, args ...interface{}) bool {
				return assert.Equal(t, expected, actual, args...) && assert.Error(t, err, args...)
			},
			want: nil,
		},
		{
			name: "RecvWithReloadSuccessAfterFewFailures",
			reloadCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 10*time.Second)
			},
			newStream: func(t *testing.T, m *mock.Mock) func() {
				mads := newMockAdsStream(t)
				mads.On("Recv").Return(nil, errors.New("stream broken")).Once()

				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(mads, nil).Once()
				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(nil, errors.New("create stream error")).Times(3)
				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(mads, nil).Once()

				return func() { mads.AssertExpectations(t) }
			},
			wantStartErr: assert.NoError,
			wantRecv: func(t assert.TestingT, expected interface{}, actual interface{}, err error, args ...interface{}) bool {
				return assert.Equal(t, expected, actual, args...) && assert.Error(t, err, args...)
			},
			want: nil,
		},
		{
			name: "RecvErrorExceedReloadTime",
			reloadCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Second)
			},
			newStream: func(t *testing.T, m *mock.Mock) func() {
				mads := newMockAdsStream(t)
				mads.On("Recv").Return(nil, errors.New("stream broken"))

				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(mads, nil).Once()
				m.On("StreamAggregatedResources", mock.Anything, []grpc.CallOption{
					grpc.WaitForReady(true),
				}).Return(nil, errors.New("create stream error"))

				return func() { mads.AssertExpectations(t) }
			},
			wantStartErr: assert.NoError,
			wantRecv: func(t assert.TestingT, expected interface{}, actual interface{}, err error, args ...interface{}) bool {
				return assert.Equal(t, expected, actual, args...) && assert.Error(t, err, args...)
			},
		},
	}

	for _, s := range tests {
		tt := s
		t.Run(tt.name, func(t *testing.T) {
			mads := newMockADSClient(t)
			if tt.newStream != nil {
				ae := tt.newStream(t, &mads.Mock)
				if ae != nil {
					defer ae()
				}
			}

			r := &reloadingStream{
				ns: mads,
				onReConnect: func(err error) {
					if tt.onReConnect != nil {
						tt.onReConnect(t, err)
					}
				},
				strategy:    &backoff.DefaultExponential,
				reloadCh:    make(chan error, 1),
				connTimeout: 100 * time.Millisecond,
				log:         &log.NoOpLogger{},
			}

			if tt.wantStartErr != nil {
				tt.wantStartErr(t, r.createStream(context.Background()))
			}

			if tt.reloadCtx != nil {
				ctx, cancel := tt.reloadCtx()
				defer cancel()
				go r.startReloader(ctx)
			}

			if tt.wantRecv != nil {
				resp, err := r.Recv()
				tt.wantRecv(t, tt.want, resp, err, "Recv()")
			}

			mads.AssertExpectations(t)
		})
	}
}

// Mocks

func newMockADSClient(t *testing.T) *mockADSClient {
	m := &mockADSClient{}
	m.Test(t)
	return m
}

type mockADSClient struct {
	mock.Mock
}

func (m *mockADSClient) StreamAggregatedResources(ctx context.Context, opts ...grpc.CallOption) (
	v3discoverypb.AggregatedDiscoveryService_StreamAggregatedResourcesClient, error) {
	args := m.Called(ctx, opts)
	if ads := args.Get(0); ads != nil {
		return ads.(v3discoverypb.AggregatedDiscoveryService_StreamAggregatedResourcesClient), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockADSClient) DeltaAggregatedResources(ctx context.Context, opts ...grpc.CallOption) (
	v3discoverypb.AggregatedDiscoveryService_DeltaAggregatedResourcesClient, error) {
	args := m.Called(ctx, opts)
	if ads := args.Get(0); ads != nil {
		return ads.(v3discoverypb.AggregatedDiscoveryService_DeltaAggregatedResourcesClient), args.Error(1)
	}
	return nil, args.Error(1)
}

func newMockAdsStream(t *testing.T) *mockAdsStream {
	m := &mockAdsStream{}
	m.Test(t)
	return m
}

type mockAdsStream struct {
	mock.Mock
}

func (m *mockAdsStream) Send(req *v3discoverypb.DiscoveryRequest) error {
	return m.Called(req).Error(0)
}

func (m *mockAdsStream) Recv() (*v3discoverypb.DiscoveryResponse, error) {
	args := m.Called()
	if dr := args.Get(0); dr != nil {
		return dr.(*v3discoverypb.DiscoveryResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAdsStream) Header() (metadata.MD, error) {
	args := m.Called()
	return args.Get(0).(metadata.MD), args.Error(1)
}

func (m *mockAdsStream) Trailer() metadata.MD {
	return m.Called().Get(0).(metadata.MD)
}

func (m *mockAdsStream) CloseSend() error {
	return m.Called().Error(0)
}

func (m *mockAdsStream) Context() context.Context {
	return m.Called().Get(0).(context.Context)
}

func (m *mockAdsStream) SendMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}

func (m *mockAdsStream) RecvMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}
