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

func (m *mockLogger) Warn(ctx context.Context, msg string, attrs map[string]any) {
	m.Called(ctx, msg, attrs)
}

func (m *mockLogger) Debug(ctx context.Context, msg string, attrs map[string]any) {
	m.Called(ctx, msg, attrs)
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
	l.Warn(context.Background(), "", nil)
	l.Debug(context.Background(), "", nil)
}

func Test_pahoLogger_Println(t *testing.T) {
	tests := []struct {
		name     string
		level    logLevel
		mockFunc func(l *mockLogger)
	}{
		{
			name:  "CriticalLevel",
			level: criticalLevel,
			mockFunc: func(l *mockLogger) {
				l.On("Error", mock.Anything, mock.Anything, mock.Anything).Return()
			},
		},
		{
			name:  "ErrorLevel",
			level: errorLevel,
			mockFunc: func(l *mockLogger) {
				l.On("Error", mock.Anything, mock.Anything, mock.Anything).Return()
			},
		},
		{
			name:  "WarnLevel",
			level: warnLevel,
			mockFunc: func(l *mockLogger) {
				l.On("Warn", mock.Anything, mock.Anything, mock.Anything).Return()
			},
		},
		{
			name:  "DebugLevel",
			level: debugLevel,
			mockFunc: func(l *mockLogger) {
				l.On("Debug", mock.Anything, mock.Anything, mock.Anything).Return()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newMockLogger(t)
			pl := &pahoLogger{
				logger: l,
				level:  tt.level,
			}

			tt.mockFunc(l)
			pl.Println("test")
			l.AssertExpectations(t)
		})
	}
}

func Test_pahoLogger_Printf(t *testing.T) {
	tests := []struct {
		name     string
		level    logLevel
		mockFunc func(l *mockLogger)
		format   string
		args     []interface{}
	}{
		{
			name:  "CriticalLevel",
			level: criticalLevel,
			mockFunc: func(l *mockLogger) {
				l.On("Error", mock.Anything, mock.MatchedBy(func(err error) bool {
					return err.Error() == "test error"
				}), mock.Anything).Return()
			},
			format: "test %s",
			args:   []interface{}{"error"},
		},
		{
			name:  "ErrorLevel",
			level: errorLevel,
			mockFunc: func(l *mockLogger) {
				l.On("Error", mock.Anything, mock.MatchedBy(func(err error) bool {
					return err.Error() == "test error"
				}), mock.Anything).Return()
			},
			format: "test %s",
			args:   []interface{}{"error"},
		},
		{
			name:  "WarnLevel",
			level: warnLevel,
			mockFunc: func(l *mockLogger) {
				l.On("Warn", mock.Anything, "test warn", mock.Anything).Return()
			},
			format: "test %s",
			args:   []interface{}{"warn"},
		},
		{
			name:  "DebugLevel",
			level: debugLevel,
			mockFunc: func(l *mockLogger) {
				l.On("Debug", mock.Anything, "test debug", mock.Anything).Return()
			},
			format: "test %s",
			args:   []interface{}{"debug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newMockLogger(t)
			pl := &pahoLogger{
				logger: l,
				level:  tt.level,
			}

			tt.mockFunc(l)
			pl.Printf(tt.format, tt.args...)
			l.AssertExpectations(t)
		})
	}
}
