package xds_test

import (
	"context"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/gojekfarm/courier-go"
	"github.com/gojekfarm/courier-go/xds"
	"github.com/gojekfarm/courier-go/xds/backoff"
	"github.com/gojekfarm/courier-go/xds/bootstrap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"os/signal"
)

func ExampleNewResolver() {
	cfg, err := bootstrap.NewConfig()
	if err != nil {
		panic(err)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)

	cc, err := grpc.DialContext(ctx, cfg.XDSServer.ServerURI, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	r := xds.NewResolver(xds.NewClient(xds.Options{
		XDSTarget:       "xds:///broker.domain",
		NodeProto:       cfg.XDSServer.NodeProto.(*corev3.Node),
		ClientConn:      cc,
		BackoffStrategy: &backoff.DefaultExponential,
	}))

	if _, err := courier.NewClient(courier.WithResolver(r)); err != nil {
		panic(err)
	}
}
