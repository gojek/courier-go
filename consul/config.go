package consul

import (
	"fmt"
	"log"
	"time"
)

type Config struct {
	ConsulAddress string
	ServiceName   string
	HealthyOnly   bool
	KVKey         string
	WatchInterval time.Duration
	Logger        *log.Logger
}

func DefaultConfig() *Config {
	return &Config{
		ConsulAddress: "localhost:8500",
		HealthyOnly:   true,
		WatchInterval: 5 * time.Minute,
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
