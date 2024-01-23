package courier

import (
	"context"
	"time"
)

// ClientMeta contains information about the internal MQTT client(s)
type ClientMeta struct {
	MultiConnMode bool
	Clients       []MQTTClientInfo
	Subscriptions map[string]QOSLevel
}

// ClientInfoEmitter emits broker info.
// This can be called concurrently, implementations should be concurrency safe.
type ClientInfoEmitter interface {
	Emit(ctx context.Context, meta ClientMeta)
}

// ClientInfoEmitterConfig is used to configure the broker info emitter.
type ClientInfoEmitterConfig struct {
	// Interval is the interval at which the broker info emitter emits broker info.
	Interval time.Duration
	Emitter  ClientInfoEmitter
}

func (cfg *ClientInfoEmitterConfig) apply(o *clientOptions) { o.infoEmitterCfg = cfg }

func (c *Client) runBrokerInfoEmitter(ctx context.Context) {
	tick := time.NewTicker(c.options.infoEmitterCfg.Interval)
	defer tick.Stop()

	em := c.options.infoEmitterCfg.Emitter

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			ir := c.infoResponse()
			cm := ClientMeta{
				MultiConnMode: ir.MultiConnMode,
				Clients:       ir.Clients,
				Subscriptions: ir.Subscriptions,
			}

			go em.Emit(ctx, cm)
		}
	}
}
