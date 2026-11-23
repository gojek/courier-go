package mqttv5

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/eclipse/paho.golang/paho"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mqtt "github.com/gojek/paho.mqtt.golang"
)

// capturePrinter records what was logged so a test can assert on it.
type capturePrinter struct{ lines []string }

func (c *capturePrinter) Println(v ...interface{}) { c.lines = append(c.lines, fmt.Sprint(v...)) }

func (c *capturePrinter) Printf(format string, v ...interface{}) {
	c.lines = append(c.lines, fmt.Sprintf(format, v...))
}

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

func newTestClient(t *testing.T, broker *url.URL, clean bool) *Client {
	t.Helper()

	opts := mqtt.NewClientOptions().SetClientID(t.Name()).SetCleanSession(clean).SetKeepAlive(30 * time.Second)
	opts.Servers = []*url.URL{broker}

	c := New(opts, Options{CleanStart: clean})

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

func TestV5_PublishSubscribe(t *testing.T) {
	c := newTestClient(t, brokerOrSkip(t), true)

	got := make(chan mqtt.Message, 1)
	wait(t, c.Subscribe("v5/test/+", 1, func(_ mqtt.Client, m mqtt.Message) { got <- m }))

	wait(t, c.Publish("v5/test/a", 1, false, []byte("hello")))

	select {
	case m := <-got:
		assert.Equal(t, "v5/test/a", m.Topic())
		assert.Equal(t, []byte("hello"), m.Payload())
	case <-time.After(5 * time.Second):
		t.Fatal("message not received")
	}

	wait(t, c.Unsubscribe("v5/test/+"))
}

func TestV5_NotConnected(t *testing.T) {
	c := New(mqtt.NewClientOptions(), Options{})

	assert.False(t, c.IsConnected())
	assert.ErrorIs(t, c.Publish("a", 0, false, "x").Error(), ErrNotStarted)
	assert.ErrorIs(t, c.Subscribe("a", 0, nil).Error(), ErrNotStarted)
}

func TestV5_UnsupportedPayload(t *testing.T) {
	c := New(mqtt.NewClientOptions(), Options{})

	assert.Error(t, c.Publish("a", 0, false, 42).Error())
}

func TestV5_ConnectFailureWithoutRetry(t *testing.T) {
	opts := mqtt.NewClientOptions().SetConnectRetry(false).SetConnectTimeout(time.Second)
	opts.Servers = []*url.URL{{Scheme: "tcp", Host: "127.0.0.1:1"}}

	tk := New(opts, Options{}).Connect()

	require.True(t, tk.WaitTimeout(5*time.Second))
	assert.Error(t, tk.Error())
}

func TestV5_SessionExpiryAndCleanStart(t *testing.T) {
	cfg := New(mqtt.NewClientOptions(), Options{CleanStart: true, SessionExpiryInterval: 90 * time.Second}).
		config(newToken(), func() {})

	assert.True(t, cfg.CleanStartOnInitialConnection)
	assert.EqualValues(t, 90, cfg.SessionExpiryInterval)

	// CleanSession is a v3.1.1 setting and must not leak into v5
	cfg = New(mqtt.NewClientOptions().SetCleanSession(true), Options{}).config(newToken(), func() {})
	assert.False(t, cfg.CleanStartOnInitialConnection)
	assert.Zero(t, cfg.SessionExpiryInterval)

	cfg = New(mqtt.NewClientOptions(), Options{SessionExpiryInterval: 200 * 365 * 24 * time.Hour}).
		config(newToken(), func() {})
	assert.EqualValues(t, uint32(math.MaxUint32), cfg.SessionExpiryInterval)
}

func TestReconnectBackoff_InitialConnect(t *testing.T) {
	v := New(mqtt.NewClientOptions().SetConnectRetryInterval(5*time.Second), Options{})

	// v3 retries the first connection at a fixed interval, it does not back off
	assert.Equal(t, time.Duration(0), v.reconnectBackoff(0))
	for _, attempt := range []int{1, 2, 10} {
		assert.Equal(t, 5*time.Second, v.reconnectBackoff(attempt), "attempt %d", attempt)
	}

	d := New(mqtt.NewClientOptions().SetConnectRetryInterval(0), Options{})
	assert.Equal(t, defaultConnectRetryInterval, d.reconnectBackoff(1))
}

func TestReconnectBackoff_AfterLoss(t *testing.T) {
	now := time.Now()
	nowFunc = func() time.Time { return now }

	t.Cleanup(func() { nowFunc = time.Now })

	v := New(mqtt.NewClientOptions().SetMaxReconnectInterval(8*time.Second), Options{})
	v.reconnecting.Store(true)

	// autopaho resets attempt to 0 on every CONNACK, so a flapping connection always asks for attempt 0
	for _, want := range []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 8 * time.Second} {
		assert.Equal(t, want, v.reconnectBackoff(0))

		now = now.Add(want)
	}

	// a connection that held starts over
	now = now.Add(time.Minute)
	assert.Equal(t, time.Duration(0), v.reconnectBackoff(0))

	u := New(mqtt.NewClientOptions().SetMaxReconnectInterval(0), Options{})
	u.reconnecting.Store(true)
	u.lastAttempt, u.lastDelay = now, defaultMaxReconnectInterval

	assert.Equal(t, defaultMaxReconnectInterval, u.reconnectBackoff(0))
}

