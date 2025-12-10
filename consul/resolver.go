// Package consul
package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/gojek/courier-go"
	"github.com/gojek/courier-go/otelcourier"
)

const (
	attrServiceName       = "service.name"
	attrConsulServiceName = "consul.service.name"
	attrSuccess           = "success"
	attrErrorType         = "error.type"

	errorTypeKVRead  = "kv_read_error"
	errorTypeKVParse = "kv_parse_error"

	attrAPIType = "api.type"

	apiTypeHealthService = "health_service"
	apiTypeKV            = "kv"

	attrKVKey = "kv.key"
)

type Resolver struct {
	client      *consulapi.Client
	serviceName string
	logger      *log.Logger

	updateChan chan []courier.TCPAddress
	doneChan   chan struct{}

	// Configuration
	waitTime    time.Duration
	healthyOnly bool

	// State management
	mu        sync.RWMutex
	lastIndex uint64

	// KV watching
	kvKey         string
	lastAddresses []courier.TCPAddress

	// Debounce
	debounceDuration time.Duration
	debounceTimer    *time.Timer
	pendingAddresses []courier.TCPAddress

	commonAttrs []attribute.KeyValue

	serviceInstances         metric.Int64ObservableGauge
	serviceInstancesCallback metric.Registration
	lastInstanceCount        int64

	consulAPIRequests metric.Int64Counter
	consulAPIDuration metric.Float64Histogram

	kvReadErrors   metric.Int64Counter
	kvValueChanges metric.Int64Counter

	addressUpdates metric.Int64Counter
}

