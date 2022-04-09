package log

import (
	"errors"
	"testing"
)

func TestNoOpLogger(t *testing.T) {
	l := NoOpLogger{}
	l.Error(errors.New("some error"), "some message")
	l.Info("some message")
	l.Debug("some message")
}
