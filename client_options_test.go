package courier

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ClientOptionSuite struct {
	suite.Suite
}

func TestClientOptionSuite(t *testing.T) {
	suite.Run(t, new(ClientOptionSuite))
}

func (s *ClientOptionSuite) Test_apply() {
	store := NewMemoryStore()
	r := resolver{}
	mc := newMockCredentialFetcher(s.T())

	tests := []struct {
		name   string
		option ClientOption
		want   *clientOptions
	}{
		{
			name:   "WithUsername",
			option: WithUsername("test"),
			want:   &clientOptions{username: "test"},
		},
		{
			name:   "WithClientID",
			option: WithClientID("clientID"),
			want:   &clientOptions{clientID: "clientID"},
		},
		{
			name:   "WithPassword",
			option: WithPassword("password"),
			want:   &clientOptions{password: "password"},
		},
		{
			name:   "WithAutoReconnect",
			option: WithAutoReconnect(true),
			want:   &clientOptions{autoReconnect: true},
		},
		{
			name:   "WithCleanSession",
			option: WithCleanSession(true),
			want:   &clientOptions{cleanSession: true},
		},
		{
			name:   "WithMaintainOrder",
			option: WithMaintainOrder(true),
			want:   &clientOptions{maintainOrder: true},
		},
		{
			name:   "WithTCPAddress",
			option: WithTCPAddress("localhost", 9999),
			want:   &clientOptions{brokerAddress: fmt.Sprintf("%s:%d", "localhost", 9999)},
		},
		{
			name:   "WithAddress",
			option: WithAddress("localhost", 9999),
			want:   &clientOptions{brokerAddress: fmt.Sprintf("%s:%d", "localhost", 9999)},
		},
		{
			name:   "WithKeepAlive",
			option: WithKeepAlive(time.Second),
			want:   &clientOptions{keepAlive: time.Second},
		},
		{
			name:   "WithConnectTimeout",
			option: WithConnectTimeout(time.Second),
			want:   &clientOptions{connectTimeout: time.Second},
		},
		{
			name:   "WithWriteTimeout",
			option: WithWriteTimeout(time.Second),
			want:   &clientOptions{writeTimeout: time.Second},
		},
		{
			name:   "WithMaxReconnectInterval",
			option: WithMaxReconnectInterval(time.Minute),
			want:   &clientOptions{maxReconnectInterval: time.Minute},
		},
		{
			name:   "WithGracefulShutdownPeriod",
			option: WithGracefulShutdownPeriod(time.Minute),
			want:   &clientOptions{gracefulShutdownPeriod: time.Minute},
		},
		{
			name:   "WithPersistence",
			option: WithPersistence(store),
			want:   &clientOptions{store: store},
		},
		{
			name:   "WithResolver",
			option: WithResolver(r),
			want:   &clientOptions{resolver: r},
		},
		{
			name:   "WithCredentialFetcher",
			option: WithCredentialFetcher(mc),
			want:   &clientOptions{credentialFetcher: mc},
		},
		{
			name:   "WithExponentialStartOptions",
			option: WithExponentialStartOptions(WithMaxInterval(time.Second)),
			want:   &clientOptions{startOptions: &startOptions{maxInterval: time.Second}},
		},
		{
			name:   "UseMultiConnectionMode",
			option: UseMultiConnectionMode,
			want:   &clientOptions{multiConnectionMode: true},
		},
	}

	for _, t := range tests {
		s.Run(t.name, func() {
			options := &clientOptions{}
			t.option.apply(options)
			s.Equal(t.want, options)
		})
	}
}

func (s *ClientOptionSuite) Test_function_based_apply() {
	emptyErrFunc := func(error) {}
	clientFunc := func(_ PubSub) {}
	ssp := func(string) bool { return true }

	tlsConfig := &tls.Config{
		RootCAs:      nil,
		ClientAuth:   tls.NoClientCert,
		ClientCAs:    nil,
		Certificates: nil,
	}

	tests := []struct {
		name   string
		option ClientOption
		want   *clientOptions
	}{
		{
			name:   "WithOnConnect",
			option: WithOnConnect(clientFunc),
			want:   &clientOptions{onConnectHandler: clientFunc},
		},
		{
			name:   "WithOnConnectionLost",
			option: WithOnConnectionLost(emptyErrFunc),
			want:   &clientOptions{onConnectionLostHandler: emptyErrFunc},
		},
		{
			name:   "WithOnReconnect",
			option: WithOnReconnect(clientFunc),
			want:   &clientOptions{onReconnectHandler: clientFunc},
		},
		{
			name:   "WithCustomDecoder",
			option: WithCustomDecoder(base64JsonDecoder),
			want:   &clientOptions{newDecoder: base64JsonDecoder},
		},
		{
			name:   "WithCustomEncoder",
			option: WithCustomEncoder(DefaultEncoderFunc),
			want:   &clientOptions{newEncoder: DefaultEncoderFunc},
		},
		{
			name:   "WithUseBase64Decoder",
			option: WithUseBase64Decoder(),
			want:   &clientOptions{newDecoder: base64JsonDecoder},
		},
		{
			name:   "WithTLS",
			option: WithTLS(tlsConfig),
			want:   &clientOptions{tlsConfig: tlsConfig},
		},
		{
			name:   "SharedSubscriptionPredicate",
			option: SharedSubscriptionPredicate(ssp),
			want:   &clientOptions{sharedSubscriptionPredicate: ssp},
		},
	}

	for _, t := range tests {
		s.Run(t.name, func() {
			options := &clientOptions{}
			t.option.apply(options)

			val1 := fmt.Sprintf("%v", options)
			val2 := fmt.Sprintf("%v", t.want)

			s.Equal(val2, val1)
		})
	}
}

func (s *ClientOptionSuite) Test_defaultOptions() {
	o := &clientOptions{
		autoReconnect:               true,
		maintainOrder:               true,
		connectTimeout:              15 * time.Second,
		writeTimeout:                10 * time.Second,
		maxReconnectInterval:        5 * time.Minute,
		gracefulShutdownPeriod:      30 * time.Second,
		keepAlive:                   60 * time.Second,
		credentialFetchTimeout:      10 * time.Second,
		newEncoder:                  DefaultEncoderFunc,
		newDecoder:                  DefaultDecoderFunc,
		store:                       inMemoryPersistence,
		logger:                      defaultLogger,
		sharedSubscriptionPredicate: defaultSharedSubscriptionPredicate,
	}

	val1 := fmt.Sprintf("%v", o)
	val2 := fmt.Sprintf("%v", defaultClientOptions())
	s.Equal(val2, val1)
}

func Test_defaultSharedSubscriptionPredicate(t *testing.T) {
	assert.True(t, defaultSharedSubscriptionPredicate("$share/group/topic"))
	assert.False(t, defaultSharedSubscriptionPredicate("topic"))
}
