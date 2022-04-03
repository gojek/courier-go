package xds

import "github.com/gojekfarm/courier-go"

type Resolver struct {
}

var _ courier.Resolver = (*Resolver)(nil)

func (r *Resolver) UpdateChan() <-chan []courier.TCPAddress {
	//TODO implement me
	panic("implement me")
}

func (r *Resolver) Done() <-chan struct{} {
	//TODO implement me
	panic("implement me")
}
