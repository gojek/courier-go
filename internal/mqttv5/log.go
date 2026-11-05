package mqttv5

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/eclipse/paho.golang/paho/log"
	mqtt "github.com/gojek/paho.mqtt.golang"
)

// slogPrinter adapts a slog.Logger to the paho.golang log.Logger interface.
type slogPrinter struct {
	l     *slog.Logger
	level slog.Level
}

func (s slogPrinter) Println(v ...interface{}) {
	s.l.Log(context.Background(), s.level, strings.TrimSpace(fmt.Sprintln(v...)))
}

func (s slogPrinter) Printf(format string, v ...interface{}) {
	s.l.Log(context.Background(), s.level, strings.TrimSpace(fmt.Sprintf(format, v...)))
}

// pahoLoggers returns the debug and error loggers for the v5 library.
// Same as the v3 library: LogLevelDefault is silent, Debug logs everything, Warn and Error log errors only.
func pahoLoggers(o *mqtt.ClientOptions) (debug, errs log.Logger) {
	if o.LogVerbosity == mqtt.LogLevelDefault {
		return log.NOOPLogger{}, log.NOOPLogger{}
	}

	h := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})).
		With(slog.String("clientID", o.ClientID))
	errs = slogPrinter{l: h, level: slog.LevelError}

	if o.LogVerbosity == mqtt.LogLevelDebug {
		return slogPrinter{l: h, level: slog.LevelDebug}, errs
	}

	return log.NOOPLogger{}, errs
}
