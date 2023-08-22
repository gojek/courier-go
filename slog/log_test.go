package slog

import (
	"bytes"
	"context"
	"github.com/gojek/courier-go"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"reflect"
	"testing"
)

func TestWithLogger(t *testing.T) {
	h := slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{AddSource: true})
	logger := New(h)

	c, err := courier.NewClient(courier.WithAddress("localhost", 1883), courier.WithLogger(logger))
	assert.NoError(t, err)

	// use reflection to read private field
	actualLogger := reflect.ValueOf(c).Elem().FieldByName("options").Elem().FieldByName("logger")

	assert.True(t, actualLogger.Equal(reflect.ValueOf(logger)))
}

func TestWithLoggerWrite(t *testing.T) {
	buf := &bytes.Buffer{}
	h := slog.NewTextHandler(buf, &slog.HandlerOptions{})
	logger := New(h)

	logger.Info(context.TODO(), "test", map[string]any{"key": "value"})
	logger.Error(context.TODO(), courier.ErrClientNotInitialized, map[string]any{"key": "value"})

	out := buf.String()
	assert.Contains(t, out, "level=INFO msg=test key=value")
	assert.Contains(t, out, "level=ERROR msg=\"courier: client not initialized\" key=value")
}
