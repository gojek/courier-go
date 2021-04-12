package courier

// OnConnectHandler is a callback that is called when the client
// state changes from disconnected to connected. Both
// at initial connection and on reconnection
type OnConnectHandler func(PubSub)

// OnConnectionLostHandler is a callback type which can be set to be
// executed upon an unintended disconnection from the MQTT broker.
// Disconnects caused by calling Disconnect or ForceDisconnect will
// not cause an WithOnConnectionLost callback to execute.
type OnConnectionLostHandler func(error)

// OnReconnectHandler is invoked prior to reconnecting after
// the initial connection is lost
type OnReconnectHandler func(PubSub)

type MessageHandler func(PubSub, Decoder)
