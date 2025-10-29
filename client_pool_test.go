package courier

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mqtt "github.com/gojek/paho.mqtt.golang"
)

type mockMQTTClient struct {
	mockClient
	publishCallCount     int64
	subscribeCallCount   int64
	unsubscribeCallCount int64
}

func (m *mockMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	atomic.AddInt64(&m.publishCallCount, 1)
	return m.Called(topic, qos, retained, payload).Get(0).(mqtt.Token)
}

func (m *mockMQTTClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	atomic.AddInt64(&m.subscribeCallCount, 1)
	return m.Called(topic, qos, callback).Get(0).(mqtt.Token)
}

func (m *mockMQTTClient) Unsubscribe(topics ...string) mqtt.Token {
	atomic.AddInt64(&m.unsubscribeCallCount, 1)
	return m.Called(topics).Get(0).(mqtt.Token)
}

func newMockMQTTClient(t *testing.T) *mockMQTTClient {
	m := &mockMQTTClient{}
	m.Test(t)
	return m
}

func TestPoolSize(t *testing.T) {
	tests := []struct {
		name       string
		poolSize   int
		shouldPool bool
	}{
		{
			name:       "Pool size 2",
			poolSize:   2,
			shouldPool: true,
		},
		{
			name:       "Pool size 0",
			poolSize:   0,
			shouldPool: false,
		},
		{
			name:       "Pool size 1",
			poolSize:   1,
			shouldPool: false,
		},
		{
			name:       "Pool size -2",
			poolSize:   -2,
			shouldPool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newMockMQTTClient(t)
			mockToken := newMockToken(t)
			mockToken.On("Wait").Return(true)
			mockToken.On("WaitTimeout", mock.Anything).Return(true)
			mockToken.On("Error").Return(nil)

			mockClient.On("IsConnectionOpen").Return(true)
			mockClient.On("Connect").Return(mockToken)
			mockClient.On("Disconnect", mock.Anything)

			originalFunc := newClientFunc.Load()
			newClientFunc.Store(func(opts *mqtt.ClientOptions) mqtt.Client {
				return mockClient
			})
			defer newClientFunc.Store(originalFunc)

			client, err := NewClient(
				WithAddress("localhost", 1883),
				WithClientID("test"),
				WithPoolSize(tt.poolSize),
			)
			assert.NoError(t, err)
			assert.NotNil(t, client)

			assert.Equal(t, tt.shouldPool, client.options.poolEnabled)

			if tt.shouldPool {
				assert.NotNil(t, client.mqttClients, "Pool should be initialized")
				assert.Equal(t, tt.poolSize, len(client.mqttClients), "Pool should have correct size")
				assert.Equal(t, tt.poolSize, client.options.poolSize, "Pool size should match")

				assert.Len(t, client.mqttClients, tt.poolSize)
				for _, state := range client.mqttClients {
					assert.NotNil(t, state, "Should get a valid connection state")
					assert.NotNil(t, state.client, "Should have a valid MQTT client")
				}
			} else {
				assert.Nil(t, client.mqttClients, "Pool should be nil")
				assert.NotNil(t, client.mqttClient, "Should have single mqtt client")
			}
		})
	}
}

func TestPoolPublish(t *testing.T) {
	tests := []struct {
		name          string
		poolSize      int
		publishCount  int
		expectedCalls int
	}{
		{
			name:          "Pool size 2",
			poolSize:      2,
			publishCount:  4,
			expectedCalls: 2,
		},
		{
			name:          "Pool size 3",
			poolSize:      3,
			publishCount:  12,
			expectedCalls: 4,
		},
		{
			name:          "Pool size 4",
			poolSize:      4,
			publishCount:  24,
			expectedCalls: 6,
		},
	}

	for _, tt := range tests {
		mockClients := make([]*mockMQTTClient, tt.poolSize)
		mockTokens := make([]*mockToken, tt.poolSize)

		for i := 0; i < tt.poolSize; i++ {
			mockClients[i] = newMockMQTTClient(t)
			mockTokens[i] = newMockToken(t)

			mockTokens[i].On("Wait").Return(true)
			mockTokens[i].On("WaitTimeout", mock.Anything).Return(true)
			mockTokens[i].On("Error").Return(nil)

			mockClients[i].On("IsConnectionOpen").Return(true)
			mockClients[i].On("Connect").Return(mockTokens[i])
			mockClients[i].On("Disconnect", mock.Anything)
			mockClients[i].On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockTokens[i])
		}

		clientIndex := 0
		originalFunc := newClientFunc.Load()
		newClientFunc.Store(func(opts *mqtt.ClientOptions) mqtt.Client {
			client := mockClients[clientIndex%tt.poolSize]
			clientIndex++
			return client
		})
		defer newClientFunc.Store(originalFunc)

		client, err := NewClient(
			WithAddress("localhost", 1883),
			WithClientID("test"),
			WithPoolSize(tt.poolSize),
		)
		assert.NoError(t, err)
		assert.NotNil(t, client)

		err = client.Start()
		assert.NoError(t, err)

		for i := range tt.poolSize {
			atomic.StoreInt64(&mockClients[i].publishCallCount, 0)
		}

		for i := 0; i < tt.publishCount; i++ {
			err = client.Publish(context.Background(), "topic", []byte("test"))
			assert.NoError(t, err)
		}

		totalPublishCount := int64(0)
		for i := range tt.poolSize {
			totalPublishCount += atomic.LoadInt64(&mockClients[i].publishCallCount)
			assert.Equal(t, int64(tt.expectedCalls), atomic.LoadInt64(&mockClients[i].publishCallCount), "Each client should have correct publish call count")
		}

		assert.Equal(t, int64(tt.publishCount), totalPublishCount, "Total publish count should match expected")
		assert.NoError(t, client.stop())
	}
}