func TestV5_OnConnectError(t *testing.T) {
	t.Run("initial connect fails the token when ConnectRetry is off", func(t *testing.T) {
		v := New(mqtt.NewClientOptions().SetConnectRetry(false), Options{})

		tok, cancelled := newToken(), false
		v.config(tok, func() { cancelled = true }).OnConnectError(errors.New("refused"))

		require.True(t, tok.WaitTimeout(time.Second))
		assert.EqualError(t, tok.Error(), "refused")
		assert.True(t, cancelled)
	})

	t.Run("each reconnect attempt calls OnReconnecting and never cancels", func(t *testing.T) {
		calls := make(chan struct{}, 4)
		opts := mqtt.NewClientOptions().SetConnectRetry(false).SetAutoReconnect(true).
			SetReconnectingHandler(func(mqtt.Client, *mqtt.ClientOptions) { calls <- struct{}{} })

		v := New(opts, Options{})
		v.up.Store(true)

		tok, cancelled := newToken(), false
		cancel := func() { cancelled = true }
		cfg := v.config(tok, cancel)

		v.connectionLost(errors.New("lost"), cancel) // fires OnReconnecting once, like v3's first retry
		cfg.OnConnectError(errors.New("refused"))    // then once per failed attempt
		cfg.OnConnectError(errors.New("refused"))

		for i := 0; i < 3; i++ {
			select {
			case <-calls:
			case <-time.After(time.Second):
				t.Fatalf("OnReconnecting called %d times, want 3", i)
			}
		}

		assert.False(t, cancelled)
	})
}

func TestV5_DeclaresEveryClientMethod(t *testing.T) {
	// V5 embeds a never-connected v3 client so OptionsReader keeps working. Any other
	// mqtt.Client method it does not declare itself would silently reach that client
	// instead of the broker, so fail here if the interface grows one.
	src, err := os.ReadFile("client.go")
	require.NoError(t, err)

	iface := reflect.TypeOf((*mqtt.Client)(nil)).Elem()

	for i := 0; i < iface.NumMethod(); i++ {
		name := iface.Method(i).Name
		if name == "OptionsReader" {
			continue
		}

		assert.Contains(t, string(src), "func (v *Client) "+name+"(", "V5 must declare %s itself", name)
	}
}

func TestV5_PublishWaitsForConnection(t *testing.T) {
	broker := brokerOrSkip(t)

	opts := mqtt.NewClientOptions().SetClientID(t.Name()).SetAutoReconnect(true).
		SetKeepAlive(30 * time.Second).SetAckTimeout(10 * time.Second)
	opts.Servers = []*url.URL{broker}

	c := New(opts, Options{CleanStart: true})

	c.Connect() // publish without waiting for CONNACK, the connection is still down
	t.Cleanup(func() { c.Disconnect(100) })

	wait(t, c.Publish("metric/one", 1, false, "queued-until-up"))
	assert.True(t, c.IsConnected())
}

func TestV5_PublishFailsWhenConnectionStaysDown(t *testing.T) {
	opts := mqtt.NewClientOptions().SetClientID(t.Name()).SetAutoReconnect(true).SetAckTimeout(time.Second)
	opts.Servers = []*url.URL{{Scheme: "tcp", Host: "127.0.0.1:1"}}

	c := New(opts, Options{CleanStart: true})

	c.Connect()
	t.Cleanup(func() { c.Disconnect(100) })

	tk := c.Publish("metric/one", 1, false, "never-sent")

	// the v5 library blocks until it has an answer, so the token is complete on return
	select {
	case <-tk.Done():
	default:
		t.Fatal("token must already be complete, the call is synchronous")
	}

	assert.Error(t, tk.Error())
}

