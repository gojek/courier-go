package otelcourier

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/gojek/courier-go"
)

type infoHandler http.HandlerFunc

type infoResponse struct {
	Clients []courier.MQTTClientInfo `json:"clients,omitempty"`
}

func (e infoHandler) callback(attrs ...attribute.KeyValue) metric.Int64Callback {
	hf := http.HandlerFunc(e)

	return func(ctx context.Context, observer metric.Int64Observer) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		if err != nil {
			return err
		}

		resp := inMemResponseWriter{headers: make(http.Header)}

		hf.ServeHTTP(&resp, req)

		var ir infoResponse
		if err := json.NewDecoder(&resp.body).Decode(&ir); err != nil {
			return err
		}

		for _, cl := range ir.Clients {
			observer.Observe(
				boolInt64(cl.Connected),
				metric.WithAttributes(append(attrs, MQTTClientID.String(cl.ClientID))...),
			)
		}

		return nil
	}
}

type inMemResponseWriter struct {
	body    bytes.Buffer
	headers http.Header
	status  int
}

func (w *inMemResponseWriter) Header() http.Header { return w.headers }

func (w *inMemResponseWriter) Write(b []byte) (int, error) { return w.body.Write(b) }

func (w *inMemResponseWriter) WriteHeader(statusCode int) { w.status = statusCode }

func boolInt64(connected bool) int64 {
	if connected {
		return 1
	}

	return 0
}
