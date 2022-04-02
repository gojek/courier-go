package types

type EndpointWatcher struct {
	Endpoint string
	Callback CallbackFunc
}

type CallbackFunc func([]string)
