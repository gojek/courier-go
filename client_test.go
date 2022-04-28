package courier

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var defOpts []ClientOption

func init() {
	brokerAddress := os.Getenv("BROKER_ADDRESS") // host:port format
	if len(brokerAddress) == 0 {
		brokerAddress = "localhost:1883"
	}

	list := strings.Split(brokerAddress, ":")
	p, _ := strconv.Atoi(list[1])

	defOpts = append(defOpts, WithTCPAddress(list[0], uint16(p)), WithClientID("clientID"))
}

type ClientSuite struct {
	suite.Suite
}

func TestClientSuite(t *testing.T) {
	suite.Run(t, new(ClientSuite))
}

func (s *ClientSuite) TestStart() {
	errConnect := errors.New("err_connect")
	defOpts := append(defOpts, WithOnConnect(func(_ PubSub) {
		s.T().Logf("connected")
	}))

	tests := []struct {
		name          string
		opts          []ClientOption
		ctxFunc       func() (context.Context, context.CancelFunc)
		checkConnect  bool
		wantErr       error
		resolver      func(*mock.Mock)
		newClientFunc func(o *mqtt.ClientOptions) mqtt.Client
	}{
		{
			name:         "Success",
			opts:         defOpts,
			checkConnect: true,
			wantErr:      nil,
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.TODO(), time.Now().Add(10*time.Second))
			},
		},
		{
			name: "ConnectWaitTimeoutError",
			opts: append(defOpts, WithConnectTimeout(5*time.Second)),
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.TODO(), time.Now().Add(10*time.Second))
			},
			wantErr: ErrConnectTimeout,
			newClientFunc: func(_ *mqtt.ClientOptions) mqtt.Client {
				m := &mockClient{}
				t := &mockToken{}
				t.On("WaitTimeout", 5*time.Second).Return(false)
				m.On("Connect").Return(t)
				return m
			},
		},
		{
			name: "ConnectWaitTimeoutErrorWithResolver",
			opts: []ClientOption{
				WithClientID("clientID"),
				WithConnectTimeout(5 * time.Second),
			},
			resolver: func(m *mock.Mock) {
				ch := make(chan []TCPAddress, 1)
				m.On("UpdateChan").Return(ch)
				go func() {
					time.Sleep(6 * time.Second)
					ch <- []TCPAddress{{Host: "localhost", Port: 1883}}
				}()
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.TODO(), time.Now().Add(10*time.Second))
			},
			wantErr: ErrConnectTimeout,
		},
		{
			name: "ConnectError",
			opts: []ClientOption{WithTCPAddress("127.0.0.1", 9999), WithOnReconnect(func(_ PubSub) {
				s.T().Logf("reconnecting")
			})},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.TODO(), time.Now().Add(10*time.Second))
			},
			newClientFunc: func(_ *mqtt.ClientOptions) mqtt.Client {
				m := &mockClient{}
				t := &mockToken{}
				t.On("WaitTimeout", 15*time.Second).Return(true)
				t.On("Error").Return(errConnect)
				m.On("Connect").Return(t)
				return m
			},
			wantErr: errConnect,
		},
	}
	for _, t := range tests {
		s.Run(t.name, func() {
			if t.newClientFunc != nil {
				newClientFunc.Store(t.newClientFunc)
			} else {
				newClientFunc.Store(mqtt.NewClient)
			}

			mr := newMockResolver(s.T())
			if t.resolver != nil {
				t.resolver(&mr.Mock)
				t.opts = append(t.opts, WithResolver(mr))
			}

			c, err := NewClient(t.opts...)
			s.NoError(err)

			if err := c.Start(); t.wantErr != nil {
				s.Equal(t.wantErr, err)
			} else {
				s.NoError(err)
			}

			if t.checkConnect {
				s.Eventually(func() bool {
					return c.IsConnected()
				}, 10*time.Second, 250*time.Millisecond)
			}

			if t.wantErr == nil {
				c.Stop()
			}

			mr.AssertExpectations(s.T())
		})
	}
}

