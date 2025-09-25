package otelcourier

import "github.com/gojek/courier-go"

// Stop implements the courier.Stopper interface to unregister any handlers or clean up resources.
func (t *OTel) Stop() error {
	if t.infoHandlerRegistration != nil {
		if err := t.infoHandlerRegistration.Unregister(); err != nil {
			return err
		}

		t.infoHandlerRegistration = nil
	}

	return nil
}

// StopMiddleware returns a Stopper middleware that ensures OTel async metric is unregistered when the client stops.
func (t *OTel) StopMiddleware(next courier.Stopper) courier.Stopper {
	return courier.StopHandlerFunc(func() error {
		err := next.Stop()

		if cleanupErr := t.Stop(); cleanupErr != nil {
			if err == nil {
				return cleanupErr
			}
		}

		return err
	})
}
