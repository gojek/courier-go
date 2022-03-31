package xds

import (
	"context"
	"encoding/base64"
	"encoding/json"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/gojekfarm/courier-go/xds/bootstrap"
	"google.golang.org/grpc"
	"os"
	"os/signal"
	"testing"
)

func TestClient_streamEndpoints(t *testing.T) {
	d, err := json.Marshal(&bootstrap.Config{XDSServer: &bootstrap.ServerConfig{
		ServerURI: "",
		NodeProto: nil,
	}})
	if err != nil {
		t.Error(err)
	}

	if err := os.Setenv("COURIER_XDS_BOOTSTRAP_CONFIG_BASE64", base64.StdEncoding.EncodeToString(d)); err != nil {
		t.Error(err)
	}

	cfg, err := bootstrap.NewConfig()
	if err != nil {
		t.Error(err)
	}

	cc, _ := grpc.Dial(cfg.XDSServer.ServerURI)
	c := &Client{
		cc:   cc,
		node: cfg.XDSServer.NodeProto.(*corev3.Node),
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)

	if err := c.streamEndpoints(ctx, []string{"customer"}, "", ""); err != nil {
		panic(err)
	}
}
