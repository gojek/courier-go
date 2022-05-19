---
title: Metrics
description: Tutorial on writing a metric middleware
---

A metric middleware is helpful to track and aggregate metrics whenever you invoke `client.Publish()`, the call is first passed through the chain of middlewares and then published to broker.

```go title="publish_metrics.go" {6,17,20,25}
func main() {
	var client *courier.Client

	pm := &promCollector{}

	client.UsePublisherMiddleware(pm.publisherMiddleware)
}

type promCollector struct {
	attemps *prometheus.CounterVec
	errors  *prometheus.CounterVec
	success *prometheus.CounterVec
}

func (p *promCollector) publisherMiddleware(next courier.Publisher) courier.Publisher {
	return courier.PublisherFunc(func(ctx context.Context, topic string, data interface{}, opts ...courier.Option) error {
		p.attemps.With(prometheus.Labels{}).Inc()

		if err := next.Publish(ctx, topic, data, opts...); err != nil {
			p.errors.With(prometheus.Labels{}).Inc()

			return err
		}

		p.success.With(prometheus.Labels{}).Inc()

		return nil
	})
}

```
