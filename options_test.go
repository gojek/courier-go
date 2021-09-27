package courier

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"***REMOVED***/metrics"
)

type ClientOptionSuite struct {
	suite.Suite
}

func TestClientOptionSuite(t *testing.T) {
	suite.Run(t, new(ClientOptionSuite))
}

func (s *ClientOptionSuite) Test_apply() {
	mCollector := metrics.NewPrometheus()
	store := NewMemoryStore()

	tests := []struct {
		name   string
		option Option
		want   *options
	}{
		{
			name:   "WithUsername",
			option: WithUsername("test"),
			want:   &options{username: "test"},
		},
		{
			name:   "WithClientID",
			option: WithClientID("clientID"),
			want:   &options{clientID: "clientID"},
		},
		{
			name:   "WithPassword",
			option: WithPassword("password"),
			want:   &options{password: "password"},
		},
		{
			name:   "WithAutoReconnect",
			option: WithAutoReconnect(true),
			want:   &options{autoReconnect: true},
		},
		{
			name:   "WithCleanSession",
			option: WithCleanSession(true),
			want:   &options{cleanSession: true},
		},
		{
			name:   "WithMaintainOrder",
			option: WithMaintainOrder(true),
			want:   &options{maintainOrder: true},
		},
		{
			name:   "WithTCPAddress",
			option: WithTCPAddress("localhost", 9999),
			want:   &options{brokerAddress: fmt.Sprintf("tcp://%s:%d", "localhost", 9999)},
		},
		{
			name:   "WithKeepAlive",
			option: WithKeepAlive(time.Second),
			want:   &options{keepAlive: time.Second},
		},
		{
			name:   "WithConnectTimeout",
			option: WithConnectTimeout(time.Second),
			want:   &options{connectTimeout: time.Second},
		},
		{
			name:   "WithWriteTimeout",
			option: WithWriteTimeout(time.Second),
			want:   &options{writeTimeout: time.Second},
		},
		{
			name:   "WithMaxReconnectInterval",
			option: WithMaxReconnectInterval(time.Minute),
			want:   &options{maxReconnectInterval: time.Minute},
		},
		{
			name:   "WithGracefulShutdownPeriod",
			option: WithGracefulShutdownPeriod(time.Minute),
			want:   &options{gracefulShutdownPeriod: time.Minute},
		},
		{
			name:   "WithCustomMetrics",
			option: WithCustomMetrics(mCollector),
			want:   &options{metricsCollector: mCollector},
		},
		{
			name:   "WithPersistence",
			option: WithPersistence(store),
			want:   &options{store: store},
		},
	}

	for _, t := range tests {
		s.Run(t.name, func() {
			options := &options{}
			t.option(options)
			s.Equal(t.want, options)
		})
	}
}

func (s *ClientOptionSuite) Test_function_based_apply() {
	emptyErrFunc := func(error) {}
	clientFunc := func(_ PubSub) {}

	tests := []struct {
		name   string
		option Option
		want   *options
	}{
		{
			name:   "WithOnConnect",
			option: WithOnConnect(clientFunc),
			want:   &options{onConnectHandler: clientFunc},
		},
		{
			name:   "WithOnConnectionLost",
			option: WithOnConnectionLost(emptyErrFunc),
			want:   &options{onConnectionLostHandler: emptyErrFunc},
		},
		{
			name:   "WithOnReconnect",
			option: WithOnReconnect(clientFunc),
			want:   &options{onReconnectHandler: clientFunc},
		},
		{
			name:   "WithCustomDecoder",
			option: WithCustomDecoder(base64JsonDecoder),
			want:   &options{newDecoder: base64JsonDecoder},
		},
		{
			name:   "WithCustomEncoder",
			option: WithCustomEncoder(defaultEncoderFunc),
			want:   &options{newEncoder: defaultEncoderFunc},
		},
		{
			name:   "WithUseBase64Decoder",
			option: WithUseBase64Decoder(),
			want:   &options{newDecoder: base64JsonDecoder},
		},
	}

	for _, t := range tests {
		s.Run(t.name, func() {
			options := &options{}
			t.option(options)

			val1 := fmt.Sprintf("%v", options)
			val2 := fmt.Sprintf("%v", t.want)

			s.Equal(val2, val1)
		})
	}
}

func (s *ClientOptionSuite) Test_defaultOptions() {
	o := &options{
		brokerAddress:          fmt.Sprintf("tcp://%s:%d", "127.0.0.1", 1883),
		autoReconnect:          true,
		maintainOrder:          true,
		connectTimeout:         15 * time.Second,
		writeTimeout:           10 * time.Second,
		maxReconnectInterval:   5 * time.Minute,
		gracefulShutdownPeriod: 30 * time.Second,
		keepAlive:              60 * time.Second,
		newEncoder:             defaultEncoderFunc,
		newDecoder:             defaultDecoderFunc,
		store:                  inMemoryPersistence,
	}

	val1 := fmt.Sprintf("%v", o)
	val2 := fmt.Sprintf("%v", defaultOptions())
	s.Equal(val2, val1)
}
