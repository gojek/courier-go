package courier

import (
	"context"

	mqtt "github.com/gojek/paho.mqtt.golang"
)

// WithLogger sets the Logger to use for the client.
func WithLogger(l Logger) ClientOption { return optionFunc(func(o *clientOptions) { o.logger = l }) }

// WithPahoLogLevel sets the log level for the underlying Paho MQTT client.
func WithPahoLogLevel(level LogLevel) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.pahoLogLevel = level.toPahoLogLevel()
	})
}

// Logger is the interface that wraps the Info and Error methods.
type Logger interface {
	Info(ctx context.Context, msg string, attrs map[string]any)
	Error(ctx context.Context, err error, attrs map[string]any)
}

var defaultLogger Logger = noOpLogger{}

type noOpLogger struct{}

func (noOpLogger) Info(context.Context, string, map[string]any) {}
func (noOpLogger) Error(context.Context, error, map[string]any) {}

type LogLevel int

const (
	LogLevelDefault LogLevel = iota // LogLevelDefault disables all log output
	LogLevelDebug
	LogLevelWarn
	LogLevelError
)

var defaultPahoLogLevel mqtt.LogLevel = mqtt.LogLevelDefault

func (l LogLevel) toPahoLogLevel() mqtt.LogLevel {
	switch l {
	case LogLevelDefault:
		return mqtt.LogLevelDefault
	case LogLevelDebug:
		return mqtt.LogLevelDebug
	case LogLevelWarn:
		return mqtt.LogLevelWarn
	case LogLevelError:
		return mqtt.LogLevelError
	default:
		return mqtt.LogLevelDefault
	}
}
