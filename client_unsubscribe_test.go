package courier

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"***REMOVED***/metrics"
)

type ClientUnsubscribeSuite struct {
	suite.Suite
}

func TestClientUnsubscribeSuite(t *testing.T) {
	suite.Run(t, new(ClientUnsubscribeSuite))
}

func (s *ClientUnsubscribeSuite) TestUnsubscribe() {
	topics := []string{"topic1", "topic2"}
	testcases := []struct {
		name           string
		pahoMock       func(*mock.Mock) *mockToken
		wantErr        bool
		useMiddlewares []UnsubscriberMiddlewareFunc
	}{
		{
			name: "Success",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(nil)
				m.On("Unsubscribe", topics).
					Return(t)
				return t
			},
		},
		{
			name: "AssertingUnsubscriberMiddleware",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(nil)
				m.On("Unsubscribe", topics).
					Return(t)
				return t
			},
			useMiddlewares: []UnsubscriberMiddlewareFunc{
				func(unsubsriber Unsubscriber) Unsubscriber {
					return UnsubscriberFunc(func(ctx context.Context, scopedTopics ...string) error {
						s.Require().Len(scopedTopics, 2)
						scopedTopics = append(scopedTopics, "another-topic")
						return unsubsriber.Unsubscribe(ctx, scopedTopics...)
					})
				},
				func(unsubsriber Unsubscriber) Unsubscriber {
					return UnsubscriberFunc(func(ctx context.Context, scopedTopics ...string) error {
						s.Require().Len(scopedTopics, 3)
						return unsubsriber.Unsubscribe(ctx, topics...)
					})
				},
			},
		},
		{
			name: "WaitTimeout",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(false)
				m.On("Unsubscribe", topics).
					Return(t)
				return t
			},
			wantErr: true,
		},
		{
			name: "Error",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(errors.New("error"))
				m.On("Unsubscribe", topics).
					Return(t)
				return t
			},
			wantErr: true,
		},
	}
	for _, t := range testcases {
		s.Run(t.name, func() {
			c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
			s.NoError(err)

			if t.useMiddlewares != nil {
				c.UseUnsubscriberMiddleware(t.useMiddlewares...)
			}

			mc := &mockClient{}
			c.mqttClient = mc
			tk := t.pahoMock(&mc.Mock)

			err = c.Unsubscribe(context.Background(), topics...)

			if !t.wantErr {
				s.NoError(err)
			} else {
				s.Error(err)
			}
			mc.AssertExpectations(s.T())
			tk.AssertExpectations(s.T())
		})
	}
}

func (s *ClientUnsubscribeSuite) TestUnsubscribeMiddleware() {
	topics := []string{"topic1", "topic2"}
	c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
	s.NoError(err)

	mc := &mockClient{}
	mc.Test(s.T())
	c.mqttClient = mc

	t := &mockToken{}
	t.On("WaitTimeout", mock.Anything).Return(true)
	t.On("Error").Return(nil)
	mc.On("Unsubscribe", topics).Return(t)

	tm := &testUnsubscribeMiddleware{}

	c.UseUnsubscriberMiddleware(tm.Middleware)
	s.Require().Len(c.usMiddlewares, 1)
	s.Equal(0, tm.timesCalled)

	s.NoError(c.Unsubscribe(context.Background(), topics...))
	s.Equal(1, tm.timesCalled)

	c.UseUnsubscriberMiddleware(tm.Middleware)
	s.Require().Len(c.usMiddlewares, 2)
	s.Equal(1, tm.timesCalled)

	s.NoError(c.Unsubscribe(context.Background(), topics...))
	s.Equal(3, tm.timesCalled)
}

type testUnsubscribeMiddleware struct {
	timesCalled int
}

func (tm *testUnsubscribeMiddleware) Middleware(us Unsubscriber) Unsubscriber {
	return UnsubscriberFunc(func(ctx context.Context, topics ...string) error {
		tm.timesCalled++
		return us.Unsubscribe(ctx, topics...)
	})
}
