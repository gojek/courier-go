package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewPrometheus(t *testing.T) {
	reg := prometheus.NewRegistry()
	p := NewPrometheus()
	assert.NotNil(t, p)
	err := p.AddToRegistry(reg)
	assert.NoError(t, err)
	assert.NotNil(t, p.Publish())
	assert.NotNil(t, p.Subscribe())
	assert.NotNil(t, p.SubscribeMultiple())
	assert.NotNil(t, p.Unsubscribe())
	assert.NotNil(t, p.CallbackOp())
}

func TestNewPrometheus_AddToRegistry_error(t *testing.T) {
	reg := prometheus.NewRegistry()
	p := NewPrometheus()
	err := p.AddToRegistry(reg)
	assert.NoError(t, err)
	err = p.AddToRegistry(reg)
	assert.Error(t, err)
}
