package courier

import (
	"context"
	"errors"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/gojekfarm/xtools/generic"
	"github.com/gojekfarm/xtools/generic/slice"
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
		ctxFn          func() (context.Context, context.CancelFunc)
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
			name: "DefaultWaitTimeout",
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
			name: "ContextDeadline",
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Second)
			},
			pahoMock: func(m *mock.Mock) *mockToken {
				ch := make(<-chan struct{})
				t := &mockToken{}
				t.On("Done").Return(ch)

				m.On("Unsubscribe", topics).Return(t)

				return t
			},
			wantErr: true,
		},
		{
			name: "TokenError",
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 10*time.Second)
			},
			pahoMock: func(m *mock.Mock) *mockToken {
				ch := make(chan struct{})

				go func() {
					<-time.After(2 * time.Second)
					ch <- struct{}{}
				}()

				t := &mockToken{}
				t.On("Done").Return(readOnlyChannel(ch))
				t.On("Error").Return(errors.New("token timed out"))

				m.On("Unsubscribe", topics).Return(t)

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
			c, err := NewClient(defOpts...)
			s.NoError(err)

			if t.useMiddlewares != nil {
				c.UseUnsubscriberMiddleware(t.useMiddlewares...)
			}

			mc := &mockClient{}
			c.mqttClient = mc
			tk := t.pahoMock(&mc.Mock)

			ctx := context.Background()
			if t.ctxFn != nil {
				_ctx, cancel := t.ctxFn()
				ctx = _ctx
				defer cancel()
			}

			err = c.Unsubscribe(ctx, topics...)

			if !t.wantErr {
				s.NoError(err)
			} else {
				s.Error(err)
			}
			mc.AssertExpectations(s.T())
			tk.AssertExpectations(s.T())
		})
	}

	s.Run("UnsubscribeOnUninitializedClient", func() {
		c := &Client{options: defaultClientOptions()}
		c.unsubscriber = unsubscriberHandler(c)
		s.True(errors.Is(c.Unsubscribe(context.Background(), topics...), ErrClientNotInitialized))
	})
}

func (s *ClientUnsubscribeSuite) TestUnsubscribeMiddleware() {
	topics := []string{"topic1", "topic2"}
	c, err := NewClient(defOpts...)
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

func Test_removeSubsFromState(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		s := &internalState{subsCalled: generic.NewSet("topic1", "topic2")}
		eo := removeSubsFromState("topic1", "topic2")
		ef := func(cc mqtt.Client) error { return nil }
		execs := []func(mqtt.Client) error{ef, ef, ef, ef}

		err := slice.Reduce(slice.MapConcurrent(execs, func(f func(mqtt.Client) error) error { return eo(f, s) }), accumulateErrors)
		assert.NoError(t, err)

		assert.False(t, s.subsCalled.HasAny("topic1", "topic2"))
	})

	t.Run("Error", func(t *testing.T) {
		s := &internalState{subsCalled: generic.NewSet("topic1", "topic2")}
		eo := removeSubsFromState("topic1", "topic2")
		ef := func(cc mqtt.Client) error { return errors.New("error") }
		execs := []func(mqtt.Client) error{ef, ef, ef, ef}

		err := slice.Reduce(slice.MapConcurrent(execs, func(f func(mqtt.Client) error) error { return eo(f, s) }), accumulateErrors)
		assert.Error(t, err)

		assert.True(t, s.subsCalled.HasAll("topic1", "topic2"))
	})
}
