package courier

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"***REMOVED***/metrics"
)

type ClientSubscriberSuite struct {
	suite.Suite
}

func TestClientSubscriberSuite(t *testing.T) {
	suite.Run(t, new(ClientSubscriberSuite))
}

func (s *ClientSubscriberSuite) TestSubscribe() {
	callback := func(_ PubSub, _ Decoder) {}
	testcases := []struct {
		name     string
		pahoMock func(*mock.Mock) *mockToken
		wantErr  bool
	}{
		{
			name: "Success",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(nil)
				m.On("Subscribe", "topic", byte(QOSOne), mock.AnythingOfType("mqtt.MessageHandler")).
					Return(t)
				return t
			},
		},
		{
			name: "WaitTimeout",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(false)
				m.On("Subscribe", "topic", byte(QOSOne), mock.AnythingOfType("mqtt.MessageHandler")).
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
				m.On("Subscribe", "topic", byte(QOSOne), mock.AnythingOfType("mqtt.MessageHandler")).
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

			mc := &mockClient{}
			c.mqttClient = mc
			tk := t.pahoMock(&mc.Mock)

			err = c.Subscribe("topic", QOSOne, callback)

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

func (s *ClientSubscriberSuite) TestSubscribeMultiple() {
	callback := func(_ PubSub, _ Decoder) {}
	topics := map[string]QOSLevel{"topic": QOSOne}
	testcases := []struct {
		name     string
		pahoMock func(*mock.Mock) *mockToken
		wantErr  bool
	}{
		{
			name: "Success",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(nil)
				m.On("SubscribeMultiple", routeFilters(topics), mock.AnythingOfType("mqtt.MessageHandler")).
					Return(t)
				return t
			},
		},
		{
			name: "WaitTimeout",
			pahoMock: func(m *mock.Mock) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(false)
				m.On("SubscribeMultiple", routeFilters(topics), mock.AnythingOfType("mqtt.MessageHandler")).
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
				m.On("SubscribeMultiple", routeFilters(topics), mock.AnythingOfType("mqtt.MessageHandler")).
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

			mc := &mockClient{}
			c.mqttClient = mc
			tk := t.pahoMock(&mc.Mock)

			err = c.SubscribeMultiple(topics, callback)

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

func (s *ClientSubscriberSuite) TestUnsubscribe() {
	topics := []string{"topic1", "topic2"}
	testcases := []struct {
		name     string
		pahoMock func(*mock.Mock) *mockToken
		wantErr  bool
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

			mc := &mockClient{}
			c.mqttClient = mc
			tk := t.pahoMock(&mc.Mock)

			err = c.Unsubscribe(topics...)

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

func (s *ClientSubscriberSuite) Test_callbackWrapper() {
	c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
	s.NoError(err)

	f := callbackWrapper(c, func(_ PubSub, _ Decoder) {
		s.T().Logf("callback called")
	})

	f(c.mqttClient, &testMsg{
		duplicate: false,
		qos:       1,
		retained:  false,
		topic:     "test",
		messageID: 1,
		payload:   []byte(`payload`),
		once:      sync.Once{},
		ack:       func() {},
	})
}

type testMsg struct {
	duplicate bool
	qos       byte
	retained  bool
	topic     string
	messageID uint16
	payload   []byte
	once      sync.Once
	ack       func()
}

func (t *testMsg) Duplicate() bool {
	return t.duplicate
}

func (t *testMsg) Qos() byte {
	return t.qos
}

func (t *testMsg) Retained() bool {
	return t.retained
}

func (t *testMsg) Topic() string {
	return t.topic
}

func (t *testMsg) MessageID() uint16 {
	return t.messageID
}

func (t *testMsg) Payload() []byte {
	return t.payload
}

func (t *testMsg) Ack() {
	t.once.Do(t.ack)
}