func NewResolver(config *Config) (*Resolver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = config.ConsulAddress

	client, err := consulapi.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("consul: failed to create client: %w", err)
	}

	logger := config.Logger
	if logger == nil {
		logger = log.New(log.Writer(), "[consul-resolver] ", log.LstdFlags)
	}

	r := &Resolver{
		client:           client,
		serviceName:      "",
		healthyOnly:      config.HealthyOnly,
		waitTime:         config.WaitTime,
		logger:           logger,
		updateChan:       make(chan []courier.TCPAddress, 1),
		doneChan:         make(chan struct{}),
		kvKey:            config.KVKey,
		debounceDuration: config.DebounceDuration,
	}

	if config.OTel != nil {
		r.commonAttrs = []attribute.KeyValue{
			attribute.String(attrServiceName, config.OTel.Service()),
		}
		if err := r.initMetrics(config.OTel); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (r *Resolver) initMetrics(otel *otelcourier.OTel) error {
	meter := otel.Meter()

	var err error

	r.serviceInstances, err = meter.Int64ObservableGauge(
		"courier.consul.service_instances",
		metric.WithDescription("Current number of discovered healthy service instances"),
		metric.WithUnit("{instance}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create service_instances metric: %w", err)
	}

	r.serviceInstancesCallback, err = meter.RegisterCallback(
		r.observeInstanceCount,
		r.serviceInstances,
	)
	if err != nil {
		return fmt.Errorf("failed to register service_instances callback: %w", err)
	}

	r.consulAPIRequests, err = meter.Int64Counter(
		"courier.consul.api.requests",
		metric.WithDescription("Total number of Consul API requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create api.requests metric: %w", err)
	}

	r.consulAPIDuration, err = meter.Float64Histogram(
		"courier.consul.api.duration",
		metric.WithDescription("Duration of Consul API calls"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create api.duration metric: %w", err)
	}

	r.kvReadErrors, err = meter.Int64Counter(
		"courier.consul.kv.read_errors",
		metric.WithDescription("Total number of KV read errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create kv.read_errors metric: %w", err)
	}

	r.kvValueChanges, err = meter.Int64Counter(
		"courier.consul.kv.value_changes",
		metric.WithDescription("Total number of KV value changes detected"),
		metric.WithUnit("{change}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create kv.value_changes metric: %w", err)
	}

	r.addressUpdates, err = meter.Int64Counter(
		"courier.consul.resolver.address_updates",
		metric.WithDescription("Total number of address list updates published"),
		metric.WithUnit("{update}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create resolver.address_updates metric: %w", err)
	}

	return nil
}

func (r *Resolver) UpdateChan() <-chan []courier.TCPAddress {
	return r.updateChan
}

func (r *Resolver) Done() <-chan struct{} {
	return r.doneChan
}

func (r *Resolver) Stop() {
	close(r.doneChan)

	r.mu.Lock()
	if r.debounceTimer != nil {
		r.debounceTimer.Stop()
	}
	r.mu.Unlock()
}

func (r *Resolver) Start() {
	if err := r.updateServiceNameFromKV(); err != nil {
		r.logger.Printf("Failed to update service name from KV: %v", err)
	}

	fmt.Println("Starting resolver for service:", r.serviceName)

	// Initial service discovery
	if err := r.discover(); err != nil {
		r.logger.Printf("Initial service discovery failed: %v", err)
	}

	var wg sync.WaitGroup

	// Start service watcher
	wg.Add(1)

	go func() {
		defer wg.Done()
		r.watchServices()
	}()

	// Start service name watcher
	if r.kvKey != "" {
		wg.Add(1)

		go func() {
			defer wg.Done()
			r.watchKV()
		}()
	}

	wg.Wait()
	close(r.updateChan)
}

func (r *Resolver) updateServiceNameFromKV() error {
	ctx := context.Background()
	apiStartTime := time.Now()
	pair, _, err := r.client.KV().Get(r.kvKey, nil)
	apiDuration := time.Since(apiStartTime)

	r.recordAPIRequest(ctx, apiTypeKV, err == nil)
	r.recordAPIDuration(ctx, apiTypeKV, apiDuration)

	if err != nil {
		return fmt.Errorf("KV get error for key '%s': %w", r.kvKey, err)
	}

	if pair == nil || len(pair.Value) == 0 {
		r.recordKVReadError(ctx, r.kvKey, errorTypeKVRead)

		return fmt.Errorf("KV key '%s' not found or is empty", r.kvKey)
	}

	var kvData struct {
		ServiceName string `json:"serviceName"`
	}

	if err := json.Unmarshal(pair.Value, &kvData); err != nil {
		r.recordKVReadError(ctx, r.kvKey, errorTypeKVParse)

		return fmt.Errorf("KV parse error for key '%s': %w", r.kvKey, err)
	}

	if kvData.ServiceName != "" {
		r.mu.Lock()
		if kvData.ServiceName != r.serviceName {
			r.logger.Printf("Initial Service name updated from '%s' to '%s' from KV", r.serviceName, kvData.ServiceName)
			r.serviceName = kvData.ServiceName
			r.lastIndex = 0
			r.mu.Unlock()
			r.recordKVValueChange(ctx, r.kvKey)
		} else {
			r.mu.Unlock()
		}
	}

	return nil
}

// watchServices continuously monitors Consul for service changes.
func (r *Resolver) watchServices() {
	for {
		select {
		case <-r.doneChan:
			return
		default:
			if err := r.discover(); err != nil {
				r.logger.Printf("Service discovery failed: %v", err)
				select {
				case <-time.After(5 * time.Second):
				case <-r.doneChan:
					return
				}
			}
		}
	}
}

// discover performs a blocking query to find service instances.
func (r *Resolver) discover() error {
	ctx := context.Background()

	r.mu.RLock()
	serviceName := r.serviceName
	queryOpts := &consulapi.QueryOptions{
		WaitIndex: r.lastIndex,
		WaitTime:  r.waitTime,
	}
	r.mu.RUnlock()

	apiStartTime := time.Now()
	services, meta, err := r.client.Health().Service(serviceName, "", r.healthyOnly, queryOpts)
	apiDuration := time.Since(apiStartTime)

	r.recordAPIRequest(ctx, apiTypeHealthService, err == nil)
	r.recordAPIDuration(ctx, apiTypeHealthService, apiDuration)

	if err != nil {
		return fmt.Errorf("failed to query services: %w", err)
	}

	r.mu.Lock()

	if serviceName != r.serviceName {
		r.mu.Unlock()

		return nil
	}

	r.lastIndex = meta.LastIndex
	addresses := r.convertToTCPAddresses(services)
	currentCount := int64(len(addresses))

	r.lastInstanceCount = currentCount

	r.mu.Unlock()

	r.logger.Printf("Discovered %v instances for service '%s'", addresses, serviceName)

	r.scheduleAddressUpdate(ctx, serviceName, addresses)

	return nil
}

func (r *Resolver) scheduleAddressUpdate(ctx context.Context, serviceName string, addresses []courier.TCPAddress) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.debounceTimer != nil {
		r.debounceTimer.Stop()
		r.debounceTimer = nil
	}

	if areAddressesEqual(r.lastAddresses, addresses) {
		r.pendingAddresses = nil

		return
	}

	if r.debounceDuration <= 0 {
		r.lastAddresses = addresses
		r.publishAddressUpdate(ctx, serviceName, addresses)

		return
	}

	r.pendingAddresses = addresses
	r.debounceTimer = time.AfterFunc(r.debounceDuration, func() {
		select {
		case <-r.doneChan:
			return
		default:
		}

		r.mu.Lock()
		addrs := r.pendingAddresses

		if addrs == nil {
			r.mu.Unlock()

			return
		}

		r.lastAddresses = addrs
		r.pendingAddresses = nil
		svcName := r.serviceName
		r.mu.Unlock()

		r.publishAddressUpdate(context.Background(), svcName, addrs)
	})
}

func (r *Resolver) publishAddressUpdate(ctx context.Context, serviceName string, addresses []courier.TCPAddress) {
	r.recordAddressUpdate(ctx, serviceName)

	select {
	case r.updateChan <- addresses:
	case <-r.doneChan:
	}
}

func areAddressesEqual(a, b []courier.TCPAddress) bool {
	if len(a) != len(b) {
		return false
	}

	mapA := make(map[courier.TCPAddress]int)
	mapB := make(map[courier.TCPAddress]int)

	for _, addr := range a {
		mapA[addr]++
	}

	for _, addr := range b {
		mapB[addr]++
	}

	for addr, count := range mapA {
		if mapB[addr] != count {
			return false
		}
	}

	return true
}

// watchKV continuously monitors a KV key for changes to the service name.
func (r *Resolver) watchKV() {
	var lastKVIndex uint64

	ctx := context.Background()

	for {
		select {
		case <-r.doneChan:
			return
		default:
			apiStartTime := time.Now()
			pair, meta, err := r.client.KV().Get(r.kvKey, &consulapi.QueryOptions{
				WaitIndex: lastKVIndex,
				WaitTime:  r.waitTime,
			})
			apiDuration := time.Since(apiStartTime)

			r.recordAPIRequest(ctx, apiTypeKV, err == nil)
			r.recordAPIDuration(ctx, apiTypeKV, apiDuration)

			if err != nil {
				r.recordKVReadError(ctx, r.kvKey, errorTypeKVRead)
				r.logger.Printf("KV watch error: %v", err)
				select {
				case <-time.After(5 * time.Second):
				case <-r.doneChan:
					return
				}

				continue
			}

			if meta != nil {
				lastKVIndex = meta.LastIndex
			}

			if pair == nil || len(pair.Value) == 0 {
				continue
			}

			var kvData struct {
				ServiceName string `json:"serviceName"`
			}

			if err := json.Unmarshal(pair.Value, &kvData); err != nil {
				r.recordKVReadError(ctx, r.kvKey, errorTypeKVParse)
				r.logger.Printf("KV parse error: %v", err)

				continue
			}

			r.mu.Lock()
			if kvData.ServiceName != "" && kvData.ServiceName != r.serviceName {
				r.logger.Printf("Service name changed from '%s' to '%s'", r.serviceName, kvData.ServiceName)
				r.serviceName = kvData.ServiceName
				r.lastIndex = 0
				r.mu.Unlock()

				r.recordKVValueChange(ctx, r.kvKey)

				if err := r.discover(); err != nil {
					r.logger.Printf("Triggered service discovery failed: %v", err)
				}
			} else {
				r.mu.Unlock()
			}
		}
	}
}

// convertToTCPAddresses converts Consul service entries to courier.TCPAddress.
func (r *Resolver) convertToTCPAddresses(services []*consulapi.ServiceEntry) []courier.TCPAddress {
	addresses := make([]courier.TCPAddress, 0, len(services))

	for _, service := range services {
		host := service.Service.Address
		if host == "" {
			host = service.Node.Address
		}

		addresses = append(addresses, courier.TCPAddress{
			Host: host,
			Port: uint16(service.Service.Port),
		})
	}

	return addresses
}

func (r *Resolver) observeInstanceCount(_ context.Context, o metric.Observer) error {
	r.mu.RLock()
	serviceName := r.serviceName
	count := r.lastInstanceCount
	r.mu.RUnlock()

	if serviceName == "" {
		return nil
	}

	attrs := append(r.commonAttrs,
		attribute.String(attrConsulServiceName, serviceName),
	)

	o.ObserveInt64(r.serviceInstances, count, metric.WithAttributes(attrs...))

	return nil
}

func (r *Resolver) recordAPIRequest(ctx context.Context, apiType string, success bool) {
	if r.consulAPIRequests == nil {
		return
	}

	attrs := append(r.commonAttrs,
		attribute.String(attrAPIType, apiType),
		attribute.Bool(attrSuccess, success),
	)

	r.consulAPIRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (r *Resolver) recordAPIDuration(ctx context.Context, apiType string, duration time.Duration) {
	if r.consulAPIDuration == nil {
		return
	}

	attrs := append(r.commonAttrs,
		attribute.String(attrAPIType, apiType),
	)

	r.consulAPIDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

func (r *Resolver) recordKVReadError(ctx context.Context, kvKey, errorType string) {
	if r.kvReadErrors == nil {
		return
	}

	attrs := append(r.commonAttrs,
		attribute.String(attrKVKey, kvKey),
		attribute.String(attrErrorType, errorType),
	)

	r.kvReadErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (r *Resolver) recordKVValueChange(ctx context.Context, kvKey string) {
	if r.kvValueChanges == nil {
		return
	}

	attrs := append(r.commonAttrs,
		attribute.String(attrKVKey, kvKey),
	)

	r.kvValueChanges.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (r *Resolver) recordAddressUpdate(ctx context.Context, serviceName string) {
	if r.addressUpdates == nil {
		return
	}

	attrs := append(r.commonAttrs,
		attribute.String(attrConsulServiceName, serviceName),
	)

	r.addressUpdates.Add(ctx, 1, metric.WithAttributes(attrs...))
}
