package updatehandler

import "github.com/gojekfarm/courier-go/xds/types"

type UpdateHandler interface {
	// NewEndpoints handles updates to xDS ClusterLoadAssignment (or tersely
	// referred to as Endpoints) resources.
	NewEndpoints(map[string][]string)
	// NewConnectionError handles connection errors from the xDS stream. The
	// error will be reported to all the resource watchers.
	NewConnectionError(err error)
	// AddEndpointWatcher adds endpoint and its handler
	AddEndpointWatcher(watcher types.EndpointWatcher)
	//RemoveEndpointWatcher removes endpoint from callbacks and cached endpoints
	RemoveEndpointWatcher(watcher types.EndpointWatcher)
}
