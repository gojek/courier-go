package xdsclient

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"testing"
)

func TestClient_streamEndpoints(t *testing.T) {
	//d, err := json.Marshal(&bootstrap.Config{XDSServer: &bootstrap.ServerConfig{
	//	ServerURI: "localhost:9100",
	//	NodeProto: &corev3.Node{
	//		Id:      "52fdfc07-2182-454f-963f-5f0f9a621d72",
	//		Cluster: "pusher",
	//		Metadata: &structpb.Struct{Fields: map[string]*structpb.Value{
	//			"APP": {Kind: &structpb.Value_StringValue{StringValue: "pusher"}},
	//		}},
	//		Locality: &corev3.Locality{
	//			Region: "asia-east1",
	//			Zone:   "asia-east1-a",
	//		},
	//	},
	//}})
	//if err != nil {
	//	t.Error(err)
	//}
	//
	//cfg, err := bootstrap.NewConfigFromContents(d)
	//if err != nil {
	//	t.Error(err)
	//}
	//
	//c, err := New(cfg.XDSServer, "customer")
	//
	//fmt.Println(err)

	ctx, f := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)

	f()
	<- ctx.Done()

	//c.Close()

	fmt.Println("Soemthiang 1")
	//
	<- ctx.Done()
	fmt.Println("Soemthiang")
}
