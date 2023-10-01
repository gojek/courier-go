package courier

import "context"

// WithLogger sets the Logger to use for the client.
func WithLogger(l Logger) ClientOption { return optionFunc(func(o *clientOptions) { o.logger = l }) }

// Logger is the interface that wraps the Info and Error methods.
type Logger interface {
	Info(ctx context.Context, msg string, attrs map[string]any)
	Error(ctx context.Context, err error, attrs map[string]any)
}

var defaultLogger Logger = noOpLogger{}

type noOpLogger struct{}

func (noOpLogger) Info(context.Context, string, map[string]any) {}
func (noOpLogger) Error(context.Context, error, map[string]any) {}
