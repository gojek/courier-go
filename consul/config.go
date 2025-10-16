package consul

import (
	"fmt"
	"log"
	"time"

	"github.com/gojek/courier-go/otelcourier"
)

type Config struct {
	ConsulAddress string
	HealthyOnly   bool
	KVKey         string
	WaitTime      time.Duration
	Logger        *log.Logger
	OTel          *otelcourier.OTel
}

func DefaultConfig() *Config {
	return &Config{
		ConsulAddress: "localhost:8500",
		HealthyOnly:   true,
		WaitTime:      5 * time.Minute,
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
