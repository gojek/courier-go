// Package consul provides a Consul-based service discovery resolver for courier-go.
package consul

import (
	"fmt"
	"log"
	"time"

	consulapi "github.com/hashicorp/consul/api"
)

// Config holds configuration for Consul resolver
type Config struct {
	// Consul client configuration
	ConsulAddress string // Address of the Consul server
	ConsulToken   string // ACL token for Consul (optional)
	DataCentre    string // Data centre to use for service discovery (optional)

	// Service discovery configuration
	ServiceName string   // Name of the service to discover
	Tags        []string // Tags to filter services (optional)
	HealthyOnly bool     // Only return healthy services

	// KV watching (optional)
	KVKey string // If set, watch this KV key for service name changes

	// Watch configuration
	WatchInterval time.Duration // Maximum time Consul waits for changes before returning (blocking query timeout)

	// TLS configuration
	TLSConfig *consulapi.TLSConfig // Optional TLS configuration for secure connections

	// Logging configuration
	Logger *log.Logger // Optional logger for error and debug messages (uses default if nil)
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ConsulAddress: "localhost:8500",
		HealthyOnly:   true,
		WatchInterval: 30 * time.Second,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("consul: service name is required")
	}

	return nil
}
