package courier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/gojek/courier-go/internal/driver"
	mqtt "github.com/gojek/paho.mqtt.golang"
	"github.com/gojekfarm/xtools/generic/slice"
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
		ctxFn          func() (context.Context, context.CancelFunc)
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
			name: "DefaultWaitTimeout",
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
			name: "ContextDeadline",
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Second)
			},
			pahoMock: func(m *mock.Mock, p interface{}) *mockToken {
				ch := make(<-chan struct{})
				t := &mockToken{}
				t.On("Done").Return(ch)

				buf := bytes.Buffer{}
				_ = DefaultEncoderFunc(context.TODO(), &buf).Encode(p)
				m.On("Publish", "topic", byte(QOSOne), false, buf.Bytes()).Return(t)

				return t
			},
			payload: "payload",
			wantErr: true,
		},
		{
			name: "TokenError",
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 10*time.Second)
			},
			pahoMock: func(m *mock.Mock, p interface{}) *mockToken {
				ch := make(chan struct{})

				go func() {
					<-time.After(2 * time.Second)
					ch <- struct{}{}
				}()

				t := &mockToken{}
				t.On("Done").Return(readOnlyChannel(ch))
				t.On("Error").Return(errors.New("token timed out"))

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

			mc := newMockClient(s.T())
			c.mqttClient = mc
			tk := t.pahoMock(&mc.Mock, t.payload)

			ctx := context.Background()
			if t.ctxFn != nil {
				_ctx, cancel := t.ctxFn()
				ctx = _ctx
				defer cancel()
			}

			err = c.Publish(ctx, "topic", t.payload, QOSOne)

			if !t.wantErr {
				s.NoError(err)
			} else {
				s.Error(err)
			}
			mc.AssertExpectations(s.T())
			tk.AssertExpectations(s.T())
		})
	}

	s.Run("PublishOnUninitializedClient", func() {
		c := &Client{options: defaultClientOptions()}
		c.publisher = publishHandler(c)
		s.True(errors.Is(c.Publish(context.Background(), "topic", "data"), ErrClientNotInitialized))
	})
}

func (s *ClientPublishSuite) TestPublishWithMultiConnectionMode() {
	type testPayload struct {
		ID int `json:"id"`
	}

	r := newMockResolver(s.T())
	ch := make(chan []TCPAddress)
	doneCh := make(chan struct{})

	r.On("UpdateChan").Return(ch)
	r.On("Done").Return(doneCh)

	go func() { ch <- []TCPAddress{testBrokerAddress, testBrokerAddress} }()

	// 7 distinct message to publish
	messages := []testPayload{
		{ID: 1},
		{ID: 2},
		{ID: 3},
		{ID: 4},
		{ID: 5},
		{ID: 6},
		{ID: 7},
	}

	c, err := NewClient(append(defOpts, WithResolver(r), UseMultiConnectionMode)...)
	s.NoError(err)

	var mcks []any
	idCh := make(chan int, 7)

	clients := map[string]mqtt.Client{}

	for i := 0; i < 3; i++ {
		mc := newMockClient(s.T())
		mt := newMockToken(s.T())

		mt.On("WaitTimeout", 10*time.Second).Return(true)
		mt.On("Error").Return(nil)

		// round-robin messages on each client
		ii := i

		for j := ii; j < len(messages); j += 3 {
			mc.On("Publish", "topic", byte(QOSZero), false, mock.Anything).
				Return(mt).
				Run(func(args mock.Arguments) {
					tp := testPayload{}
					s.NoError(json.NewDecoder(bytes.NewReader(args.Get(3).([]byte))).Decode(&tp))
					idCh <- tp.ID
				})
		}

		tba := testBrokerAddress
		tba.Port = uint16(1883 + i)

		clients[fmt.Sprintf("%s-%d", tba.String(), ii)] = mc

		mcks = append(mcks, mc, mt)
	}

	s.Len(clients, 3)

	c.reloadClients(clients)

	invokes := slice.MapConcurrentWithContext(context.Background(), messages,
		func(ctx context.Context, tp testPayload) error {
			return c.Publish(ctx, "topic", &tp)
		},
	)

	var actualIDs []int
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(idCh)

		for i := range idCh {
			actualIDs = append(actualIDs, i)
		}
	}()
	wg.Wait()

	s.ElementsMatch([]int{1, 4, 7, 2, 5, 3, 6}, actualIDs)

	require.Len(s.T(), invokes, 7)
	s.NoError(slice.Reduce(invokes, accumulateErrors))

	require.Len(s.T(), mcks, 6)
	mock.AssertExpectationsForObjects(s.T(), mcks...)
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

// v5Client is a PropsPublisher so the publish path under MQTT v5 can be asserted without a broker.
type v5Client struct {
	mqtt.Client
	topic string
	props *driver.PublishProps
}

func (v *v5Client) OptionsReader() mqtt.ClientOptionsReader { return mqtt.ClientOptionsReader{} }

func (v *v5Client) PublishWithProps(
	topic string, _ byte, _ bool, _ interface{}, props *driver.PublishProps,
) mqtt.Token {
	v.topic, v.props = topic, props

	tk := &mockToken{}
	tk.On("WaitTimeout", 10*time.Second).Return(true)
	tk.On("Error").Return(nil)

	return tk
}

func TestPublish_UserPropertiesFromContext(t *testing.T) {
	fake := &v5Client{}

	newClientFunc.Store(func(*mqtt.ClientOptions) mqtt.Client { return fake })
	defer newClientFunc.Store(mqtt.NewClient)

	c, err := NewClient(WithAddress("127.0.0.1", 1883))
	require.NoError(t, err)

	ctx := WithUserProperties(context.Background(), map[string]string{"k": "v"})
	require.NoError(t, c.Publish(ctx, "topic", "payload"))

	require.Equal(t, "topic", fake.topic)
	require.Equal(t, map[string]string{"k": "v"}, fake.props.UserProperties)

	require.NoError(t, c.Publish(context.Background(), "topic", "payload"))
	require.Empty(t, fake.props.UserProperties)

	ctx = WithContentType(ctx, "application/json")
	ctx = WithResponseTopic(ctx, "reply/me")
	ctx = WithCorrelationData(ctx, []byte("id-1"))
	require.NoError(t, c.Publish(ctx, "topic", "payload"))
	require.Equal(t, &driver.PublishProps{
		UserProperties:  map[string]string{"k": "v"},
		ContentType:     "application/json",
		ResponseTopic:   "reply/me",
		CorrelationData: []byte("id-1"),
	}, fake.props)
}
