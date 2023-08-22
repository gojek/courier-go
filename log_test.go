package courier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithLogger(t *testing.T) {
	c, err := NewClient(append(defOpts, WithLogger(defaultLogger))...)
	assert.NoError(t, err)

	assert.Equal(t, defaultLogger, c.options.logger)
}
