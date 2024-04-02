package otelcourier

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/gojek/courier-go"
)

type infoEmitter struct{ value atomic.Value }

func newInfoEmitter() *infoEmitter {
	e := &infoEmitter{}
	e.value.Store(courier.ClientMeta{})

	return e
}

func (e *infoEmitter) emit(meta courier.ClientMeta) { e.value.Store(meta) }

func (e *infoEmitter) load() courier.ClientMeta { return e.value.Load().(courier.ClientMeta) }

func (e *infoEmitter) callback(attrs ...attribute.KeyValue) metric.Int64Callback {
	return func(_ context.Context, observer metric.Int64Observer) error {
		for _, cl := range e.load().Clients {
			observer.Observe(
				boolInt64(cl.Connected),
				metric.WithAttributes(append(attrs, MQTTClientID.String(cl.ClientID))...),
			)
		}

		return nil
	}
}

func (t *OTel) Emit(_ context.Context, meta courier.ClientMeta) { t.emitter.emit(meta) }

func boolInt64(connected bool) int64 {
	if connected {
		return 1
	}

	return 0
}
