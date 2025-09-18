package consul

import (
	"log"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.ConsulAddress != "localhost:8500" {
		t.Errorf("Expected ConsulAddress to be 'localhost:8500', got '%s'", config.ConsulAddress)
	}

	if !config.HealthyOnly {
		t.Error("Expected HealthyOnly to be true by default")
	}

	if config.WaitTime != 5*time.Minute {
		t.Errorf("Expected WaitTime to be 5m, got %v", config.WaitTime)
	}
}

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
			expectErr: true,
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

func TestConfigWithAllOptions(t *testing.T) {
	logger := log.New(log.Writer(), "[test] ", log.LstdFlags)
	config := &Config{
		ConsulAddress: "consul.example.com:8500",
		ServiceName:   "test-service",
		KVKey:         "testConsul",
		HealthyOnly:   true,
		WaitTime:      60 * time.Second,
		Logger:        logger,
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}

	if config.ConsulAddress != "consul.example.com:8500" {
		t.Errorf("ConsulAddress mismatch")
	}
	if config.KVKey != "testConsul" {
		t.Errorf("Key value mismatch")
	}
	if config.ServiceName != "test-service" {
		t.Errorf("ServiceName mismatch")
	}
	if !config.HealthyOnly {
		t.Error("HealthyOnly should be true")
	}
	if config.WaitTime != 60*time.Second {
		t.Errorf("WaitTime mismatch")
	}
	if config.Logger != logger {
		t.Error("Logger mismatch")
	}
}
