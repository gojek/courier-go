package courier

import (
	"context"
	"crypto/tls"
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

	"github.com/gojekfarm/xtools/generic"
)

var defOpts []ClientOption
var testBrokerAddress TCPAddress

func init() {
	brokerAddress := os.Getenv("BROKER_ADDRESS") // host:port format
	if len(brokerAddress) == 0 {
		brokerAddress = "localhost:1883"
	}

	list := strings.Split(brokerAddress, ":")
	p, _ := strconv.Atoi(list[1])

	testBrokerAddress = TCPAddress{Host: list[0], Port: uint16(p)}
	defOpts = append(defOpts, WithAddress(list[0], uint16(p)), WithClientID("clientID"))
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
			opts: []ClientOption{WithAddress("127.0.0.1", 9999), WithOnReconnect(func(_ PubSub) {
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

			ctx, cancel := context.WithCancel(context.Background())
			errCh := make(chan error, 1)

			go func() { errCh <- c.Run(ctx) }()

			if t.checkConnect {
				s.Eventually(func() bool {
					return c.IsConnected()
				}, 10*time.Second, 250*time.Millisecond)
			}

			cancel()

			if err := <-errCh; t.wantErr != nil {
				s.Equal(t.wantErr, err)
			} else {
				s.NoError(err)
			}

			mr.AssertExpectations(s.T())
		})
	}
	newClientFunc.Store(mqtt.NewClient)

	s.Run("WithUninitializedClient", func() {
		c := &Client{
			options: &clientOptions{brokerAddress: "localhost:1883"},
		}
		s.True(errors.Is(c.Run(context.Background()), ErrClientNotInitialized))
	})
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

func TestNewClientWithCredentialFetcher(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mcf := newMockCredentialFetcher(t)
		mcf.On("Credentials", mock.Anything).Return(&Credential{
			Username: "username",
			Password: "password",
		}, nil)

		newClientFunc.Store(func(opts *mqtt.ClientOptions) mqtt.Client {
			assert.Equal(t, "username", opts.Username)
			assert.Equal(t, "password", opts.Password)
			return mqtt.NewClient(opts)
		})
		defer func() {
			newClientFunc.Store(mqtt.NewClient)
		}()

		c, err := NewClient(append(defOpts, WithCredentialFetcher(mcf))...)

		assert.NoError(t, c.Start())
		mcf.AssertExpectations(t)

		assert.Eventually(t, func() bool {
			return c.IsConnected()
		}, 10*time.Second, 250*time.Millisecond)

		c.Stop()

		assert.NoError(t, err)
		mcf.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mcf := newMockCredentialFetcher(t)
		ml := newMockLogger(t)

		credErr := errors.New("error")
		mcf.On("Credentials", mock.Anything).Return(nil, credErr)
		ml.On("Error", mock.Anything, credErr, map[string]any{"message": "failed to fetch credentials"})
		ml.On("Debug", mock.Anything, mock.Anything, mock.Anything)

		c, err := NewClient(append(defOpts, WithCredentialFetcher(mcf), WithLogger(ml))...)
		assert.NoError(t, err)

		assert.NoError(t, c.Start())

		assert.Eventually(t, func() bool {
			return ml.AssertExpectations(t)
		}, 10*time.Second, 250*time.Millisecond)

		commsErr := errors.New("[client]   Connect comms goroutine - error triggered EOF\n")
		ml.On("Error", mock.Anything, commsErr, nil).Maybe()
		c.Stop()

		mcf.AssertExpectations(t)

		// wait for mocked logger calling goroutines to finish
		time.Sleep(1 * time.Second)
		// restore paho logger back to default logger
		mqtt.CRITICAL = mqtt.NOOPLogger{}
		mqtt.ERROR = mqtt.NOOPLogger{}
		mqtt.WARN = mqtt.NOOPLogger{}
		mqtt.DEBUG = mqtt.NOOPLogger{}
	})
}

