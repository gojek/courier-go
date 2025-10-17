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

func TestPoolNewPooledConnection(t *testing.T) {
	mockClient := newMockMQTTClient(t)
	conn := newPooledConnection(mockClient, "test")

	assert.NotNil(t, conn)
	assert.Equal(t, mockClient, conn.client)
	assert.Equal(t, "test", conn.id)
}

func TestPoolGetNextPoolConnection(t *testing.T) {
	mockClient := newMockMQTTClient(t)

	originalFunc := newClientFunc.Load()
	newClientFunc.Store(func(opts *mqtt.ClientOptions) mqtt.Client {
		return mockClient
	})
	defer newClientFunc.Store(originalFunc)

	client, err := NewClient(
		WithAddress("localhost", 1883),
		WithClientID("test"),
		WithPoolSize(2),
	)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	conn1 := client.getNextPoolConnection()
	conn2 := client.getNextPoolConnection()
	conn3 := client.getNextPoolConnection()

	assert.NotNil(t, conn1)
	assert.NotNil(t, conn2)
	assert.NotNil(t, conn3)

	assert.NotEqual(t, conn1, conn2)
	assert.NotEqual(t, conn2, conn3)

	assert.Equal(t, conn1, conn3)
}

func TestPoolConnection(t *testing.T) {
	mockClient := newMockMQTTClient(t)
	mockToken := newMockToken(t)
	mockToken.On("Wait").Return(true)
	mockToken.On("WaitTimeout", mock.Anything).Return(true)
	mockToken.On("Error").Return(nil)

	mockClient.On("IsConnectionOpen").Return(true)
	mockClient.On("Connect").Return(mockToken)
	mockClient.On("Disconnect", mock.Anything)
	mockClient.On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockToken)
	mockClient.On("Subscribe", mock.Anything, mock.Anything, mock.Anything).Return(mockToken)
	mockClient.On("Unsubscribe", mock.Anything).Return(mockToken)

	originalFunc := newClientFunc.Load()
	newClientFunc.Store(func(opts *mqtt.ClientOptions) mqtt.Client {
		return mockClient
	})
	defer newClientFunc.Store(originalFunc)

	client, err := NewClient(
		WithAddress("localhost", 1883),
		WithClientID("test"),
		WithPoolSize(2),
	)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	err = client.Start()
	assert.NoError(t, err)

	atomic.StoreInt64(&mockClient.publishCallCount, 0)
	atomic.StoreInt64(&mockClient.subscribeCallCount, 0)
	atomic.StoreInt64(&mockClient.unsubscribeCallCount, 0)

	err = client.Publish(context.Background(), "test/topic", []byte("test"), QOSOne)
	assert.NoError(t, err)

	publishCount := atomic.LoadInt64(&mockClient.publishCallCount)
	assert.Equal(t, publishCount, int64(1), "Publish should be called")

	err = client.Subscribe(context.Background(), "test/topic", func(ctx context.Context, ps PubSub, m *Message) {}, QOSOne)
	assert.NoError(t, err)

	subscribeCount := atomic.LoadInt64(&mockClient.subscribeCallCount)

	err = client.Unsubscribe(context.Background(), "test/topic")
	assert.NoError(t, err)

	unsubscribeCount := atomic.LoadInt64(&mockClient.unsubscribeCallCount)

	assert.Equal(t, publishCount, int64(1), "Publish should be called")
	assert.Equal(t, subscribeCount, int64(1), "Subscribe should be called")
	assert.Equal(t, unsubscribeCount, int64(1), "Unsubscribe should be called")

	assert.NoError(t, client.stop())
}
