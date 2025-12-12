package consul

import (
	"fmt"
	"log"
	"time"

	"github.com/gojek/courier-go/otelcourier"
)

const DefaultDebounceDuration = 5 * time.Second

type Config struct {
	ConsulAddress    string
	HealthyOnly      bool
	KVKey            string
	WaitTime         time.Duration
	Logger           *log.Logger
	OTel             *otelcourier.OTel
	DebounceDuration time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		ConsulAddress:    "localhost:8500",
		HealthyOnly:      true,
		WaitTime:         5 * time.Minute,
		DebounceDuration: DefaultDebounceDuration,
	}
}

func (c *Config) applyDefaults() {
	if c.DebounceDuration == 0 {
		c.DebounceDuration = DefaultDebounceDuration
	}
}

func (c *Config) Validate() error {
	if c.KVKey == "" {
		return fmt.Errorf("consul: KV key is required")
	}

	if c.ConsulAddress == "" {
		return fmt.Errorf("consul: Consul address is required")
	}

	return nil
}
