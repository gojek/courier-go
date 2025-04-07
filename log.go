package courier

import (
	"context"
	"fmt"
)

// WithLogger sets the Logger to use for the client.
func WithLogger(l Logger) ClientOption { return optionFunc(func(o *clientOptions) { o.logger = l }) }

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
)

type pahoLogger struct {
	logger Logger
	level  logLevel
}

func (l *pahoLogger) Println(v ...interface{}) {
	switch l.level {
	case errorLevel:
		l.logger.Error(context.Background(), fmt.Errorf(fmt.Sprintln(v...)), nil)
	case warnLevel:
		l.logger.Warn(context.Background(), fmt.Sprintln(v...), nil)
	case debugLevel:
		l.logger.Debug(context.Background(), fmt.Sprintln(v...), nil)
	}
}

func (l *pahoLogger) Printf(format string, v ...interface{}) {
	switch l.level {
	case errorLevel:
		l.logger.Error(context.Background(), fmt.Errorf(format, v...), nil)
	case warnLevel:
		l.logger.Warn(context.Background(), fmt.Sprintf(format, v...), nil)
	case debugLevel:
		l.logger.Debug(context.Background(), fmt.Sprintf(format, v...), nil)
	}
}
