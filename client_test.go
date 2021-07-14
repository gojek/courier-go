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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"***REMOVED***/metrics"
)

type ClientSuite struct {
	suite.Suite
}

func TestClientSuite(t *testing.T) {
	suite.Run(t, new(ClientSuite))
}

func (s *ClientSuite) TestStart() {
	errConnect := errors.New("err_connect")
	brokerAddress := os.Getenv("BROKER_ADDRESS") // host:port format

	defOpts := []Option{WithOnConnect(func(_ PubSub) {
		s.T().Logf("connected")
	}), WithCustomMetrics(metrics.NewPrometheus())}

	if brokerAddress != "" {
		list := strings.Split(brokerAddress, ":")
		p, _ := strconv.Atoi(list[1])
		defOpts = append(defOpts, WithTCPAddress(list[0], uint16(p)))
	}

	tests := []struct {
		name          string
		opts          []Option
		ctxFunc       func() (context.Context, context.CancelFunc)
		checkConnect  bool
		wantErr       error
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
			opts: []Option{WithConnectTimeout(5 * time.Second), WithCustomMetrics(metrics.NewPrometheus())},
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
			name: "ConnectError",
			opts: []Option{WithTCPAddress("127.0.0.1", 9999), WithOnReconnect(func(_ PubSub) {
				s.T().Logf("reconnecting")
			}), WithCustomMetrics(metrics.NewPrometheus())},
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
				newClientFunc = t.newClientFunc
			} else {
				newClientFunc = mqtt.NewClient
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
		})
	}
}

func TestNewClient(t *testing.T) {
	cc, err := NewClient()
	assert.NoError(t, err)
	assert.NotNil(t, cc.mqttClient)
	assert.NotNil(t, cc.options.metricsCollector)
}

func TestNewClient_prometheus_error(t *testing.T) {
	// register one already, second register should give error
	m := metrics.NewPrometheus()
	_ = m.AddToRegistry(prometheus.DefaultRegisterer)

	_, err := NewClient()
	assert.Error(t, err)
}

func TestNewClient_WithOptions(t *testing.T) {
	c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
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
