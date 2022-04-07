package client

import (
	"context"
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"reflect"
	"testing"
)

func Test_clientDecorator_ParseResponse(t *testing.T) {
	tests := []struct {
		name          string
		msg           proto.Message
		resources     []*anypb.Any
		vsn           string
		nonce         string
		err           error
		nilMiddleware bool
		wantErr       bool
	}{
		{
			name:      "middleware_successfully_called",
			msg:       &anypb.Any{},
			resources: []*anypb.Any{{}},
			vsn:       "1",
			nonce:     "1",
			wantErr:   false,
		},
		{
			name:      "eds_error",
			msg:       &anypb.Any{},
			resources: []*anypb.Any{{}},
			vsn:       "1",
			nonce:     "1",
			err:       errors.New("some_error"),
			wantErr:   true,
		},
		{
			name:          "no_middleware_func",
			msg:           &anypb.Any{},
			resources:     []*anypb.Any{{}},
			vsn:           "1",
			nonce:         "1",
			nilMiddleware: true,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockClient{mock.Mock{}}
			mc.On("ParseResponse", tt.msg).Return(tt.resources, tt.vsn, tt.nonce, tt.err)

			middlewareClient := &mockDecorator{}
			parseMiddleware := middlewareClient.ParseMiddleware

			if tt.nilMiddleware {
				parseMiddleware = nil
			}
			d, err := GetClientDecorator(middlewareOpts{
				client:                  mc,
				parseResponseMiddleware: parseMiddleware,
			})

			resources, vsn, nonce, err := d.ParseResponse(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(resources, tt.resources) {
				t.Errorf("ParseResponse() got = %v, want %v", resources, tt.resources)
			}
			if vsn != tt.vsn {
				t.Errorf("ParseResponse() got1 = %v, want %v", resources, tt.vsn)
			}
			if nonce != tt.nonce {
				t.Errorf("ParseResponse() got2 = %v, want %v", nonce, tt.nonce)
			}

			mc.AssertExpectations(t)

			if !middlewareClient.parseCalled && !tt.nilMiddleware {
				t.Errorf("Middleware func not called")
				return
			}

			if !tt.nilMiddleware && !(middlewareClient.parseArgs[0].(proto.Message) == tt.msg) {
				t.Errorf("ParseResponse() parseArgs = %v, want %v", middlewareClient.parseArgs, tt.msg)
			}
		})
	}
}

func Test_clientDecorator_Receive(t *testing.T) {
	tests := []struct {
		name          string
		msg           proto.Message
		err           error
		nilMiddleware bool
		wantErr       bool
	}{
		{
			name:    "middleware_successfully_called",
			msg:     &anypb.Any{},
			wantErr: false,
		},
		{
			name:    "receive_error",
			msg:     &anypb.Any{},
			err:     errors.New("some_error"),
			wantErr: true,
		},
		{
			name:          "no_middleware_func",
			msg:           &anypb.Any{},
			nilMiddleware: true,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockClient{mock.Mock{}}
			mc.On("Receive").Return(tt.msg, tt.err)

			middlewareClient := &mockDecorator{}
			recvMiddleware := middlewareClient.ReceiveMiddleware

			if tt.nilMiddleware {
				recvMiddleware = nil
			}
			d, err := GetClientDecorator(middlewareOpts{
				client:            mc,
				receiveMiddleware: recvMiddleware,
			})
			msg, err := d.Receive()
			if (err != nil) != tt.wantErr {
				t.Errorf("Receive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(msg, tt.msg) {
				t.Errorf("Receive() got = %v, want %v", msg, tt.msg)
			}

			mc.AssertExpectations(t)

			if !middlewareClient.receiveCalled && !tt.nilMiddleware {
				t.Errorf("Middleware func not called")
				return
			}
		})
	}
}

func Test_clientDecorator_RestartEDSClient(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		nilMiddleware bool
		wantErr       bool
	}{
		{
			name:    "middleware_successfully_called",
			wantErr: false,
		},
		{
			name:    "restart_error",
			err:     errors.New("some_error"),
			wantErr: true,
		},
		{
			name:          "no_middleware_func",
			nilMiddleware: true,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockClient{mock.Mock{}}
			mc.On("RestartEDSClient", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*client.mockConnection")).Return(tt.err)

			middlewareClient := &mockDecorator{}
			restartMiddleware := middlewareClient.RestartFuncMiddleware

			if tt.nilMiddleware {
				restartMiddleware = nil
			}
			d, err := GetClientDecorator(middlewareOpts{
				client:            mc,
				restartMiddleware: restartMiddleware,
			})

			if err = d.RestartEDSClient(context.Background(), &mockConnection{mock.Mock{}}); (err != nil) != tt.wantErr {
				t.Errorf("RestartEDSClient() error = %v, wantErr %v", err, tt.wantErr)
			}

			mc.AssertExpectations(t)

			if !middlewareClient.restartCalled && !tt.nilMiddleware {
				t.Errorf("Middleware func not called")
				return
			}
		})
	}
}

