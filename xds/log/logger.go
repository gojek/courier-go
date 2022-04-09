package log

type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(err error, msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
}

type NoOpLogger struct{}

func (l *NoOpLogger) Info(_ string, _ ...interface{}) {}

func (l *NoOpLogger) Error(_ error, _ string, _ ...interface{}) {}

func (l *NoOpLogger) Debug(_ string, _ ...interface{}) {}
