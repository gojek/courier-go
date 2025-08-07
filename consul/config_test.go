package consul

import (
	"log"
	"testing"
	"time"

	consulapi "github.com/hashicorp/consul/api"
)

// TestDefaultConfig tests that DefaultConfig returns valid defaults
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.ConsulAddress != "localhost:8500" {
		t.Errorf("Expected ConsulAddress to be 'localhost:8500', got '%s'", config.ConsulAddress)
	}

	if !config.HealthyOnly {
		t.Error("Expected HealthyOnly to be true by default")
	}

	if config.WatchInterval != 30*time.Second {
		t.Errorf("Expected WatchInterval to be 30s, got %v", config.WatchInterval)
	}
}

// TestConfigValidate tests configuration validation
func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				ServiceName: "test-service",
			},
			expectErr: false,
		},
		{
			name: "missing service name",
			config: &Config{
				ConsulAddress: "localhost:8500",
			},
			expectErr: true,
		},
		{
			name: "empty service name",
			config: &Config{
				ServiceName: "",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestConfigWithAllOptions tests config with all options set
func TestConfigWithAllOptions(t *testing.T) {
	logger := log.New(log.Writer(), "[test] ", log.LstdFlags)
	tlsConfig := &consulapi.TLSConfig{
		Address: "consul.example.com",
	}

	config := &Config{
		ConsulAddress: "consul.example.com:8500",
		ConsulToken:   "test-token",
		DataCentre:    "dc1",
		ServiceName:   "test-service",
		Tags:          []string{"env:prod", "version:v1"},
		HealthyOnly:   true,
		WatchInterval: 60 * time.Second,
		TLSConfig:     tlsConfig,
		Logger:        logger,
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}

	// Verify all fields are set correctly
	if config.ConsulAddress != "consul.example.com:8500" {
		t.Errorf("ConsulAddress mismatch")
	}
	if config.ConsulToken != "test-token" {
		t.Errorf("ConsulToken mismatch")
	}
	if config.DataCentre != "dc1" {
		t.Errorf("DataCentre mismatch")
	}
	if config.ServiceName != "test-service" {
		t.Errorf("ServiceName mismatch")
	}
	if len(config.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(config.Tags))
	}
	if !config.HealthyOnly {
		t.Error("HealthyOnly should be true")
	}
	if config.WatchInterval != 60*time.Second {
		t.Errorf("WatchInterval mismatch")
	}
	if config.TLSConfig != tlsConfig {
		t.Error("TLSConfig mismatch")
	}
	if config.Logger != logger {
		t.Error("Logger mismatch")
	}
}
