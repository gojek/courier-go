package courier

import (
	"encoding/json"
	"net/http"
)

// InfoHandler returns a http.Handler that exposes the connected clients information
func (c *Client) InfoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ir := c.infoResponse()

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ir)
	})
}
