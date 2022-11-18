module github.com/gojek/courier-go/xds

go 1.16

require (
	github.com/envoyproxy/go-control-plane v0.10.1
	github.com/gojek/courier-go v0.3.0
	github.com/golang/protobuf v1.5.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/gojek/courier-go => ../
