module github.com/gojek/courier-go/xds

go 1.24

require (
	github.com/envoyproxy/go-control-plane v0.12.0
	github.com/gojek/courier-go v0.7.14
	github.com/golang/protobuf v1.5.4
	github.com/stretchr/testify v1.9.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240227224415-6ceb2ff114de
	google.golang.org/grpc v1.63.2
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cncf/xds/go v0.0.0-20231128003011-0fa0005c9caa // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.0.4 // indirect
	github.com/gojek/paho.mqtt.golang v1.7.1 // indirect
	github.com/gojekfarm/xtools/generic v0.7.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240227224415-6ceb2ff114de // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/gojek/courier-go => ../
