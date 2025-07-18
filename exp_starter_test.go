package courier

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	mqtt "github.com/gojek/paho.mqtt.golang"
)

type ExponentialStartStrategySuite struct {
	suite.Suite

	mockClient *mockClient
}

func TestExponentialStartStrategySuite(t *testing.T) {
	suite.Run(t, new(ExponentialStartStrategySuite))
}

func (s *ExponentialStartStrategySuite) SetupSuite() {
	newClientFunc.Store(func(o *mqtt.ClientOptions) mqtt.Client {
		m := &mockClient{}
		m.Test(s.T())
		s.mockClient = m
		return m
	})
}

func (s *ExponentialStartStrategySuite) TearDownSuite() {
	newClientFunc.Store(mqtt.NewClient)
}

func (s *ExponentialStartStrategySuite) TestSuccessfulStartOnFirstTry() {
	c, err := NewClient(defOpts...)
	s.NoError(err)

	tk := &mockToken{}
	tk.Test(s.T())
	tk.On("WaitTimeout", 15*time.Second).Return(true).
		After(150 * time.Millisecond).Once()
	tk.On("Error").Return(nil).Once()

	s.mockClient.On("Connect").Return(tk).Once()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ExponentialStartStrategy(ctx, c)

	time.Sleep(time.Second)

	tk.AssertExpectations(s.T())
	s.mockClient.AssertExpectations(s.T())
}

func (s *ExponentialStartStrategySuite) TestReconnectAttemptOnFailure() {
	c, err := NewClient(defOpts...)
	s.NoError(err)

	tk := &mockToken{}
	tk.Test(s.T())

	tk.On("WaitTimeout", 15*time.Second).Return(true).
		After(50 * time.Millisecond)
	tk.On("Error").
		Return(&net.AddrError{Err: "connection refused", Addr: ":1883"}).Times(4)
	tk.On("Error").Return(nil)

	s.mockClient.On("Connect").Return(tk)

	ctx, cancel := context.WithCancel(context.Background())

	onRetryCalled := &atomic.Bool{}
	onRetryCalled.Store(false)

	ExponentialStartStrategy(ctx, c, WithMaxInterval(2*time.Second), WithOnRetry(func(err error) {
		onRetryCalled.Store(true)
		s.EqualError(err, "address :1883: connection refused")
	}))

	time.Sleep(3 * time.Second)
	cancel()

	time.Sleep(time.Second)

	s.True(onRetryCalled.Load())
	tk.AssertExpectations(s.T())
	s.mockClient.AssertExpectations(s.T())
}

func (s *ExponentialStartStrategySuite) TestReconnectAttemptStopOnCancel() {
	c, err := NewClient(defOpts...)
	s.NoError(err)

	tk := &mockToken{}
	tk.Test(s.T())

	tk.On("WaitTimeout", 15*time.Second).Return(true).
		After(50 * time.Millisecond)
	tk.On("Error").
		Return(&net.AddrError{Err: "connection refused", Addr: ":1883"})

	s.mockClient.On("Connect").Return(tk)

	ctx, cancel := context.WithCancel(context.Background())

	onRetryCalled := &atomic.Bool{}
	onRetryCalled.Store(false)

	ExponentialStartStrategy(ctx, c, WithMaxInterval(5*time.Second), WithOnRetry(func(err error) {
		onRetryCalled.Store(true)
		s.EqualError(err, "address :1883: connection refused")
	}))

	time.Sleep(1 * time.Second)
	cancel()

	time.Sleep(3 * time.Second)

	s.True(onRetryCalled.Load())
	tk.AssertExpectations(s.T())
	s.mockClient.AssertExpectations(s.T())
}

func (s *ExponentialStartStrategySuite) TestReconnectAttemptOnFailureBeyondMaxTimeout() {
	c, err := NewClient(defOpts...)
	s.NoError(err)

	tk := &mockToken{}
	tk.Test(s.T())
	tk.On("WaitTimeout", 15*time.Second).Return(true).
		After(5 * time.Millisecond)
	tk.On("Error").Return(&net.AddrError{Err: "connection refused", Addr: ":1883"}).Times(8)
	tk.On("Error").Return(nil)

	s.mockClient.On("Connect").Return(tk)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ExponentialStartStrategy(ctx, c, WithMaxInterval(3*time.Second))

	time.Sleep(10 * time.Second)
	cancel()

	tk.AssertExpectations(s.T())
	s.mockClient.AssertExpectations(s.T())
}

func Test_defaultStartOptions(t *testing.T) {
	so := &startOptions{maxInterval: 30 * time.Second}
	val1 := fmt.Sprintf("%v", so)
	val2 := fmt.Sprintf("%v", defaultStartOptions())
	assert.Equal(t, val2, val1)
}