func TestV5_ReconnectResubscribesAndFiresHooks(t *testing.T) {
	broker := brokerOrSkip(t)

	lost := make(chan error, 4)
	reconnecting := make(chan struct{}, 4)
	connected := make(chan struct{}, 4)

	opts := mqtt.NewClientOptions().SetClientID(t.Name()).SetAutoReconnect(true).
		SetKeepAlive(30 * time.Second).SetAckTimeout(10 * time.Second).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) { lost <- err }).
		SetReconnectingHandler(func(mqtt.Client, *mqtt.ClientOptions) { reconnecting <- struct{}{} }).
		SetOnConnectHandler(func(mqtt.Client) { connected <- struct{}{} })
	opts.Servers = []*url.URL{broker}

	// CleanStart drops the session, so the broker reports no session and the client must resubscribe
	c := New(opts, Options{CleanStart: true})

	tk := c.Connect()
	require.True(t, tk.WaitTimeout(5*time.Second))
	require.NoError(t, tk.Error())

	t.Cleanup(func() { c.Disconnect(100) })
	recv(t, connected, "OnConnect after the first connection")

	got := make(chan mqtt.Message, 1)
	wait(t, c.Subscribe("v5/reconnect", 1, func(_ mqtt.Client, m mqtt.Message) { got <- m }))

	c.cm.TerminateConnectionForTest()

	recv(t, lost, "OnConnectionLost after the connection drops")
	recv(t, reconnecting, "OnReconnecting after the connection drops")
	recv(t, connected, "OnConnect after the reconnect")

	require.Eventually(t, c.IsConnected, 10*time.Second, 50*time.Millisecond)

	// the subscription survives only if resubscribe ran on the new connection
	wait(t, c.Publish("v5/reconnect", 1, false, "after-reconnect"))

	select {
	case m := <-got:
		assert.Equal(t, []byte("after-reconnect"), m.Payload())
	case <-time.After(5 * time.Second):
		t.Fatal("message not received after the reconnect, the resubscribe did not happen")
	}
}

func recv[T any](t *testing.T, ch <-chan T, what string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func TestV5_ConnectIsNotReentrant(t *testing.T) {
	o := mqtt.NewClientOptions().SetConnectRetry(true).SetConnectRetryInterval(time.Hour)
	o.Servers = []*url.URL{{Scheme: "tcp", Host: "127.0.0.1:1"}} // nothing listens, so the attempt stays open

	v := New(o, Options{})
	v.reconnecting.Store(true)

	first := v.Connect()
	t.Cleanup(func() { v.Disconnect(10) })

	cm := v.cm

	assert.False(t, v.reconnecting.Load(), "Connect starts a fresh attempt")
	assert.Same(t, first, v.Connect(), "a second Connect joins the attempt already running")
	assert.Same(t, cm, v.cm, "a second connection manager would fight the first over the client ID")

	v.Disconnect(10)
	assert.NotSame(t, first, v.Connect(), "Connect after Disconnect starts a new attempt")
}

func TestTopicMatches(t *testing.T) {
	for _, tc := range []struct {
		filter, topic string
		want          bool
	}{
		{"a/b", "a/b", true},
		{"a/b", "a/c", false},
		{"a/+", "a/b", true},
		{"a/+", "a/b/c", false},
		{"a/+/c", "a/b/c", true},
		{"a/#", "a/b/c", true},
		{"a/#", "a", true},
		{"#", "a/b", true},
		{"a/b", "a/b/c", false},
		{"a/b/c", "a/b", false},
		{"$share/group/a/b", "a/b", true},
		{"$share/group/a/#", "a/b/c", true},
		{"$share/group/a/b", "a/c", false},
	} {
		assert.Equal(t, tc.want, topicMatches(tc.filter, tc.topic), "%q vs %q", tc.filter, tc.topic)
	}
}

func TestV5_HandlerCanSubscribe(t *testing.T) {
	// paho's own router runs handlers under a read lock it also needs to register one
	v := New(mqtt.NewClientOptions(), Options{})

	done := make(chan struct{})

	v.AddRoute("a/b", func(c mqtt.Client, _ mqtt.Message) {
		c.Subscribe("reply/1", 1, func(mqtt.Client, mqtt.Message) {})
		c.Unsubscribe("a/b")
		close(done)
	})

	go func() { _, _ = v.route(paho.PublishReceived{Packet: &paho.Publish{Topic: "a/b"}}) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a handler that subscribes deadlocked")
	}
}

func TestV5_UnmatchedMessageIsLogged(t *testing.T) {
	v := New(mqtt.NewClientOptions().SetLogLevel(mqtt.LogLevelError), Options{})

	p := &capturePrinter{}
	v.errs = p

	handled, err := v.route(paho.PublishReceived{Packet: &paho.Publish{Topic: "nobody/listening"}})

	assert.NoError(t, err)
	assert.False(t, handled)
	require.Len(t, p.lines, 1)
	assert.Contains(t, p.lines[0], "nobody/listening")
}
