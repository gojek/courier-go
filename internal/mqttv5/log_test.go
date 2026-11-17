package mqttv5

import (
	"errors"
	"fmt"
	"testing"

	"github.com/eclipse/paho.golang/paho"
	"github.com/eclipse/paho.golang/paho/log"
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

func TestLogResubscribeFailures(t *testing.T) {
	subs := []paho.SubscribeOptions{{Topic: "a"}, {Topic: "b"}, {Topic: "c"}}

	t.Run("a failed call is reported once", func(t *testing.T) {
		p := &capturePrinter{}
		logResubscribeFailures(p, subs, nil, errors.New("connection down"))

		require.Len(t, p.lines, 1)
		assert.Contains(t, p.lines[0], "3 topic(s)")
		assert.Contains(t, p.lines[0], "connection down")
	})

	t.Run("each rejected topic is named", func(t *testing.T) {
		p := &capturePrinter{}
		// granted, not authorized, granted
		logResubscribeFailures(p, subs, &paho.Suback{Reasons: []byte{1, 0x87, 1}}, nil)

		require.Len(t, p.lines, 1)
		assert.Contains(t, p.lines[0], `"b"`)
		assert.Contains(t, p.lines[0], "0x87")
	})

	t.Run("a granted resubscribe logs nothing", func(t *testing.T) {
		p := &capturePrinter{}
		logResubscribeFailures(p, subs, &paho.Suback{Reasons: []byte{0, 1, 2}}, nil)

		assert.Empty(t, p.lines)
	})
}

func TestPahoLoggers(t *testing.T) {
	for level, want := range map[mqtt.LogLevel][2]bool{ // {debug is real, errors is real}
		mqtt.LogLevelDefault: {false, false},
		mqtt.LogLevelDebug:   {true, true},
		mqtt.LogLevelWarn:    {false, true},
		mqtt.LogLevelError:   {false, true},
	} {
		debug, errs := pahoLoggers(mqtt.NewClientOptions().SetLogLevel(level))

		_, debugNoop := debug.(log.NOOPLogger)
		_, errsNoop := errs.(log.NOOPLogger)

		assert.Equal(t, want, [2]bool{!debugNoop, !errsNoop}, "level %v", level)
	}
}
