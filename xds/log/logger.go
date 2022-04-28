package log

// Logger is used to log messages of info, error and debug levels
type Logger interface {
	// Info is used to log info level log messages
	Info(msg string, keysAndValues ...interface{})
	// Error is used to log error level log messages
	Error(err error, msg string, keysAndValues ...interface{})
	// Debug is used to log debug level log messages
	Debug(msg string, keysAndValues ...interface{})
}

// NoOpLogger is a noop implementation of Logger
type NoOpLogger struct{}

// Info logs info level log messages
func (l *NoOpLogger) Info(_ string, _ ...interface{}) {}

// Error logs error level log messages
func (l *NoOpLogger) Error(_ error, _ string, _ ...interface{}) {}

// Debug logs debug level log messages
func (l *NoOpLogger) Debug(_ string, _ ...interface{}) {}
