package updatehandler

type UpdateHandler interface {
	// NewEndpoints handles updates to xDS ClusterLoadAssignment (or tersely
	// referred to as Endpoints) resources.
	NewEndpoints(map[string][]string)
	// NewConnectionError handles connection errors from the xDS stream. The
	// error will be reported to all the resource watchers.
	NewConnectionError(err error)
}
