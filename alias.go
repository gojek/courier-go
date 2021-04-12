package courier

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Store is an interface which can be used to provide implementations
// for message persistence.
//
// [IMPORTANT]
// When implementing a store with a shared storage (ex: redis) across multiple
// application instances, it should be ensured that the keys are namespaced for
// each application instance otherwise there will be collisions.
// The messages are identified based on message id from the MQTT packets, and they have values
// in range (0, 2^16), this coincides with max number of in-flight messages
type Store = mqtt.Store

// Message defines the externals that a message implementation must support.
// These are received messages that are passed to the callbacks, not internal messages
type Message = mqtt.Message

// NewMemoryStore returns a pointer to a new instance of
// mqtt.MemoryStore, the instance is not initialized and ready to
// use until Open() has been called on it.
func NewMemoryStore() Store {
	return mqtt.NewMemoryStore()
}
