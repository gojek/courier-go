package driver

import (
	"math"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mqtt "github.com/gojek/paho.mqtt.golang"
)

func brokerOrSkip(t *testing.T) *url.URL {
	t.Helper()

	addr := os.Getenv("BROKER_ADDRESS")
	if addr == "" {
		addr = "localhost:1883"
	}

	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Skipf("no MQTT broker at %s: %v", addr, err)
	}

	_ = conn.Close()

	return &url.URL{Scheme: "tcp", Host: addr}
}

func newTestClient(t *testing.T, broker *url.URL, clean bool) *V5 {
	t.Helper()

	opts := mqtt.NewClientOptions().SetClientID(t.Name()).SetCleanSession(clean).SetKeepAlive(30 * time.Second)
	opts.Servers = []*url.URL{broker}

	c := NewV5(opts, V5Options{CleanStart: clean})

	tk := c.Connect()
	require.True(t, tk.WaitTimeout(5*time.Second))
	require.NoError(t, tk.Error())
	require.True(t, c.IsConnected())

	t.Cleanup(func() { c.Disconnect(100) })

	return c
}

func wait(t *testing.T, tk mqtt.Token) {
	t.Helper()

	require.True(t, tk.WaitTimeout(5*time.Second))
	require.NoError(t, tk.Error())
}

func TestV5_PublishSubscribeWithProps(t *testing.T) {
	c := newTestClient(t, brokerOrSkip(t), true)

	got := make(chan mqtt.Message, 1)
	wait(t, c.Subscribe("v5/test/+", 1, func(_ mqtt.Client, m mqtt.Message) { got <- m }))

	wait(t, c.PublishWithProps("v5/test/a", 1, false, []byte("hello"), &PublishProps{
		UserProperties: map[string]string{"traceparent": "00-abc-def-01"},
		ContentType:    "text/plain",
	}))

	select {
	case m := <-got:
		assert.Equal(t, "v5/test/a", m.Topic())
		assert.Equal(t, []byte("hello"), m.Payload())

		p := m.(MessageWithProps).Props()
		assert.Equal(t, map[string]string{"traceparent": "00-abc-def-01"}, p.UserProperties)
		assert.Equal(t, "text/plain", p.ContentType)
	case <-time.After(5 * time.Second):
		t.Fatal("message not received")
	}

	wait(t, c.Unsubscribe("v5/test/+"))
}

func TestV5_PublishWithoutProps(t *testing.T) {
	c := newTestClient(t, brokerOrSkip(t), true)

	got := make(chan mqtt.Message, 1)
	wait(t, c.Subscribe("v5/plain", 0, func(_ mqtt.Client, m mqtt.Message) { got <- m }))
	wait(t, c.Publish("v5/plain", 0, false, "text"))

	select {
	case m := <-got:
		assert.Equal(t, []byte("text"), m.Payload())
		assert.Empty(t, m.(MessageWithProps).Props().UserProperties)
	case <-time.After(5 * time.Second):
		t.Fatal("message not received")
	}
}

func TestV5_NotConnected(t *testing.T) {
	c := NewV5(mqtt.NewClientOptions(), V5Options{})

	assert.False(t, c.IsConnected())
	assert.ErrorIs(t, c.Publish("a", 0, false, "x").Error(), ErrNotStarted)
	assert.ErrorIs(t, c.Subscribe("a", 0, nil).Error(), ErrNotStarted)
}

func TestV5_UnsupportedPayload(t *testing.T) {
	c := NewV5(mqtt.NewClientOptions(), V5Options{})

	assert.Error(t, c.Publish("a", 0, false, 42).Error())
}

func TestV5_ConnectFailureWithoutRetry(t *testing.T) {
	opts := mqtt.NewClientOptions().SetConnectRetry(false).SetConnectTimeout(time.Second)
	opts.Servers = []*url.URL{{Scheme: "tcp", Host: "127.0.0.1:1"}}

	tk := NewV5(opts, V5Options{}).Connect()

	require.True(t, tk.WaitTimeout(5*time.Second))
	assert.Error(t, tk.Error())
}

func TestV5_SessionExpiryAndCleanStart(t *testing.T) {
	cfg := NewV5(mqtt.NewClientOptions(), V5Options{CleanStart: true, SessionExpiryInterval: 90 * time.Second}).
		config(newToken(), func() {})

	assert.True(t, cfg.CleanStartOnInitialConnection)
	assert.EqualValues(t, 90, cfg.SessionExpiryInterval)

	// CleanSession is a v3.1.1 setting and must not leak into v5
	cfg = NewV5(mqtt.NewClientOptions().SetCleanSession(true), V5Options{}).config(newToken(), func() {})
	assert.False(t, cfg.CleanStartOnInitialConnection)
	assert.Zero(t, cfg.SessionExpiryInterval)

	cfg = NewV5(mqtt.NewClientOptions(), V5Options{SessionExpiryInterval: 200 * 365 * 24 * time.Hour}).
		config(newToken(), func() {})
	assert.EqualValues(t, uint32(math.MaxUint32), cfg.SessionExpiryInterval)
}
