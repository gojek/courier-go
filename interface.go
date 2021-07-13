package courier

// PubSub exposes all the operational functionalities of Client with
// Publisher, Subscriber, Unsubscriber and ConnectionInformer
type PubSub interface {
	Publisher
	Subscriber
	Unsubscriber
	ConnectionInformer
}

// ConnectionInformer can be used to get information about the connection
type ConnectionInformer interface {
	// IsConnected checks whether the client is connected to the broker
	IsConnected() bool
}
