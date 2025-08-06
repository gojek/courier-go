// Package consul provides a Consul-based service discovery resolver for courier-go.
package consul

import (
	"fmt"
	"sync"
	"time"

	"github.com/gojek/courier-go"
	consulapi "github.com/hashicorp/consul/api"
)

// Resolver implements courier.Resolver interface using Consul for service discovery.
type Resolver struct {
	client      *consulapi.Client //The actual Consul API client that communicates with Consul server
	serviceName string            // The name of the service to discover
	dataCentre  string            // The data centre to use for service discovery
	tags        []string          // Tags to filter services

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
}

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

	// Watch configuration
	WatchInterval time.Duration // Maximum time Consul waits for changes before returning (blocking query timeout)

	// TLS configuration
	TLSConfig *consulapi.TLSConfig // Optional TLS configuration for secure connections
}

func NewResolver(config *Config) (*Resolver, error) {
	if config.ServiceName == "" {
		return nil, fmt.Errorf("consul: service name is required")
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

	resolver := &Resolver{
		client:        client,
		serviceName:   config.ServiceName,
		dataCentre:    config.DataCentre,
		tags:          config.Tags,
		healthyOnly:   config.HealthyOnly,
		watchInterval: config.WatchInterval,
		updateChan:    make(chan []courier.TCPAddress, 1),
		doneChan:      make(chan struct{}),
	}

	// Start watching for service changes
	go resolver.watch()

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
		// Log error but continue watching
		// TODO: Add proper logging
	}

	for {
		select {
		case <-r.doneChan: // Stop watching if done
			return
		default:
			// Use long polling - discoverServices will block until changes or timeout
			if err := r.discoverServices(); err != nil {
				// On error, wait briefly before retrying to avoid tight loop
				select {
				case <-r.doneChan:
					return
				case <-time.After(time.Second * 5):
					// Continue after brief pause
				}
			}
			// No additional waiting needed - Consul's blocking query handles the timing
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

	//Prepare variables for Consul API response
	var services []*consulapi.ServiceEntry

	var meta *consulapi.QueryMeta

	var err error

	//Query Consul for services with or without health filtering
	if r.healthyOnly {
		services, meta, err = r.client.Health().Service(r.serviceName, "", true, queryOpts) // ONLY HEALTHY RETURN
	} else {
		services, meta, err = r.client.Health().Service(r.serviceName, "", false, queryOpts)
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

	// Send update to the existing courier resolver flow
	select {
	case r.updateChan <- addresses:
	case <-r.doneChan:
		return nil
	default:
		// Channel is full, skip this update
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