func TestNewClientWithResolverOption(t *testing.T) {
	mc := newMockClient(t)
	mt := newMockToken(t)
	mt.On("WaitTimeout", 15*time.Second).Return(true)
	mt.On("Error").Return(nil)
	mc.On("Connect").Return(mt)
	mc.On("IsConnectionOpen").After(2 * time.Second).Return(true)
	mc.On("Disconnect", uint(30*time.Second/time.Millisecond)).After(10 * time.Millisecond).Return()
	newClientFunc.Store(func(_ *mqtt.ClientOptions) mqtt.Client { return mc })
	defer func() {
		newClientFunc.Store(mqtt.NewClient)
	}()

	mr := newMockResolver(t)
	c, err := NewClient(WithResolver(mr))

	assert.NoError(t, err)
	mr.AssertExpectations(t)

	rCh := make(chan []TCPAddress, 1)
	dCh := make(chan struct{}, 1)
	go func() {
		rCh <- []TCPAddress{{Host: "localhost", Port: 1883}}
	}()

	mr.On("UpdateChan").Return(rCh)
	mr.On("Done").Return(dCh)
	assert.NoError(t, c.Start())

	assert.Eventually(t, func() bool {
		return c.IsConnected()
	}, 10*time.Second, 250*time.Millisecond)

	c.Stop()
	dCh <- struct{}{}

	mr.AssertExpectations(t)
}

func TestNewClient(t *testing.T) {
	cc, err := NewClient()
	assert.EqualError(t, err, "at least WithTCPAddress or WithResolver ClientOption should be used")
	assert.Nil(t, cc)

	cc, err = NewClient(WithTCPAddress("localhost", 1883))
	assert.NoError(t, err)
	assert.NotNil(t, cc.mqttClient)
}

func TestNewClient_WithOptions(t *testing.T) {
	c, err := NewClient(defOpts...)
	assert.NoError(t, err)
	assert.NotNil(t, c)
}

func Test_reconnectHandler(t *testing.T) {
	o := defaultOptions()
	o.onReconnectHandler = func(_ PubSub) {
		t.Logf("reconnectHandler called")
	}
	c := &Client{options: o}
	f := reconnectHandler(c, c.options)
	f(c.mqttClient, &mqtt.ClientOptions{})
}

func Test_connectionLostHandler(t *testing.T) {
	o := defaultOptions()
	o.onConnectionLostHandler = func(err error) {
		t.Logf("onConnectionLostHandler called")
	}
	c := &Client{options: o}
	f := connectionLostHandler(c.options)
	f(c.mqttClient, errors.New("disconnected"))
}

func Test_onConnectHandler(t *testing.T) {
	o := defaultOptions()
	o.onConnectHandler = func(_ PubSub) {
		t.Logf("onConnectHandler called")
	}
	c := &Client{options: o}
	f := onConnectHandler(c, c.options)
	f(c.mqttClient)
}

// mocks
func newMockClient(t *testing.T) *mockClient {
	m := &mockClient{}
	m.Test(t)
	return m
}

type mockClient struct {
	mock.Mock
}

func (m *mockClient) IsConnected() bool {
	return m.Called().Bool(0)
}

func (m *mockClient) IsConnectionOpen() bool {
	return m.Called().Bool(0)
}

func (m *mockClient) Connect() mqtt.Token {
	return m.Called().Get(0).(mqtt.Token)
}

func (m *mockClient) Disconnect(quiesce uint) {
	m.Called(quiesce)
}

func (m *mockClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	return m.Called(topic, qos, retained, payload).Get(0).(mqtt.Token)
}

func (m *mockClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	return m.Called(topic, qos, callback).Get(0).(mqtt.Token)
}

func (m *mockClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	return m.Called(filters, callback).Get(0).(mqtt.Token)
}

func (m *mockClient) Unsubscribe(topics ...string) mqtt.Token {
	return m.Called(topics).Get(0).(mqtt.Token)
}

func (m *mockClient) AddRoute(topic string, callback mqtt.MessageHandler) {
	m.Called(topic, callback)
}

func (m *mockClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.ClientOptionsReader{}
}

func newMockToken(t *testing.T) *mockToken {
	m := &mockToken{}
	m.Test(t)
	return m
}

type mockToken struct {
	mock.Mock
}

func (m *mockToken) Wait() bool {
	return m.Called().Bool(0)
}

func (m *mockToken) WaitTimeout(duration time.Duration) bool {
	return m.Called(duration).Bool(0)
}

func (m *mockToken) Done() <-chan struct{} {
	return m.Called().Get(0).(<-chan struct{})
}

func (m *mockToken) Error() error {
	return m.Called().Error(0)
}

func newMockResolver(t *testing.T) *mockResolver {
	m := &mockResolver{}
	m.Test(t)
	return m
}

type mockResolver struct {
	mock.Mock
}

func (m *mockResolver) UpdateChan() <-chan []TCPAddress {
	if ch := m.Called().Get(0); ch != nil {
		return ch.(chan []TCPAddress)
	}
	return nil
}

func (m *mockResolver) Done() <-chan struct{} {
	if ch := m.Called().Get(0); ch != nil {
		return ch.(chan struct{})
	}
	return nil
}
