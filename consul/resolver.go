// Package consul provides a Consul-based service discovery resolver for courier-go.
package consul

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/gojek/courier-go"
)

// Resolver implements courier.Resolver interface using Consul for service discovery.
type Resolver struct {
	client      *consulapi.Client //The actual Consul API client that communicates with Consul server
	serviceName string            // The name of the service to discover
	dataCentre  string            // The data centre to use for service discovery
	tags        []string          // Tags to filter services
	logger      *log.Logger       // Logger for error and debug messages

	updateChan chan []courier.TCPAddress // Channel to send updates on service addresses
	doneChan   chan struct{}             // Channel to signal when the resolver is done

	// Configuration
	watchInterval time.Duration // Maximum time Consul waits for changes (blocking query timeout)
	healthyOnly   bool          // Only return healthy services &  Prevents connections to failing brokers

	// State management for consul
	mu        sync.RWMutex // Mutex to protect state changes
	lastIndex uint64       // Last Consul index used for long polling
	isRunning bool         // Whether the resolver is currently running
	stopOnce  sync.Once    // Ensures Stop() can only be called once

	// KV watching (optional)
	kvKey       string        // If set, watch this KV key for service name changes
	triggerChan chan struct{} // Channel to trigger immediate service discovery
}

func NewResolver(config *Config) (*Resolver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = config.ConsulAddress

	if config.ConsulToken != "" {
		consulConfig.Token = config.ConsulToken
	}

	if config.DataCentre != "" {
		consulConfig.Datacenter = config.DataCentre
	}

	if config.TLSConfig != nil {
		consulConfig.TLSConfig = *config.TLSConfig
	}

	client, err := consulapi.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("consul: failed to create client: %w", err)
	}

	// Use provided logger or create a default one
	logger := config.Logger
	if logger == nil {
		logger = log.New(log.Writer(), "[consul-resolver] ", log.LstdFlags)
	}

	resolver := &Resolver{
		client:        client,
		serviceName:   config.ServiceName,
		dataCentre:    config.DataCentre,
		tags:          config.Tags,
		healthyOnly:   config.HealthyOnly,
		watchInterval: config.WatchInterval,
		logger:        logger,
		updateChan:    make(chan []courier.TCPAddress, 1),
		doneChan:      make(chan struct{}),
		kvKey:         config.KVKey,           // Add KV key if provided
		triggerChan:   make(chan struct{}, 1), // Buffered to prevent blocking
	}

	// Start watching for service changes (existing goroutine)
	go resolver.watch()

	// Start KV watcher if KV key is provided (new goroutine)
	if resolver.kvKey != "" {
		go resolver.watchKV()
	}

	return resolver, nil
}

// UpdateChan returns a channel where TCPAddress updates can be received.
func (r *Resolver) UpdateChan() <-chan []courier.TCPAddress {
	return r.updateChan
}

// Done returns a channel which is closed when the Resolver is no longer running.
func (r *Resolver) Done() <-chan struct{} {
	return r.doneChan
}

// Stop gracefully stops the resolver and releases resources
func (r *Resolver) Stop() {
	r.stopOnce.Do(func() { // Ensure Stop() can only be called once
		r.mu.Lock()         // Lock to safely change state
		defer r.mu.Unlock() // Unlock after changing state
		//Even though sync.Once prevents multiple executions,
		// we still need the mutex to protect shared state that other
		//  goroutines might be accessing.
		if r.isRunning {
			r.isRunning = false
			close(r.doneChan)
			close(r.updateChan)
			close(r.triggerChan)
		}
	})
}

// watch continuously monitors Consul for service changes using long polling
func (r *Resolver) watch() {
	r.mu.Lock()
	r.isRunning = true
	r.mu.Unlock()

	// Initial discovery
	// Users expect immediate broker discovery
	if err := r.discoverServices(); err != nil {
		r.logger.Printf("Initial service discovery failed: %v (continuing to watch)", err)
	}

	for {
		select {
		case <-r.doneChan: // Stop watching if done
			return
		case <-r.triggerChan: // Immediate discovery triggered by service name change
			// Note: lastIndex is already reset to 0 in watchKV() when service name changes
			if err := r.discoverServices(); err != nil {
				r.logger.Printf("Triggered service discovery failed: %v", err)
			}
		default:
			// Use long polling - discoverServices will block until changes or timeout
			if err := r.discoverServices(); err != nil {
				r.logger.Printf("Service discovery failed: %v (retrying in 5 seconds)", err)
				// On error, wait briefly before retrying to avoid tight loop
				select {
				case <-r.doneChan:
					return
				case <-time.After(time.Second * 5):
				}
			}
		}
	}
}

