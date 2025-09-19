package consul

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gojek/courier-go"
	consulapi "github.com/hashicorp/consul/api"
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
	<-resolver.UpdateChan() // consume initial update
	<-resolver.UpdateChan() // consume second update

	resolver.Stop()

	mu.Lock()
	if callCount < 2 {
		t.Errorf("Expected at least 2 calls to discover, got %d", callCount)
	}
	mu.Unlock()
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
