package courier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWithLogger(t *testing.T) {
	c, err := NewClient(append(defOpts, WithLogger(defaultLogger))...)
	assert.NoError(t, err)

	assert.Equal(t, defaultLogger, c.options.logger)
}

type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Info(ctx context.Context, msg string, attrs map[string]any) {
	m.Called(ctx, msg, attrs)
}

func (m *mockLogger) Error(ctx context.Context, err error, attrs map[string]any) {
	m.Called(ctx, err, attrs)
}

func newMockLogger(t *testing.T) *mockLogger {
	m := &mockLogger{}
	m.Test(t)
	return m
}

func Test_noOpLogger(t *testing.T) {
	l := noOpLogger{}
	l.Info(context.Background(), "", nil)
	l.Error(context.Background(), nil, nil)
}

func TestWithPahoLogLevel(t *testing.T) {
	c, err := NewClient(append(defOpts, WithPahoLogLevel(LogLevelDefault))...)
	assert.NoError(t, err)
	assert.Equal(t, LogLevelDefault.toPahoLogLevel(), c.options.pahoLogLevel)

	c, err = NewClient(append(defOpts, WithPahoLogLevel(LogLevelDebug))...)
	assert.NoError(t, err)
	assert.Equal(t, LogLevelDebug.toPahoLogLevel(), c.options.pahoLogLevel)

	c, err = NewClient(append(defOpts, WithPahoLogLevel(LogLevelWarn))...)
	assert.NoError(t, err)
	assert.Equal(t, LogLevelWarn.toPahoLogLevel(), c.options.pahoLogLevel)

	c, err = NewClient(append(defOpts, WithPahoLogLevel(LogLevelError))...)
	assert.NoError(t, err)
	assert.Equal(t, LogLevelError.toPahoLogLevel(), c.options.pahoLogLevel)
}
