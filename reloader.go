package courier

// Reloader reloads and reconnects to new broker addresses
type Reloader interface {
	// Load asks client to reload and reconnect to new broker address
	Load() []ClientOption
}

type DefaultReloader struct {
	c *Client
}

func (drl *DefaultReloader) Load() []ClientOption {
	return nil
}
