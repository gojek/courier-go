package courier

import (
	"encoding/json"
	"net/http"
)

// TelemetryHandler returns a http.Handler that exposes the connected clients information
func (c *Client) TelemetryHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cl := c.allClientInfo()

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"multi":   c.options.multiConnectionMode,
			"clients": cl,
		})
	})
}
