package driver

import (
	"testing"

	"github.com/eclipse/paho.golang/paho/log"
	mqtt "github.com/gojek/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
)

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
