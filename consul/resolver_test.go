package consul

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/gojek/courier-go"
)

func TestNewResolver(t *testing.T) {
	config := &Config{
		ConsulAddress: "localhost:8500",
		KVKey:         "test",
	}

	resolver, err := NewResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Stop()

	if resolver == nil {
		t.Fatal("Expected resolver to be created, got nil")
	}

	if resolver.UpdateChan() == nil {
		t.Fatal("Expected UpdateChan to be available")
	}

	if resolver.Done() == nil {
		t.Fatal("Expected Done channel to be available")
	}
}

func TestNewResolver_InvalidConfig(t *testing.T) {
	config := &Config{}
	_, err := NewResolver(config)
	if err == nil {
		t.Fatal("Expected error for invalid config, got nil")
	}
}

func TestNewResolver_FailedClient(t *testing.T) {
	config := &Config{
		ConsulAddress: "",
		KVKey:         "test",
	}
	_, err := NewResolver(config)
	if err == nil {
		t.Fatal("Expected error for failed client creation, got nil")
	}
}

func TestResolver_Stop(t *testing.T) {
	config := &Config{
		ConsulAddress: "localhost:8500",
		KVKey:         "test",
	}
	resolver, err := NewResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}

	resolver.Stop()

	select {
	case <-resolver.Done():
		// Success
	default:
		t.Fatal("Expected done channel to be closed")
	}
}

func TestResolver_UpdateServiceNameFromKV_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/kv/test" {
			data := consulapi.KVPair{
				Key:   "test",
				Value: []byte(`{"serviceName": "new-service"}`),
			}
			if err := json.NewEncoder(w).Encode([]consulapi.KVPair{data}); err != nil {
				t.Fatalf("failed to encode kv pair: %v", err)
			}
		}
	}))
	defer server.Close()

	config := &Config{
		ConsulAddress: server.Listener.Addr().String(),
		KVKey:         "test",
	}
	resolver, err := NewResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Stop()

	err = resolver.updateServiceNameFromKV()
	if err != nil {
		t.Fatalf("updateServiceNameFromKV failed: %v", err)
	}

	if resolver.serviceName != "new-service" {
		t.Errorf("Expected serviceName to be 'new-service', got '%s'", resolver.serviceName)
	}
}

func TestResolver_Discover_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		services := []*consulapi.ServiceEntry{
			{
				Node:    &consulapi.Node{Address: "127.0.0.1"},
				Service: &consulapi.AgentService{Port: 8080},
			},
		}
		if err := json.NewEncoder(w).Encode(services); err != nil {
			t.Fatalf("failed to encode services: %v", err)
		}
	}))
	defer server.Close()

	config := &Config{
		ConsulAddress: server.Listener.Addr().String(),
		KVKey:         "test",
	}
	resolver, err := NewResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Stop()

	go func() {
		err := resolver.discover()
		if err != nil {
			t.Errorf("discover failed: %v", err)
		}
	}()

	select {
	case addresses := <-resolver.UpdateChan():
		if len(addresses) != 1 {
			t.Fatalf("Expected 1 address, got %d", len(addresses))
		}
		if addresses[0].Host != "127.0.0.1" || addresses[0].Port != 8080 {
			t.Errorf("Unexpected address: %+v", addresses[0])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for addresses")
	}
}

func TestResolver_ConvertToTCPAddresses(t *testing.T) {
	resolver := &Resolver{}
	services := []*consulapi.ServiceEntry{
		{
			Node:    &consulapi.Node{Address: "127.0.0.1"},
			Service: &consulapi.AgentService{Service: "test", Address: "127.0.0.2", Port: 8080},
		},
		{
			Node:    &consulapi.Node{Address: "127.0.0.3"},
			Service: &consulapi.AgentService{Service: "test", Address: "", Port: 8081},
		},
	}

	addresses := resolver.convertToTCPAddresses(services)
	expected := []courier.TCPAddress{
		{Host: "127.0.0.2", Port: 8080},
		{Host: "127.0.0.3", Port: 8081},
	}

	if len(addresses) != len(expected) {
		t.Fatalf("Expected %d addresses, got %d", len(expected), len(addresses))
	}

	for i := range addresses {
		if addresses[i] != expected[i] {
			t.Errorf("Expected address %+v, got %+v", expected[i], addresses[i])
		}
	}
}

func TestResolver_UpdateServiceNameFromKV_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &Config{
		ConsulAddress: server.Listener.Addr().String(),
		KVKey:         "test",
	}
	resolver, err := NewResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Stop()

	err = resolver.updateServiceNameFromKV()
	if err == nil {
		t.Fatal("Expected an error but got nil")
	}
}

func TestResolver_Discover_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &Config{
		ConsulAddress: server.Listener.Addr().String(),
		KVKey:         "test",
	}
	resolver, err := NewResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Stop()

	err = resolver.discover()
	if err == nil {
		t.Fatal("Expected an error but got nil")
	}
}

func TestResolver_WatchServices(t *testing.T) {
	var callCount int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		services := []*consulapi.ServiceEntry{
			{
				Node:    &consulapi.Node{Address: "127.0.0.1"},
				Service: &consulapi.AgentService{Port: 8080},
			},
		}
		if err := json.NewEncoder(w).Encode(services); err != nil {
			t.Fatalf("failed to encode services: %v", err)
		}
	}))
	defer server.Close()

	config := &Config{
		ConsulAddress: server.Listener.Addr().String(),
		KVKey:         "test",
	}
	resolver, err := NewResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}

	go resolver.watchServices()
	<-resolver.UpdateChan()

	resolver.Stop()
}

