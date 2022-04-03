package client

import (
	"context"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
)

type restartFunc func(context.Context, *grpc.ClientConn) error

type sendFunc func(resourceNames []string, version string, nonce string, errMsg string) error

type parseFunc func(r proto.Message) ([]*anypb.Any, string, string, error)

type receiveFunc func() (proto.Message, error)
