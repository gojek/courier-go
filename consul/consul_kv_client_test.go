package consul

import (
	"testing"
	"time"
)

func TestSimpleGetValue(t *testing.T) {
	// Just get the value and print it
	err := PrintTestConsulValue()
	if err != nil {
		t.Logf("Error: %v", err)
		t.Log("Make sure Consul is running and has a value for key 'testConsul'")
		t.Log("You can set it with: consul kv put testConsul '{\"message\":\"hello world\"}'")
	}
}

// TestLiveServiceSwitching demonstrates live service discovery switching based on KV changes
//
// Prerequisites:
// 1. Consul running on localhost:8500
// 2. At least 2 services registered in Consul (e.g., "consul", "web")
// 3. Key "testConsul" exists with JSON: {"serviceName":"consul","message":"test"}
//
// Test Flow:
// 1. Shows current service IPs from KV
// 2. Waits for you to change serviceName in KV
// 3. Immediately shows new service IPs
func TestLiveServiceSwitching(t *testing.T) {
	// Get current service name from KV
	currentServiceName, err := GetServiceNameFromKV("testConsul")
	if err != nil {
		t.Skipf("Skipping test - could not get serviceName from testConsul: %v", err)
		t.Log("Setup: consul kv put testConsul '{\"serviceName\":\"consul\",\"message\":\"test\"}'")
		return
	}

	t.Logf("🎯 Starting live service discovery test...")
	t.Logf("📋 Current service from KV: '%s'", currentServiceName)

	// Create resolver with KV watching
	config := &Config{
		ConsulAddress: "localhost:8500",
		ServiceName:   currentServiceName,
		HealthyOnly:   false,
		WatchInterval: 5 * time.Minute, // Long interval to test immediate triggering
		KVKey:         "testConsul",
	}

	resolver, err := NewResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Stop()

	// Give resolver time to start
	time.Sleep(500 * time.Millisecond)
	updateChan := resolver.UpdateChan()

	t.Log("🚀 Waiting for initial service discovery...")

	// Step 1: Wait for initial discovery
	select {
	case addresses := <-updateChan:
		t.Logf("✅ Initial discovery for service '%s':", currentServiceName)
		if len(addresses) == 0 {
			t.Logf("   ❌ No instances found")
		} else {
			t.Logf("   ✅ Found %d instances:", len(addresses))
			for i, addr := range addresses {
				t.Logf("      %d. %s:%d", i+1, addr.Host, addr.Port)
			}
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for initial service discovery")
	}

	t.Log("\n💡 MANUAL ACTION REQUIRED:")
	t.Log("   Change the service name in KV store:")
	t.Log("   consul kv put testConsul '{\"serviceName\":\"web\",\"message\":\"test\"}'")
	t.Log("   (Waiting up to 35 seconds for immediate response...)")

	// Step 2: Wait for KV change to trigger discovery
	startTime := time.Now()
	select {
	case addresses := <-updateChan:
		responseTime := time.Since(startTime)
		t.Logf("\n🔄 SERVICE SWITCH DETECTED!")
		t.Logf("⚡ Response time: %v", responseTime)

		if len(addresses) == 0 {
			t.Logf("   ❌ No instances found for new service")
		} else {
			t.Logf("   ✅ Found %d instances for new service:", len(addresses))
			for i, addr := range addresses {
				t.Logf("      %d. %s:%d", i+1, addr.Host, addr.Port)
			}
		}

		if responseTime < 35*time.Second {
			t.Logf("🎉 SUCCESS: Immediate trigger in %v (much faster than 60s watch interval)", responseTime)
		} else {
			t.Errorf("❌ Too slow: %v (expected < 35s)", responseTime)
		}

	case <-time.After(35 * time.Second):
		t.Error("❌ TIMEOUT: No service discovery update received")
		t.Error("   Check if you changed the KV serviceName value")
	}
}
