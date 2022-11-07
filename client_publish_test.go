package courier

import (
	"bytes"
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ClientPublishSuite struct {
	suite.Suite
}

func TestClientPublishSuite(t *testing.T) {
	suite.Run(t, new(ClientPublishSuite))
}

func (s *ClientPublishSuite) TestPublish() {
	tests := []struct {
		name           string
		payload        interface{}
		pahoMock       func(*mock.Mock, interface{}) *mockToken
		wantErr        bool
		useMiddlewares []PublisherMiddlewareFunc
	}{
		{
			name: "Success",
			pahoMock: func(m *mock.Mock, p interface{}) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(nil)
				buf := bytes.Buffer{}
				_ = DefaultEncoderFunc(context.TODO(), &buf).Encode(p)
				m.On("Publish", "topic", byte(QOSOne), false, buf.Bytes()).Return(t)
				return t
			},
			payload: "payload",
		},
		{
			name: "AssertingPublisherMiddleware",
			pahoMock: func(m *mock.Mock, p interface{}) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(nil)
				buf := bytes.Buffer{}
				_ = DefaultEncoderFunc(context.TODO(), &buf).Encode(p)
				m.On("Publish", "topic-new", byte(QOSOne), false, buf.Bytes()).Return(t)
				return t
			},
			payload: "payload",
			useMiddlewares: []PublisherMiddlewareFunc{
				func(publisher Publisher) Publisher {
					return PublisherFunc(func(ctx context.Context, topic string, message interface{}, opts ...Option) error {
						s.Equal("topic", topic)
						o := composeOptions(opts)
						s.Equal(QOSOne, QOSLevel(o.qos))
						s.False(o.retained)
						s.IsType("", message)
						return publisher.Publish(ctx, "topic-new", message, opts...)
					})
				},
				func(publisher Publisher) Publisher {
					return PublisherFunc(func(ctx context.Context, topic string, message interface{}, opts ...Option) error {
						s.Equal("topic-new", topic)
						o := composeOptions(opts)
						s.Equal(QOSOne, QOSLevel(o.qos))
						s.False(o.retained)
						s.IsType("", message)
						return publisher.Publish(ctx, topic, message, opts...)
					})
				},
			},
		},
		{
			name: "WaitTimeout",
			pahoMock: func(m *mock.Mock, p interface{}) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(false)
				buf := bytes.Buffer{}
				_ = DefaultEncoderFunc(context.TODO(), &buf).Encode(p)
				m.On("Publish", "topic", byte(QOSOne), false, buf.Bytes()).Return(t)
				return t
			},
			payload: "payload",
			wantErr: true,
		},
		{
			name: "Error",
			pahoMock: func(m *mock.Mock, p interface{}) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(errors.New("random_error"))
				buf := bytes.Buffer{}
				_ = DefaultEncoderFunc(context.TODO(), &buf).Encode(p)
				m.On("Publish", "topic", byte(QOSOne), false, buf.Bytes()).Return(t)
				return t
			},
			payload: "payload",
			wantErr: true,
		},
		{
			name:    "EncodeError",
			payload: math.Inf(1),
			pahoMock: func(m *mock.Mock, p interface{}) *mockToken {
				return &mockToken{}
			},
			wantErr: true,
		},
	}

	for _, t := range tests {
		s.Run(t.name, func() {
			c, err := NewClient(defOpts...)
			s.NoError(err)

			if t.useMiddlewares != nil {
				c.UsePublisherMiddleware(t.useMiddlewares...)
			}

			mc := &mockClient{}
			c.mqttClient = mc
			tk := t.pahoMock(&mc.Mock, t.payload)

			err = c.Publish(context.Background(), "topic", t.payload, QOSOne)

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

func (s *ClientPublishSuite) TestPublishMiddleware() {
	c, err := NewClient(defOpts...)
	s.NoError(err)

	mc := &mockClient{}
	mc.Test(s.T())
	c.mqttClient = mc

	t := &mockToken{}
	t.On("WaitTimeout", mock.Anything).Return(true)
	t.On("Error").Return(nil)
	mc.On("Publish", "topic", mock.Anything, false, mock.Anything).Return(t)

	tm := &testPublishMiddleware{}

	c.UsePublisherMiddleware(tm.Middleware)
	s.Require().Len(c.pMiddlewares, 1)
	s.Equal(0, tm.timesCalled)

	s.NoError(c.Publish(context.Background(), "topic", "data"))
	s.Equal(1, tm.timesCalled)

	c.UsePublisherMiddleware(tm.Middleware)
	s.Require().Len(c.pMiddlewares, 2)
	s.Equal(1, tm.timesCalled)

	s.NoError(c.Publish(context.Background(), "topic", "data"))
	s.Equal(3, tm.timesCalled)
}

type testPublishMiddleware struct {
	timesCalled int
}

func (tm *testPublishMiddleware) Middleware(p Publisher) Publisher {
	return PublisherFunc(func(ctx context.Context, topic string, message interface{}, opts ...Option) error {
		tm.timesCalled++
		return p.Publish(ctx, topic, message, opts...)
	})
}
