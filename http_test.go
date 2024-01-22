package courier

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient_TelemetryHandler(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		opts   func(*testing.T) []ClientOption
	}{
		{
			name:   "single connection mode",
			status: http.StatusOK,
			opts:   func(t *testing.T) []ClientOption { return nil },
			body: fmt.Sprintf(`{"clients":[{"addresses":[{"host":"%s","port":%d}],"client_id":"clientID","username":"","resume_subs":false,"clean_session":false,"auto_reconnect":true,"connected":true}],"multi":false}
`, testBrokerAddress.Host, testBrokerAddress.Port),
		},
		{
			name:   "multi connection mode",
			status: http.StatusOK,
			opts: func(t *testing.T) []ClientOption {
				ch := make(chan []TCPAddress, 1)
				dCh := make(chan struct{})
				mr := newMockResolver(t)

				mr.On("UpdateChan").Return(ch)
				mr.On("Done").Return(dCh)

				go func() {
					ch <- []TCPAddress{testBrokerAddress, testBrokerAddress}

					<-time.After(2 * time.Second)
					close(ch)

					dCh <- struct{}{}
				}()

				return []ClientOption{
					WithResolver(mr),
					UseMultiConnectionMode,
				}
			},
			body: fmt.Sprintf(`{"clients":[{"addresses":[{"host":"%s","port":%d}],"client_id":"clientID-0-1","username":"","resume_subs":false,"clean_session":false,"auto_reconnect":true,"connected":true},{"addresses":[{"host":"%s","port":%d}],"client_id":"clientID-1-1","username":"","resume_subs":false,"clean_session":false,"auto_reconnect":true,"connected":true}],"multi":true}
`, testBrokerAddress.Host, testBrokerAddress.Port, testBrokerAddress.Host, testBrokerAddress.Port),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eOpts := tt.opts(t)

			c, err := NewClient(append(defOpts, eOpts...)...)
			assert.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				_ = c.Run(ctx)
			}()

			h := c.InfoHandler()

			assert.True(t, WaitForConnection(c, 2*time.Second, 100*time.Millisecond))

			req := httptest.NewRequest(http.MethodGet, "/telemetry", nil)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			assert.Equal(t, tt.status, rr.Code)
			assert.Equal(t, tt.body, rr.Body.String())
		})
	}
}
