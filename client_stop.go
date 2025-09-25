package courier

// UseStopMiddleware allows setting a Stopper middleware to intercept Stop calls.
func (c *Client) UseStopMiddleware(mwf StopMiddlewareFunc) {
	c.stopMiddleware = mwf
	if mwf != nil {
		c.stopHandler = mwf.Middleware(stopHandlerFunc(c))
	} else {
		c.stopHandler = stopHandlerFunc(c)
	}
}

func stopHandlerFunc(c *Client) Stopper {
	return StopHandlerFunc(func() error {
		return c.stop()
	})
}