// discoverServices queries Consul for service instances and sends updates
func (r *Resolver) discoverServices() error {
	queryOpts := &consulapi.QueryOptions{
		WaitIndex: r.lastIndex,     // Enable blocking queries for efficient change detection
		WaitTime:  r.watchInterval, // How long Consul should wait for changes
	}
	if r.dataCentre != "" {
		queryOpts.Datacenter = r.dataCentre
	}

	// Read serviceName with proper locking to avoid race conditions
	r.mu.RLock()
	currentServiceName := r.serviceName
	r.mu.RUnlock()

	//Prepare variables for Consul API response
	var services []*consulapi.ServiceEntry

	var meta *consulapi.QueryMeta

	var err error

	//Query Consul for services with or without health filtering
	if r.healthyOnly {
		services, meta, err = r.client.Health().Service(currentServiceName, "", true, queryOpts) // ONLY HEALTHY RETURN
	} else {
		services, meta, err = r.client.Health().Service(currentServiceName, "", false, queryOpts)
	}

	if err != nil {
		return fmt.Errorf("consul: failed to query services: %w", err)
	}

	//Store Consul index for next long-polling request
	r.lastIndex = meta.LastIndex

	//Apply client-side tag filtering
	if len(r.tags) > 0 {
		services = r.filterByTags(services)
	}

	//Convert Consul format to courier-go format [extract port and host]
	addresses := r.convertToTCPAddresses(services)

	r.logger.Printf("Discovered %d service instances for service '%s'", len(addresses), currentServiceName)

	// Send update to the existing courier resolver flow
	select {
	case r.updateChan <- addresses:
	case <-r.doneChan:
		return nil
	default:
	}

	return nil
}

// filterByTags filters services by required tags
func (r *Resolver) filterByTags(services []*consulapi.ServiceEntry) []*consulapi.ServiceEntry {
	if len(r.tags) == 0 {
		return services
	}

	var filtered []*consulapi.ServiceEntry
	//Keep only services that have ALL required tag
	for _, service := range services {
		if r.hasAllTags(service.Service.Tags) {
			filtered = append(filtered, service)
		}
	}

	return filtered
}

// hasAllTags checks if service has all required tags
func (r *Resolver) hasAllTags(serviceTags []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range serviceTags {
		tagSet[tag] = true
	}

	for _, requiredTag := range r.tags {
		if !tagSet[requiredTag] {
			return false
		}
	}

	return true
}

// convertToTCPAddresses converts Consul service entries to courier.TCPAddress
func (r *Resolver) convertToTCPAddresses(services []*consulapi.ServiceEntry) []courier.TCPAddress {
	addresses := make([]courier.TCPAddress, 0, len(services))

	for _, service := range services {
		address := courier.TCPAddress{
			Host: service.Service.Address,
			Port: uint16(service.Service.Port),
		}

		// Use node address if service address is empty
		if address.Host == "" {
			address.Host = service.Node.Address
		}

		addresses = append(addresses, address)
	}

	return addresses
}

// watchKV monitors the KV key for service name changes (simple goroutine 2)
func (r *Resolver) watchKV() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.doneChan:
			return
		case <-ticker.C: // Get current service name from KV
			pair, _, err := r.client.KV().Get(r.kvKey, nil)
			if err != nil {
				r.logger.Printf("KV watch error: %v", err)

				continue
			}

			if pair == nil {
				continue
			}
			// Parse serviceName from JSON
			var kvData map[string]interface{}
			if err := json.Unmarshal(pair.Value, &kvData); err != nil {
				r.logger.Printf("KV parse error: %v", err)

				continue
			}

			serviceName, ok := kvData["serviceName"].(string)
			if !ok {
				continue
			}

			// Update service name if changed
			r.mu.Lock()
			if serviceName != r.serviceName {
				r.logger.Printf("Service name changed from '%s' to '%s'", r.serviceName, serviceName)
				r.serviceName = serviceName
				r.lastIndex = 0 // Reset index since we're switching to a different service with different indices
				r.mu.Unlock()

				// Trigger immediate discovery
				select {
				case r.triggerChan <- struct{}{}:
				default:
				}
			} else {
				r.mu.Unlock()
			}
		}
	}
}
