package courier

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopMiddleware(t *testing.T) {
	c, err := NewClient(defOpts...)
	require.NoError(t, err)

	var middlewareCalled bool
	var mu sync.Mutex

	middleware := func(next Stopper) Stopper {
		return StopHandlerFunc(func() error {
			mu.Lock()
			middlewareCalled = true
			mu.Unlock()
			return next.Stop()
		})
	}

	c.UseStopMiddleware(middleware)

	assert.NotNil(t, c.stopMiddleware)
	assert.NotNil(t, c.stopHandler)

	c.Stop()

	mu.Lock()
	assert.True(t, middlewareCalled, "Middleware should have been called")
	mu.Unlock()
}

func TestStopHandlerFunc(t *testing.T) {
	called := false
	expectedError := errors.New("test error")

	handler := StopHandlerFunc(func() error {
		called = true
		return expectedError
	})

	err := handler.Stop()
	assert.True(t, called, "Function should have been called")
	assert.Equal(t, expectedError, err, "Should return expected error")
}

func TestStopMiddlewareFunc(t *testing.T) {
	nextCalled := false
	nextHandler := StopHandlerFunc(func() error {
		nextCalled = true
		return nil
	})

	middlewareCalled := false
	middleware := StopMiddlewareFunc(func(next Stopper) Stopper {
		middlewareCalled = true
		return next
	})

	wrappedHandler := middleware.Middleware(nextHandler)
	assert.True(t, middlewareCalled, "Middleware function should have been called")

	err := wrappedHandler.Stop()
	assert.NoError(t, err)
	assert.True(t, nextCalled, "Next handler should have been called")
}