func TestResolver_WatchKV(t *testing.T) {
	var callCount int
	var mu sync.Mutex
	kvHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var data consulapi.KVPair
		if callCount == 0 {
			data = consulapi.KVPair{
				Key:   "test",
				Value: []byte(`{"serviceName": "service-a"}`),
			}
		} else {
			data = consulapi.KVPair{
				Key:   "test",
				Value: []byte(`{"serviceName": "service-b"}`),
			}
		}
		callCount++
		if err := json.NewEncoder(w).Encode([]consulapi.KVPair{data}); err != nil {
			t.Fatalf("failed to encode kv pair: %v", err)
		}
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/kv/test" {
			kvHandler(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode([]*consulapi.ServiceEntry{}); err != nil {
			t.Fatalf("failed to encode services: %v", err)
		}
	}))
	defer server.Close()

	config := &Config{
		ConsulAddress: server.Listener.Addr().String(),
		KVKey:         "test",
	}
	resolver, err := NewResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	resolver.serviceName = "initial-service"

	go resolver.watchKV()

	time.Sleep(100 * time.Millisecond)

	resolver.mu.RLock()
	if resolver.serviceName != "service-b" {
		t.Errorf("Expected service name to be 'service-b', got '%s'", resolver.serviceName)
	}
	resolver.mu.RUnlock()

	resolver.Stop()
}

func TestAreAddressesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []courier.TCPAddress
		b        []courier.TCPAddress
		expected bool
	}{
		{
			name:     "both empty slices",
			a:        []courier.TCPAddress{},
			b:        []courier.TCPAddress{},
			expected: true,
		},
		{
			name:     "both nil slices",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "one empty one nil",
			a:        []courier.TCPAddress{},
			b:        nil,
			expected: true,
		},
		{
			name: "identical single address",
			a: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
			},
			b: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
			},
			expected: true,
		},
		{
			name: "identical multiple addresses same order",
			a: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.2", Port: 8081},
			},
			b: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.2", Port: 8081},
			},
			expected: true,
		},
		{
			name: "identical multiple addresses different order",
			a: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.2", Port: 8081},
			},
			b: []courier.TCPAddress{
				{Host: "127.0.0.2", Port: 8081},
				{Host: "127.0.0.1", Port: 8080},
			},
			expected: true,
		},
		{
			name: "different lengths",
			a: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
			},
			b: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.2", Port: 8081},
			},
			expected: false,
		},
		{
			name: "different hosts same port",
			a: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
			},
			b: []courier.TCPAddress{
				{Host: "127.0.0.2", Port: 8080},
			},
			expected: false,
		},
		{
			name: "same host different ports",
			a: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
			},
			b: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8081},
			},
			expected: false,
		},
		{
			name: "duplicate addresses same count",
			a: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.1", Port: 8080},
			},
			b: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.1", Port: 8080},
			},
			expected: true,
		},
		{
			name: "duplicate addresses different count",
			a: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.2", Port: 8081},
			},
			b: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.2", Port: 8081},
				{Host: "127.0.0.2", Port: 8081},
			},
			expected: false,
		},
		{
			name: "one empty one non-empty",
			a:    []courier.TCPAddress{},
			b: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
			},
			expected: false,
		},
		{
			name: "complex case with multiple duplicates",
			a: []courier.TCPAddress{
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.2", Port: 8081},
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.3", Port: 8082},
			},
			b: []courier.TCPAddress{
				{Host: "127.0.0.3", Port: 8082},
				{Host: "127.0.0.1", Port: 8080},
				{Host: "127.0.0.2", Port: 8081},
				{Host: "127.0.0.1", Port: 8080},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := areAddressesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("areAddressesEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestResolver_Discover_AreAddressesEqualScenario(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var services []*consulapi.ServiceEntry

		switch callCount {
		case 1:
			services = []*consulapi.ServiceEntry{
				{
					Node:    &consulapi.Node{Address: "127.0.0.1"},
					Service: &consulapi.AgentService{Port: 8080},
				},
			}
		case 2:
			services = []*consulapi.ServiceEntry{
				{
					Node:    &consulapi.Node{Address: "127.0.0.1"},
					Service: &consulapi.AgentService{Port: 8080},
				},
			}
		case 3:
			services = []*consulapi.ServiceEntry{
				{
					Node:    &consulapi.Node{Address: "127.0.0.1"},
					Service: &consulapi.AgentService{Port: 8080},
				},
				{
					Node:    &consulapi.Node{Address: "127.0.0.2"},
					Service: &consulapi.AgentService{Port: 8081},
				},
			}
		}

		if err := json.NewEncoder(w).Encode(services); err != nil {
			t.Fatalf("failed to encode services: %v", err)
		}
	}))
	defer server.Close()

	config := &Config{
		ConsulAddress: server.Listener.Addr().String(),
		KVKey:         "test",
	}
	resolver, err := NewResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Stop()

	resolver.serviceName = "test-service"

	go func() {
		if err := resolver.discover(); err != nil {
			t.Errorf("first discover failed: %v", err)
		}
	}()

	select {
	case addresses := <-resolver.UpdateChan():
		if len(addresses) != 1 {
			t.Fatalf("Expected 1 address in first update, got %d", len(addresses))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for first update")
	}

	go func() {
		if err := resolver.discover(); err != nil {
			t.Errorf("second discover failed: %v", err)
		}
	}()

	select {
	case <-resolver.UpdateChan():
		t.Fatal("Should not receive update when addresses are unchanged (areAddressesEqual returns true)")
	case <-time.After(100 * time.Millisecond):
	}

	go func() {
		if err := resolver.discover(); err != nil {
			t.Errorf("third discover failed: %v", err)
		}
	}()

	select {
	case addresses := <-resolver.UpdateChan():
		if len(addresses) != 2 {
			t.Fatalf("Expected 2 addresses in third update, got %d", len(addresses))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for third update")
	}
}
