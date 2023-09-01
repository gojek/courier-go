package courier

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-multierror"

	"github.com/gojekfarm/xtools/generic/slice"
	"github.com/gojekfarm/xtools/generic/xmap"
)

// TCPAddress specifies Host and Port for remote broker
type TCPAddress struct {
	Host string
	Port uint16
}

func (t TCPAddress) String() string { return fmt.Sprintf("%s:%d", t.Host, t.Port) }

// Resolver sends TCPAddress updates on channel returned by UpdateChan() channel.
type Resolver interface {
	// UpdateChan returns a channel where TCPAddress updates can be received.
	UpdateChan() <-chan []TCPAddress
	// Done returns a channel which is closed when the Resolver is no longer running.
	Done() <-chan struct{}
}

// WithResolver sets the specified Resolver.
func WithResolver(resolver Resolver) ClientOption {
	return optionFunc(func(c *clientOptions) {
		c.resolver = resolver
	})
}

func (c *Client) watchAddressUpdates(r Resolver) {
	for {
		select {
		case <-r.Done():
			return
		case addrs := <-r.UpdateChan():
			if err := c.attemptConnections(addrs); err != nil {
				c.options.logger.Error(context.Background(), err, map[string]any{
					"action":    "attemptConnections",
					"addresses": addrs,
				})
			}
		}
	}
}

func (c *Client) attemptConnections(addrs []TCPAddress) error {
	if len(addrs) == 0 {
		return nil
	}

	if c.options.multiConnectionMode {
		return c.attemptMultiConnections(addrs)
	}

	return c.attemptSingleConnection(addrs)
}

func (c *Client) attemptMultiConnections(addrs []TCPAddress) error {
	clients, err := c.multipleClients(addrs)
	if err != nil {
		return err
	}

	if err := c.reloadClients(clients); err != nil {
		return err
	}

	return c.resumeSubscriptions()
}

func (c *Client) resumeSubscriptions() error {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	if len(c.subscriptions) == 0 {
		return nil
	}

	return slice.Reduce(slice.MapConcurrent(xmap.Values(c.subscriptions), func(tm *subscriptionMeta) error {
		return c.subscriber.Subscribe(context.Background(), tm.topic, tm.callback, tm.options...)
	}), accumulateErrors)
}

func (c *Client) reloadClients(clients map[string]mqtt.Client) error {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()

	oldClients := xmap.Values(c.mqttClients)
	c.mqttClients = clients

	go func(oldClients []mqtt.Client) {
		if len(oldClients) == 0 {
			return
		}

		if err := slice.Reduce(
			slice.MapConcurrent(oldClients, func(cc mqtt.Client) error {
				cc.Disconnect(uint(c.options.gracefulShutdownPeriod / time.Millisecond))

				return nil
			}),
			accumulateErrors,
		); err != nil {
			c.options.logger.Error(context.Background(), err, map[string]any{"action": "reloadClients"})
		}
	}(oldClients)

	return nil
}

func (c *Client) multipleClients(addrs []TCPAddress) (map[string]mqtt.Client, error) {
	clients := &sync.Map{}

	if err := slice.Reduce(slice.MapConcurrent(addrs, func(addr TCPAddress) error {
		opts := *c.options
		opts.brokerAddress = fmt.Sprintf("%s:%d", addr.Host, addr.Port)

		cc := newClientFunc.Load().(func(*mqtt.ClientOptions) mqtt.Client)(toClientOptions(c, &opts))

		t := cc.Connect()
		if !t.WaitTimeout(c.options.connectTimeout) {
			return ErrConnectTimeout
		}

		if err := t.Error(); err != nil {
			return err
		}

		clients.Store(addr.String(), cc)

		return nil
	}), accumulateErrors); err != nil {
		return nil, err
	}

	res := map[string]mqtt.Client{}

	clients.Range(func(key, value interface{}) bool {
		// nolint: errcheck
		res[key.(string)] = value.(mqtt.Client)

		return true
	})

	return res, nil
}

func (c *Client) newClient(addrs []TCPAddress, attempt int) mqtt.Client {
	addr := addrs[attempt%len(addrs)]

	opts := *c.options
	opts.brokerAddress = fmt.Sprintf("%s:%d", addr.Host, addr.Port)

	cc := newClientFunc.Load().(func(*mqtt.ClientOptions) mqtt.Client)(toClientOptions(c, &opts))

	t := cc.Connect()
	if !t.WaitTimeout(c.options.connectTimeout) {
		return c.newClient(addrs, attempt+1)
	}

	if err := t.Error(); err != nil {
		// TODO: add retry backoff or use ExponentialStartStrategy utility
		c.options.logger.Error(context.Background(), err, map[string]any{
			"action": "newClient",
			"addr":   addr,
		})

		return c.newClient(addrs, attempt+1)
	}

	return cc
}

func (c *Client) reloadClient(cc mqtt.Client) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()

	oldClient := c.mqttClient
	c.mqttClient = cc

	go func() {
		if oldClient != nil {
			oldClient.Disconnect(uint(c.options.gracefulShutdownPeriod / time.Millisecond))
		}
	}()
}

func accumulateErrors(prev error, curr error) error {
	var err *multierror.Error

	switch {
	case errors.As(prev, &err):
		err.ErrorFormat = singleLineFormatFunc

		return multierror.Append(err, curr).ErrorOrNil()
	default:
		return multierror.Append(&multierror.Error{ErrorFormat: singleLineFormatFunc}, prev, curr).ErrorOrNil()
	}
}

func singleLineFormatFunc(es []error) string {
	if len(es) == 1 {
		return fmt.Sprintf("1 error occurred: [%s]", es[0])
	}

	errorsList := make([]string, len(es))
	for i, err := range es {
		errorsList[i] = fmt.Sprintf("[%s]", err)
	}

	return fmt.Sprintf(
		"%d errors occurred: %s",
		len(es), strings.Join(errorsList, " | "))
}
