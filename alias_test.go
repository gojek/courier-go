package courier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMemoryStore(t *testing.T) {
	s := NewMemoryStore()
	assert.NotNil(t, s)
}