func TestPoolPublishRoundRobin(t *testing.T) {
	poolSize := 3
	mockClients := make([]*mockMQTTClient, poolSize)
	mockTokens := make([]*mockToken, poolSize)

	for i := 0; i < poolSize; i++ {
		mockClients[i] = newMockMQTTClient(t)
		mockTokens[i] = newMockToken(t)

		mockTokens[i].On("Wait").Return(true)
		mockTokens[i].On("WaitTimeout", mock.Anything).Return(true)
		mockTokens[i].On("Error").Return(nil)

		mockClients[i].On("IsConnectionOpen").Return(true)
		mockClients[i].On("Connect").Return(mockTokens[i])
		mockClients[i].On("Disconnect", mock.Anything)
		mockClients[i].On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockTokens[i])
	}

	clientIndex := 0
	originalFunc := newClientFunc.Load()
	newClientFunc.Store(func(opts *mqtt.ClientOptions) mqtt.Client {
		client := mockClients[clientIndex%poolSize]
		clientIndex++
		return client
	})
	defer newClientFunc.Store(originalFunc)

	client, err := NewClient(
		WithAddress("localhost", 1883),
		WithClientID("test"),
		WithPoolSize(poolSize),
	)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	err = client.Start()
	assert.NoError(t, err)

	for i := range poolSize {
		atomic.StoreInt64(&mockClients[i].publishCallCount, 0)
	}

	expectedPublishes := 6
	for i := 0; i < expectedPublishes; i++ {
		err = client.Publish(context.Background(), "topic", []byte("test"))
		assert.NoError(t, err)
	}

	totalPublishCount := int64(0)
	for i := range poolSize {
		totalPublishCount += atomic.LoadInt64(&mockClients[i].publishCallCount)
	}

	assert.Equal(t, int64(expectedPublishes), totalPublishCount, "Total publish count should match expected")
	assert.NoError(t, client.stop())
}

func TestPoolSubscribe(t *testing.T) {
	poolSize := 3
	mockClients := make([]*mockMQTTClient, poolSize)
	mockTokens := make([]*mockToken, poolSize)

	for i := 0; i < poolSize; i++ {
		mockClients[i] = newMockMQTTClient(t)
		mockTokens[i] = newMockToken(t)

		mockTokens[i].On("Wait").Return(true)
		mockTokens[i].On("WaitTimeout", mock.Anything).Return(true)
		mockTokens[i].On("Error").Return(nil)

		mockClients[i].On("IsConnectionOpen").Return(true)
		mockClients[i].On("Connect").Return(mockTokens[i])
		mockClients[i].On("Disconnect", mock.Anything)

		mockClients[i].On("Subscribe", mock.Anything, mock.Anything, mock.Anything).Return(mockTokens[i])
		mockClients[i].On("Unsubscribe", mock.Anything).Return(mockTokens[i])
	}

	clientIndex := 0
	originalFunc := newClientFunc.Load()
	newClientFunc.Store(func(opts *mqtt.ClientOptions) mqtt.Client {
		client := mockClients[clientIndex%poolSize]
		clientIndex++
		return client
	})
	defer newClientFunc.Store(originalFunc)

	client, err := NewClient(
		WithAddress("localhost", 1883),
		WithClientID("test"),
		WithPoolSize(poolSize),
	)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	err = client.Start()
	assert.NoError(t, err)

	for i := range poolSize {
		atomic.StoreInt64(&mockClients[i].subscribeCallCount, 0)
		atomic.StoreInt64(&mockClients[i].unsubscribeCallCount, 0)
	}

	subsCallback := func(ctx context.Context, ps PubSub, m *Message) {}
	err = client.Subscribe(context.Background(), "normal/topic", subsCallback, QOSOne)
	assert.NoError(t, err)

	normalSubCount := int64(0)
	for i := range poolSize {
		normalSubCount += atomic.LoadInt64(&mockClients[i].subscribeCallCount)
	}
	assert.Equal(t, int64(1), normalSubCount, "Normal subscription should use only one client")

	for i := range poolSize {
		atomic.StoreInt64(&mockClients[i].subscribeCallCount, 0)
	}

	err = client.Subscribe(context.Background(), "$share/group/shared/topic", subsCallback, QOSOne)
	assert.NoError(t, err)

	sharedSubCount := int64(0)
	for i := range poolSize {
		sharedSubCount += atomic.LoadInt64(&mockClients[i].subscribeCallCount)
	}
	assert.Equal(t, int64(poolSize), sharedSubCount, "Shared subscription should be attempted on all pool clients")

	for i := range poolSize {
		atomic.StoreInt64(&mockClients[i].subscribeCallCount, 0)
	}

	err = client.Subscribe(context.Background(), "$share/group/shared/topic", subsCallback, QOSOne)
	assert.NoError(t, err)

	duplicateSubCount := int64(0)
	for i := range poolSize {
		duplicateSubCount += atomic.LoadInt64(&mockClients[i].subscribeCallCount)
	}
	assert.Equal(t, int64(0), duplicateSubCount, "Duplicate shared subscription should be 0")

	err = client.Unsubscribe(context.Background(), "normal/topic")
	assert.NoError(t, err)

	unsubCount := int64(0)
	for i := range poolSize {
		unsubCount += atomic.LoadInt64(&mockClients[i].unsubscribeCallCount)
	}
	assert.GreaterOrEqual(t, unsubCount, int64(1), "Unsubscribe should be called at least once")

	assert.NoError(t, client.stop())
}