func Test_clientDecorator_SendRequest(t *testing.T) {
	tests := []struct {
		name                   string
		resourceNames          []string
		version, nonce, errMsg string
		err                    error
		nilMiddleware          bool
		wantErr                bool
	}{
		{
			name:          "middleware_successfully_called",
			resourceNames: []string{"cluster"},
			version:       "1",
			nonce:         "1",
			errMsg:        "some_error",
			wantErr:       false,
		},
		{
			name:          "send_error",
			resourceNames: []string{"cluster"},
			version:       "1",
			nonce:         "1",
			errMsg:        "some_error",
			err:           errors.New("some_error"),
			wantErr:       true,
		},
		{
			name:          "no_middleware_func",
			resourceNames: []string{"cluster"},
			version:       "1",
			nonce:         "1",
			errMsg:        "some_error",
			nilMiddleware: true,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockClient{mock.Mock{}}
			mc.On("SendRequest", tt.resourceNames, tt.version, tt.nonce, tt.errMsg).Return(tt.err)
			middlewareClient := &mockDecorator{}
			sendFunc := middlewareClient.SendRequestMiddleware

			if tt.nilMiddleware {
				sendFunc = nil
			}
			d, err := GetClientDecorator(middlewareOpts{
				client:         mc,
				sendMiddleware: sendFunc,
			})
			if err = d.SendRequest(tt.resourceNames, tt.version, tt.nonce, tt.errMsg); (err != nil) != tt.wantErr {
				t.Errorf("SendRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			mc.AssertExpectations(t)

			if !middlewareClient.sendCalled && !tt.nilMiddleware {
				t.Errorf("Middleware func not called")
				return
			}

			if !tt.nilMiddleware &&
				!((middlewareClient.sendArgs[0].([]string)[0] == tt.resourceNames[0]) &&
					(middlewareClient.sendArgs[1] == tt.version) &&
					(middlewareClient.sendArgs[2] == tt.nonce) &&
					(middlewareClient.sendArgs[3] == tt.errMsg)) {
				t.Errorf("Middleware received wrong args")
			}
		})
	}
}

type mockDecorator struct {
	restartCalled bool
	restartArgs   []interface{}
	sendCalled    bool
	sendArgs      []interface{}
	receiveCalled bool
	receiveArgs   []interface{}
	parseCalled   bool
	parseArgs     []interface{}
}

func (m *mockDecorator) RestartFuncMiddleware(restartFunc restartFunc) restartFunc {
	return func(ctx context.Context, connInterface grpc.ClientConnInterface) error {
		m.restartCalled = true
		m.restartArgs = append(m.restartArgs, ctx, connInterface)
		err := restartFunc(ctx, connInterface)
		return err
	}
}

func (m *mockDecorator) SendRequestMiddleware(sendFunc sendFunc) sendFunc {
	return func(resourceNames []string, version string, nonce string, errMsg string) error {
		m.sendCalled = true
		m.sendArgs = append(m.restartArgs, resourceNames, version, nonce, errMsg)
		err := sendFunc(resourceNames, version, nonce, errMsg)
		return err
	}
}

func (m *mockDecorator) ReceiveMiddleware(receiveFunc2 receiveFunc) receiveFunc {
	return func() (proto.Message, error) {
		m.receiveCalled = true
		msg, err := receiveFunc2()
		return msg, err
	}
}

func (m *mockDecorator) ParseMiddleware(parseFunc parseFunc) parseFunc {
	return func(r proto.Message) ([]*anypb.Any, string, string, error) {
		m.parseCalled = true
		m.parseArgs = append(m.restartArgs, r)
		resources, vasn, nonce, err := parseFunc(r)
		return resources, vasn, nonce, err
	}
}

type mockClient struct {
	mock.Mock
}

func (m *mockClient) RestartEDSClient(ctx context.Context, cc grpc.ClientConnInterface) error {
	return m.Called(ctx, cc).Error(0)
}

func (m *mockClient) SendRequest(resourceNames []string, version, nonce, errMsg string) error {
	return m.Called(resourceNames, version, nonce, errMsg).Error(0)
}

func (m *mockClient) ParseResponse(r proto.Message) ([]*anypb.Any, string, string, error) {
	args := m.Called(r)
	return args.Get(0).([]*anypb.Any), args.Get(1).(string), args.Get(2).(string), args.Error(3)
}

func (m *mockClient) Receive() (proto.Message, error) {
	args := m.Called()
	return args.Get(0).(proto.Message), args.Error(1)
}