func TestNewClientWithExponentialStartOptions(t *testing.T) {
	c, err := NewClient(append(defOpts, WithExponentialStartOptions(WithMaxInterval(10*time.Second)))...)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() { errCh <- c.Run(ctx) }()

	assert.Eventually(t, func() bool { return c.IsConnected() }, 10*time.Second, 250*time.Millisecond)

	cancel()

	assert.NoError(t, <-errCh)
}

func TestNewClient(t *testing.T) {
	cc, err := NewClient()
	assert.EqualError(t, err, "at least WithAddress or WithResolver ClientOption should be used")
	assert.Nil(t, cc)

	cc, err = NewClient(WithAddress("localhost", 1883))
	assert.NoError(t, err)
	assert.NotNil(t, cc.mqttClient)
}

func TestNewClient_WithOptions(t *testing.T) {
	c, err := NewClient(defOpts...)
	assert.NoError(t, err)
	assert.NotNil(t, c)
}

func Test_reconnectHandler(t *testing.T) {
	o := defaultClientOptions()
	ml := newMockLogger(t)
	o.logger = ml
	o.onReconnectHandler = func(_ PubSub) {
		t.Logf("reconnectHandler called")
	}
	c := &Client{
		options: o,
		mqttClient: mqtt.NewClient(&mqtt.ClientOptions{
			ClientID: "clientID",
		}),
	}

	ml.On("Info", mock.Anything, "reconnecting", map[string]any{"client_id": "clientID"}).Return()

	f := reconnectHandler(c, c.options)
	f(c.mqttClient, &mqtt.ClientOptions{ClientID: "clientID"})
}

func Test_connectionLostHandler(t *testing.T) {
	o := defaultClientOptions()
	ml := newMockLogger(t)
	o.logger = ml
	o.onConnectionLostHandler = func(err error) {
		t.Logf("onConnectionLostHandler called")
	}
	c := &Client{
		options:    o,
		mqttClient: mqtt.NewClient(&mqtt.ClientOptions{ClientID: "clientID"}),
	}

	ml.On("Error", mock.Anything, mock.MatchedBy(func(err error) bool {
		return err.Error() == "disconnected"
	}), map[string]any{
		"message":   "connection lost",
		"client_id": "clientID",
	}).Return()

	f := connectionLostHandler(c, c.options)
	f(c.mqttClient, errors.New("disconnected"))
	ml.AssertExpectations(t)
}

func Test_onConnectHandler(t *testing.T) {
	o := defaultClientOptions()
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

func readOnlyChannel(ch chan struct{}) <-chan struct{} {
	return ch
}

func Test_formatAddressWithProtocol(t *testing.T) {
	tests := []struct {
		name string
		opts *clientOptions
		want string
	}{
		{
			name: "TLSConfigNotPresent",
			opts: &clientOptions{brokerAddress: "localhost:1883"},
			want: "tcp://localhost:1883",
		},
		{
			name: "TLSConfigPresent",
			opts: &clientOptions{tlsConfig: &tls.Config{}, brokerAddress: "localhost:1883"},
			want: "tls://localhost:1883",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, formatAddressWithProtocol(tt.opts), "formatAddressWithProtocol(%v)", tt.opts)
		})
	}
}

func matchErr(want error) any {
	return mock.MatchedBy(func(err error) bool {
		return err.Error() == want.Error()
	})
}

func TestClient_removeStoredSubsCalled(t *testing.T) {
	cc1 := mqtt.NewClient(&mqtt.ClientOptions{})

	c := &Client{mqttClients: map[string]*internalState{
		"clientID-1": {
			subsCalled: generic.NewSet("topic"),
			client:     cc1,
		},
		"clientID-2": {
			subsCalled: generic.NewSet("topic"),
			client:     mqtt.NewClient(&mqtt.ClientOptions{}),
		},
	}}

	c.removeStoredSubsCalled(cc1)

	assert.Empty(t, c.mqttClients["clientID-1"].subsCalled)
	assert.NotEmpty(t, c.mqttClients["clientID-2"].subsCalled)
}
