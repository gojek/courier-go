package client

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
)

//ToDo: Use this client and not the stream directly
type clientDecorator struct {
	client Client

	restartFunc       restartFunc
	sendFunc          sendFunc
	parseResponseFunc parseFunc
	receiveFunc       receiveFunc
}

type middlewareOpts struct {
	client Client

	restartMiddleware       func(restartFunc) restartFunc
	sendMiddleware          func(sendFunc) sendFunc
	parseResponseMiddleware func(parseFunc) parseFunc
	receiveMiddleware       func(receiveFunc) receiveFunc
}

func GetClientDecorator(opts middlewareOpts) (Client, error) {
	if opts.client == nil {
		return nil, fmt.Errorf("middleware_wrapped_client: nil client provided")
	}

	receive := opts.client.Receive
	if opts.receiveMiddleware != nil {
		receive = opts.receiveMiddleware(receive)
	}

	parse := opts.client.ParseResponse
	if opts.parseResponseMiddleware != nil {
		parse = opts.parseResponseMiddleware(parse)
	}

	send := opts.client.SendRequest
	if opts.sendMiddleware != nil {
		send = opts.sendMiddleware(send)
	}

	restart := opts.client.RestartEDSClient
	if opts.restartMiddleware != nil {
		restart = opts.restartMiddleware(restart)
	}

	return &clientDecorator{
		client: opts.client,

		restartFunc:       restart,
		sendFunc:          send,
		parseResponseFunc: parse,
		receiveFunc:       receive,
	}, nil
}

func (d *clientDecorator) RestartEDSClient(ctx context.Context, cc grpc.ClientConnInterface) error {
	return d.restartFunc(ctx, cc)
}

func (d *clientDecorator) SendRequest(resourceNames []string, version, nonce, errMsg string) error {
	return d.sendFunc(resourceNames, version, nonce, errMsg)
}

func (d *clientDecorator) ParseResponse(r proto.Message) ([]*anypb.Any, string, string, error) {
	return d.parseResponseFunc(r)
}

func (d *clientDecorator) Receive() (proto.Message, error) {
	return d.receiveFunc()
}
