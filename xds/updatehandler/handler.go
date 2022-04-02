package updatehandler

import (
	"github.com/gojekfarm/courier-go/xds/types"
	"log"
	"sync"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type Config struct {
	ConnErrCallback func(error)
	Epw             []types.EndpointWatcher
}

type endpointUpdateHandler struct {
	//endpoints caches the latest endpoints
	endpoints map[string][]string

	//callbacks hold the callback for each endpoint name
	callbackMap map[string]types.CallbackFunc

	//connErrCallback is the callback when connection error is received on xds client
	connErrCallback func(error)

	mu sync.Mutex
}

func New(cfg Config) UpdateHandler {
	h := &endpointUpdateHandler{connErrCallback: cfg.ConnErrCallback}
	h.initiateEndpoints(cfg.Epw)
	return h
}

func (h *endpointUpdateHandler) AddEndpointWatcher(watcher types.EndpointWatcher) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.callbackMap[watcher.Endpoint] = watcher.Callback
}

func (h *endpointUpdateHandler) RemoveEndpointWatcher(watcher types.EndpointWatcher) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.callbackMap, watcher.Endpoint)
	delete(h.endpoints, watcher.Endpoint)
}

func (h *endpointUpdateHandler) NewEndpoints(endpoints map[string][]string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	updates := h.getDiff(endpoints)

	for endpoint, addresses := range updates {
		if f, ok := h.callbackMap[endpoint]; ok {
			f(addresses)
			continue
		}
		log.Printf("No callback registered for subscribed resource: %s", endpoint)
	}

	h.endpoints = endpoints
}

func (h *endpointUpdateHandler) NewConnectionError(err error) {
	if h.connErrCallback != nil {
		h.connErrCallback(err)
	}
}

func (h *endpointUpdateHandler) initiateEndpoints(epw []types.EndpointWatcher) {
	endpoints := make(map[string][]string)
	callbacks := make(map[string]types.CallbackFunc)

	for _, v := range epw {
		endpoints[v.Endpoint] = []string{}
		callbacks[v.Endpoint] = v.Callback
	}

	h.endpoints = endpoints
	h.callbackMap = callbacks
}

func (h *endpointUpdateHandler) getDiff(updatedEndpoints map[string][]string) map[string][]string {
	ret := make(map[string][]string)

	for key, val := range h.endpoints {
		newVal, ok := updatedEndpoints[key]
		if ok && !cmp.Equal(val, newVal, cmpopts.SortSlices(func(a, b string) bool { return a < b })) {
			ret[key] = newVal
		}
	}

	return ret
}
