package xds_test

import (
	"context"
	"os"
	"os/signal"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gojekfarm/courier-go"
	"github.com/gojekfarm/courier-go/xds"
	"github.com/gojekfarm/courier-go/xds/backoff"
	"github.com/gojekfarm/courier-go/xds/bootstrap"
)

func ExampleNewResolver() {
	cfg, err := bootstrap.NewConfigFromContents([]byte(`{
 "xds_server": {
   "server_uri": "localhost:9100",
   "node": {
     "id": "52fdfc07-2182-454f-963f-5f0f9a621d72",
     "cluster": "cluster",
     "metadata": {
       "TRAFFICDIRECTOR_GCP_PROJECT_NUMBER": "123456789012345",
       "TRAFFICDIRECTOR_NETWORK_NAME": "thedefault"
     },
     "locality": {
       "zone": "uscentral-5"
     }
   }
 }
}`,
	))
	if err != nil {
		panic(err)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)

	cc, err := grpc.DialContext(ctx, cfg.XDSServer.ServerURI, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	xdsClient := xds.NewClient(xds.Options{
		XDSTarget:       "xds:///broker.domain",
		NodeProto:       cfg.XDSServer.NodeProto.(*corev3.Node),
		ClientConn:      cc,
		BackoffStrategy: &backoff.DefaultExponential,
	})

	if err := xdsClient.Start(ctx); err != nil {
		panic(err)
	}

	r := xds.NewResolver(xdsClient)

	c, err := courier.NewClient(courier.WithResolver(r))
	if err != nil {
		panic(err)
	}

	if err := c.Start(); err != nil {
		panic(err)
	}

	<-ctx.Done()
}
