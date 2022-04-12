module github.com/gojek/courier-go/otelcourier

go 1.16

require (
	github.com/gojek/courier-go v0.1.0
	github.com/stretchr/testify v1.7.1
	go.opentelemetry.io/otel v1.6.1
	go.opentelemetry.io/otel/sdk v1.6.1
	go.opentelemetry.io/otel/trace v1.6.1
)

replace github.com/gojek/courier-go => ../
