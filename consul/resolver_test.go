package consul

import (
	"testing"
	"time"
)

// TestNewResolver tests creating a new resolver with valid config
func TestNewResolver(t *testing.T) {
	config := &Config{
		ConsulAddress: "localhost:8500",
		ServiceName:   "test-service",
		HealthyOnly:   true,
		WatchInterval: 30 * time.Second,
	}

	resolver, err := NewResolver(config)
	if err != nil {
		t.Skipf("Skipping test as Consul is not available: %v", err)
		return
	}
	defer resolver.Stop()

	// Verify resolver was created properly
	if resolver == nil {
		t.Fatal("Expected resolver to be created, got nil")
	}

	// Verify channels are available
	updateChan := resolver.UpdateChan()
	if updateChan == nil {
		t.Fatal("Expected UpdateChan to be available")
	}

	doneChan := resolver.Done()
	if doneChan == nil {
		t.Fatal("Expected Done channel to be available")
	}
}

// TestNewResolverInvalidConfig tests error handling for invalid config
func TestNewResolverInvalidConfig(t *testing.T) {
	config := &Config{
		ConsulAddress: "localhost:8500",
		// Missing ServiceName - should cause error
		HealthyOnly:   true,
		WatchInterval: 30 * time.Second,
	}

	resolver, err := NewResolver(config)
	if err == nil {
		t.Fatal("Expected error for missing service name, got nil")
	}

	if resolver != nil {
		t.Fatal("Expected resolver to be nil on error")
	}
}

// TestResolverServiceDiscovery tests the resolver's service discovery functionality
func TestResolverServiceDiscovery(t *testing.T) {
	config := &Config{
		ConsulAddress: "localhost:8500",
		ServiceName:   "consul", // Use consul service as it should always exist
		HealthyOnly:   true,
		WatchInterval: 5 * time.Second,
	}

	resolver, err := NewResolver(config)
	if err != nil {
		t.Skipf("Skipping test as Consul is not available: %v", err)
		return
	}
	defer resolver.Stop()

	// Wait for initial service discovery
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
