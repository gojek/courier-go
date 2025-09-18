package consul

import (
	"testing"
	"time"
)

func TestNewResolver(t *testing.T) {
	config := &Config{
		ConsulAddress: "localhost:8500",
		ServiceName:   "test-service",
		HealthyOnly:   true,
		WaitTime:      5 * time.Minute,
	}

	resolver, err := NewResolver(config)
	if err != nil {
		t.Skipf("Skipping test as Consul is not available: %v", err)
		return
	}
	defer resolver.Stop()

	if resolver == nil {
		t.Fatal("Expected resolver to be created, got nil")
	}

	updateChan := resolver.UpdateChan()
	if updateChan == nil {
		t.Fatal("Expected UpdateChan to be available")
	}

	doneChan := resolver.Done()
	if doneChan == nil {
		t.Fatal("Expected Done channel to be available")
	}
}

func TestNewResolverInvalidConfig(t *testing.T) {
	config := &Config{
		ConsulAddress: "localhost:8500",
		HealthyOnly:   true,
		WaitTime:      5 * time.Minute,
	}

	resolver, err := NewResolver(config)
	if err == nil {
		t.Fatal("Expected error for missing service name, got nil")
	}

	if resolver != nil {
		t.Fatal("Expected resolver to be nil on error")
	}
}

func TestResolverServiceDiscovery(t *testing.T) {
	config := &Config{
		ConsulAddress: "localhost:8500",
		ServiceName:   "consul",
		HealthyOnly:   true,
		WaitTime:      5 * time.Minute,
	}

	resolver, err := NewResolver(config)
	if err != nil {
		t.Skipf("Skipping test as Consul is not available: %v", err)
		return
	}
	defer resolver.Stop()

	select {
	case addresses := <-resolver.UpdateChan():
		t.Logf("Discovered %d service addresses", len(addresses))
		for i, addr := range addresses {
			t.Logf("  Address %d: %s:%d", i+1, addr.Host, addr.Port)
		}
	case <-time.After(10 * time.Second):
		t.Log("No service updates received within timeout - this might be expected if service doesn't exist")
	}
}
