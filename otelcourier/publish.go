package otelcourier

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gojek/courier-go"
)

const (
	publishSpanName   = "otelcourier.Publish"
	publishErrMessage = "publish error"
)

func (t *Tracer) PublisherMiddleware(next courier.Publisher) courier.Publisher {
	return courier.PublisherFunc(func(
		ctx context.Context,
		topic string,
		message interface{},
		opts ...courier.Option,
	) error {
		traceOpts := []trace.SpanStartOption{
			trace.WithAttributes(MQTTTopic.String(topic)),
			trace.WithAttributes(semconv.ServiceNameKey.String(t.service)),
			trace.WithSpanKind(trace.SpanKindProducer),
		}
		traceOpts = append(traceOpts, mapOptions(opts)...)

		ctx, span := t.tracer.Start(ctx, publishSpanName, traceOpts...)
		defer span.End()

		if tmc, ok := message.(propagation.TextMapCarrier); ok {
			t.propagator.Inject(ctx, tmc)
		}

		err := next.Publish(ctx, topic, message, opts...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, publishErrMessage)
		}

		return err
	})
}

func mapOptions(opts []courier.Option) []trace.SpanStartOption {
	res := make([]trace.SpanStartOption, 0, len(opts))

	for _, opt := range opts {
		switch opt := opt.(type) {
		case courier.QOSLevel:
			res = append(res, trace.WithAttributes(MQTTQoS.Int(int(opt))))
		case courier.Retained:
			res = append(res, trace.WithAttributes(MQTTRetained.Bool(bool(opt))))
		}
	}

	return res
}
