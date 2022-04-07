package client

import (
	"context"
	"errors"
	"fmt"
	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/mock"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/anypb"
	"reflect"
	"testing"
)

func Test_client_ParseResponse(t *testing.T) {
	tests := []struct {
		name      string
		message   proto.Message
		resources []*anypb.Any
		vsn       string
		nonce     string
		wantErr   bool
	}{
		{
			name: "parse_success",
			message: &v3discoverypb.DiscoveryResponse{
				VersionInfo: "1",
				Resources: []*anypb.Any{{
					TypeUrl: resource.EndpointType,
				}},
				TypeUrl: resource.EndpointType,
				Nonce:   "1",
			},
			resources: []*anypb.Any{{
				TypeUrl: resource.EndpointType,
			}},
			vsn:   "1",
			nonce: "1",
		},
		{
			name: "parse_failure",
			message: &v3discoverypb.DiscoveryResponse{
				VersionInfo: "1",
				Resources: []*anypb.Any{{
					TypeUrl: resource.ClusterType,
				}},
				TypeUrl: resource.ClusterType,
				Nonce:   "1",
			},
			vsn:     "",
			nonce:   "",
			wantErr: true,
		},
		{
			name: "unrecognised_message_type",
			message: &anypb.Any{TypeUrl: "random_message"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{}
			resources, vsn, nonce, err := c.ParseResponse(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(resources, tt.resources) {
				t.Errorf("ParseResponse() got = %v, want %v", resources, tt.resources)
			}
			if vsn != tt.vsn {
				t.Errorf("ParseResponse() got = %v, want %v", vsn, tt.vsn)
			}
			if nonce != tt.nonce {
				t.Errorf("ParseResponse() got = %v, want %v", nonce, tt.nonce)
			}
		})
	}
}

func Test_client_Receive(t *testing.T) {
	discoveryResponse := &v3discoverypb.DiscoveryResponse{
		VersionInfo: "1",
		Resources: []*anypb.Any{{
			TypeUrl: resource.ClusterType,
		}},
		TypeUrl: resource.ClusterType,
		Nonce:   "1",
	}

	tests := []struct {
		name      string
		edsClient func() *mockEds
		msg       proto.Message
		wantErr   bool
	}{
		{
			name: "receive_success",
			edsClient: func() *mockEds {
				es := &mockEds{mock.Mock{}}
				es.On("Recv").Return(discoveryResponse, nil)
				return es
			},
			msg:     discoveryResponse,
			wantErr: false,
		},
		{
			name: "receive_error",
			edsClient: func() *mockEds {
				es := &mockEds{mock.Mock{}}
				es.On("Recv").Return(nil, errors.New("some error"))
				return es
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edsClient := tt.edsClient()
			c := &client{
				stream: edsClient,
			}

			got, err := c.Receive()
			if (err != nil) != tt.wantErr {
				t.Errorf("Receive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.msg) {
				t.Errorf("Receive() got = %v, want %v", got, tt.msg)
			}

			edsClient.AssertExpectations(t)
		})
	}
}

func Test_client_RestartEDSClient(t *testing.T) {

	tests := []struct {
		name    string
		conn    func() *mockConnection
		wantErr bool
	}{
		{
			name: "success",
			conn: func() *mockConnection {
				m := &mockConnection{mock.Mock{}}
				m.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.endpoint.v3.EndpointDiscoveryService/StreamEndpoints",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(&mockEds{}, nil)
				return m
			},
		},
		{
			name: "failure",
			conn: func() *mockConnection {
				m := &mockConnection{mock.Mock{}}
				m.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.endpoint.v3.EndpointDiscoveryService/StreamEndpoints",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(nil, errors.New("some error"))
				return m
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldEdsClient := &mockEds{mock.Mock{}}
			c := &client{
				stream: oldEdsClient,
			}

			if err := c.RestartEDSClient(context.Background(), tt.conn()); (err != nil) != tt.wantErr {
				t.Errorf("RestartEDSClient() error = %v, wantErr %v", err, tt.wantErr)
				if c.stream != oldEdsClient {
					t.Errorf("RestartEDSClient() c.stream = %v, want %v", c.stream, oldEdsClient)
				}
			}
		})
	}
}

func Test_client_SendRequest(t *testing.T) {
	c := client{
		nodeProto: &v3corepb.Node{
			Id:      "123",
			Cluster: "cluster",
		},
	}

	tests := []struct {
		name          string
		resourceNames []string
		version       string
		nonce         string
		errMsg        string
		wantErr       bool
		err           error
	}{
		{
			name:          "success",
			resourceNames: []string{"cluster-1", "cluster-2"},
			version:       "1",
			nonce:         "1",
			errMsg:        "",
		},
		{
			name:          "success_with_error_msg",
			resourceNames: []string{"cluster-1", "cluster-2"},
			version:       "1",
			nonce:         "1",
			errMsg:        "some error",
		},
		{
			name:          "failure",
			resourceNames: []string{"cluster-1", "cluster-2"},
			version:       "1",
			nonce:         "1",
			wantErr:       true,
			err: fmt.Errorf("xds: stream.Send(%+v) failed: %v", &v3discoverypb.DiscoveryRequest{
				Node:          c.nodeProto,
				TypeUrl:       resource.EndpointType,
				ResourceNames: []string{"cluster-1", "cluster-2"},
				VersionInfo:   "1",
				ResponseNonce: "1",
			}, "some error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edsClient := &mockEds{mock.Mock{}}
			if tt.wantErr {
				edsClient.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
					expectedRequest := &v3discoverypb.DiscoveryRequest{
						Node:          c.nodeProto,
						TypeUrl:       resource.EndpointType,
						ResourceNames: tt.resourceNames,
						VersionInfo:   tt.version,
						ResponseNonce: tt.nonce,
					}

					if tt.errMsg != "" {
						expectedRequest.ErrorDetail = &statuspb.Status{
							Code: int32(codes.InvalidArgument), Message: tt.errMsg,
						}
					}
					return proto.Equal(expectedRequest, req)
				})).Return(errors.New("some error"))
			}
			edsClient.On("Send", mock.MatchedBy(func(req *v3discoverypb.DiscoveryRequest) bool {
				expectedRequest := &v3discoverypb.DiscoveryRequest{
					Node:          c.nodeProto,
					TypeUrl:       resource.EndpointType,
					ResourceNames: tt.resourceNames,
					VersionInfo:   tt.version,
					ResponseNonce: tt.nonce,
				}

				if tt.errMsg != "" {
					expectedRequest.ErrorDetail = &statuspb.Status{
						Code: int32(codes.InvalidArgument), Message: tt.errMsg,
					}
				}
				return proto.Equal(expectedRequest, req)
			})).Return(nil)

			c.stream = edsClient

			err := c.SendRequest(tt.resourceNames, tt.version, tt.nonce, tt.errMsg)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			if (err != nil) && (errors.Is(err, tt.err)) {
				t.Errorf("SendRequest() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		cc      func(m *mockEds) grpc.ClientConnInterface
		want    Client
		wantErr bool
	}{
		{
			name: "success",
			cc: func(me *mockEds) grpc.ClientConnInterface {
				m := mockConnection{mock.Mock{}}

				m.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.endpoint.v3.EndpointDiscoveryService/StreamEndpoints",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(me, nil)

				return &m
			},
		},
		{
			name: "failure",
			cc: func(_ *mockEds) grpc.ClientConnInterface {
				m := mockConnection{mock.Mock{}}

				m.On("NewStream", mock.Anything, mock.Anything,
					"/envoy.service.endpoint.v3.EndpointDiscoveryService/StreamEndpoints",
					[]grpc.CallOption{grpc.FailFastCallOption{FailFast: false}}).Return(nil, errors.New("some error"))

				return &m
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eds := &mockEds{mock.Mock{}}
			cc := tt.cc(eds)
			wantClient := client{
				nodeProto: &v3corepb.Node{
					Id:      "123",
					Cluster: "cluster",
				},
				stream: eds,
			}

			got, err := NewClient(context.Background(),
				&v3corepb.Node{
					Id:      "123",
					Cluster: "cluster",
				},
				cc)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (err == nil) && !(proto.Equal(got.(*client).nodeProto, wantClient.nodeProto)) {
				t.Errorf("NewClient() got = %v, want %v", got.(*client).nodeProto, wantClient.nodeProto)
			}

			cc.(*mockConnection).AssertExpectations(t)
		})
	}
}
