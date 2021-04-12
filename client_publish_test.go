package courier

import (
	"bytes"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"***REMOVED***/metrics"
)

type ClientPublishSuite struct {
	suite.Suite
}

func TestClientPublishSuite(t *testing.T) {
	suite.Run(t, new(ClientPublishSuite))
}

func (s *ClientPublishSuite) TestPublish() {
	tests := []struct {
		name     string
		payload  interface{}
		pahoMock func(*mock.Mock, interface{}) *mockToken
		wantErr  bool
	}{
		{
			name: "Success",
			pahoMock: func(m *mock.Mock, p interface{}) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(true)
				t.On("Error").Return(nil)
				buf := bytes.Buffer{}
				_ = defaultEncoderFunc(&buf).Encode(p)
				m.On("Publish", "topic", byte(QOSOne), false, buf.Bytes()).Return(t)
				return t
			},
			payload: "payload",
		},
		{
			name: "WaitTimeout",
			pahoMock: func(m *mock.Mock, p interface{}) *mockToken {
				t := &mockToken{}
				t.On("WaitTimeout", 10*time.Second).Return(false)
				buf := bytes.Buffer{}
				_ = defaultEncoderFunc(&buf).Encode(p)
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
				_ = defaultEncoderFunc(&buf).Encode(p)
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
			c, err := NewClient(WithCustomMetrics(metrics.NewPrometheus()))
			s.NoError(err)

			mc := &mockClient{}
			c.mqttClient = mc
			tk := t.pahoMock(&mc.Mock, t.payload)

			err = c.Publish("topic", QOSOne, false, t.payload)

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
