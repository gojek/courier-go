package courier

import (
	"context"
	"fmt"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var pahoLogInit sync.Once

// WithLogger sets the Logger to use for the client.
func WithLogger(l Logger) ClientOption {
	pahoLogInit.Do(func() {
		initPahoLogging(l)
	})

	return optionFunc(func(o *clientOptions) { o.logger = l })
}

// Logger is the interface that wraps the Info and Error methods.
type Logger interface {
	Error(ctx context.Context, err error, attrs map[string]any)
	Warn(ctx context.Context, msg string, attrs map[string]any)
	Info(ctx context.Context, msg string, attrs map[string]any)
	Debug(ctx context.Context, msg string, attrs map[string]any)
}

var defaultLogger Logger = noOpLogger{}

type noOpLogger struct{}

func (noOpLogger) Error(context.Context, error, map[string]any)  {}
func (noOpLogger) Warn(context.Context, string, map[string]any)  {}
func (noOpLogger) Info(context.Context, string, map[string]any)  {}
func (noOpLogger) Debug(context.Context, string, map[string]any) {}

type logLevel int

const (
	debugLevel logLevel = iota
	warnLevel
	errorLevel
	criticalLevel
)

type pahoLogger struct {
	logger Logger
	level  logLevel
}

func (l pahoLogger) log(msg string, err error) {
	switch l.level {
	case errorLevel, criticalLevel:
		l.logger.Error(context.Background(), err, nil)
	case warnLevel:
		l.logger.Warn(context.Background(), msg, nil)
	case debugLevel:
		l.logger.Debug(context.Background(), msg, nil)
	}
}

func (l pahoLogger) Println(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	l.log(msg, fmt.Errorf(msg))
}

func (l pahoLogger) Printf(format string, v ...interface{}) {
	l.log(fmt.Sprintf(format, v...), fmt.Errorf(format, v...))
}

func initPahoLogging(l Logger) {
	mqtt.DEBUG = pahoLogger{logger: l, level: debugLevel}
	mqtt.WARN = pahoLogger{logger: l, level: warnLevel}
	mqtt.ERROR = pahoLogger{logger: l, level: errorLevel}
	mqtt.CRITICAL = pahoLogger{logger: l, level: criticalLevel}
}
