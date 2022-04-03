package client

import (
	"context"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
)

type clientDecorator struct {
	client Client

	restartMiddleware       func(restartFunc) restartFunc
	sendMiddleware          func(sendFunc) sendFunc
	parseResponseMiddleware func(parseFunc) parseFunc
	receiveMiddleware       func(receiveFunc) receiveFunc
}

func (d *clientDecorator) RestartEDSClient(ctx context.Context, cc *grpc.ClientConn) error {
	if d.restartMiddleware != nil {
		err := d.restartMiddleware(d.client.RestartEDSClient)(ctx, cc)
		return err
	}
	return d.client.RestartEDSClient(ctx, cc)
}

func (d *clientDecorator) SendRequest(resourceNames []string, version, nonce, errMsg string) error {
	if d.sendMiddleware != nil {
		return d.sendMiddleware(d.client.SendRequest)(resourceNames, version, nonce, errMsg)
	}
	return d.client.SendRequest(resourceNames, version, nonce, errMsg)
}

func (d *clientDecorator) ParseResponse(r proto.Message) ([]*anypb.Any, string, string, error) {
	if d.parseResponseMiddleware != nil {
		return d.parseResponseMiddleware(d.client.ParseResponse)(r)
	}
	return d.client.ParseResponse(r)
}

func (d *clientDecorator) Receive() (proto.Message, error) {
	if d.receiveMiddleware != nil {
		return d.receiveMiddleware(d.client.Receive)()
	}
	return d.client.Receive()
}
