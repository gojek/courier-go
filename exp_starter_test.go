package courier

import (
	"context"
	"net"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/suite"

	"***REMOVED***/metrics"
)

type ExponentialStartStrategySuite struct {
	suite.Suite

	mockClient *mockClient
}

func TestExponentialStartStrategySuite(t *testing.T) {
	suite.Run(t, new(ExponentialStartStrategySuite))
}

func (s *ExponentialStartStrategySuite) SetupSuite() {
	newClientFunc = func(o *mqtt.ClientOptions) mqtt.Client {
		m := &mockClient{}
		m.Test(s.T())
		s.mockClient = m
		return m
	}
}

func (s *ExponentialStartStrategySuite) TearDownSuite() {
	newClientFunc = mqtt.NewClient
}

func (s *ExponentialStartStrategySuite) TestSuccessfulStartOnFirstTry() {
	c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
	s.NoError(err)

	tk := &mockToken{}
	tk.Test(s.T())
	tk.On("WaitTimeout", 15*time.Second).Return(true).
		After(150 * time.Millisecond).Once()
	tk.On("Error").Return(nil).Once()

	s.mockClient.On("Connect").Return(tk).Once()
	s.mockClient.On("Disconnect", uint(30*time.Second/time.Millisecond)).
		Return(true).After(time.Second).Once()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- ExponentialStartStrategy(c, WithMaxInterval(30*time.Second))(ctx)
	}()

	time.Sleep(time.Second)
	cancel()

	s.NoError(<-errCh)
	tk.AssertExpectations(s.T())
	s.mockClient.AssertExpectations(s.T())
}

func (s *ExponentialStartStrategySuite) TestReconnectAttemptOnFailure() {
	c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
	s.NoError(err)

	tk := &mockToken{}
	tk.Test(s.T())
	tk.On("WaitTimeout", 15*time.Second).Return(true).
		After(5 * time.Millisecond).Once()
	tk.On("Error").Return(&net.AddrError{Err: "connection refused", Addr: ":1883"}).Once()
	tk.On("WaitTimeout", 15*time.Second).Return(true).
		After(50 * time.Millisecond)
	tk.On("Error").Return(nil)

	s.mockClient.On("Connect").Return(tk)
	s.mockClient.On("Disconnect", uint(30*time.Second/time.Millisecond)).
		Return(true).After(time.Second).Once()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)

	go func() {
		errCh <- ExponentialStartStrategy(c, WithMaxInterval(2*time.Second), WithOnRetry(func(err error) {
			s.EqualError(err, "address :1883: connection refused")
		}))(ctx)
	}()

	time.Sleep(3 * time.Second)
	cancel()

	s.NoError(<-errCh)
	tk.AssertExpectations(s.T())
	s.mockClient.AssertExpectations(s.T())
}

func (s *ExponentialStartStrategySuite) TestReconnectAttemptOnFailureBeyondMaxTimeout() {
	c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
	s.NoError(err)

	tk := &mockToken{}
	tk.Test(s.T())
	tk.On("WaitTimeout", 15*time.Second).Return(true).
		After(5 * time.Millisecond)
	tk.On("Error").Return(&net.AddrError{Err: "connection refused", Addr: ":1883"}).Times(8)
	tk.On("Error").Return(nil)

	s.mockClient.On("Connect").Return(tk)
	s.mockClient.On("Disconnect", uint(30*time.Second/time.Millisecond)).
		Return(true).After(time.Second).Once()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)

	go func() {
		errCh <- ExponentialStartStrategy(c, WithMaxInterval(3*time.Second))(ctx)
	}()

	time.Sleep(10 * time.Second)
	cancel()

	s.NoError(<-errCh)
	tk.AssertExpectations(s.T())
	s.mockClient.AssertExpectations(s.T())
}
