package courier

// Stopper defines behaviour of an MQTT client stop.
type Stopper interface {
	Stop() error
}

// StopHandlerFunc defines signature of a Stop function.
type StopHandlerFunc func() error

// Stop implements Stopper interface on StopHandlerFunc.
func (f StopHandlerFunc) Stop() error {
	return f()
}

type stopMiddleware interface {
	Middleware(stopper Stopper) Stopper
}

// StopMiddlewareFunc functions are closures that intercept Stopper.Stop calls.
type StopMiddlewareFunc func(Stopper) Stopper

// Middleware allows StopMiddlewareFunc to implement the stopMiddleware interface.
func (smw StopMiddlewareFunc) Middleware(stopper Stopper) Stopper {
	return smw(stopper)
}
