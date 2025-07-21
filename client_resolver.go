package courier

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"

	mqtt "github.com/gojek/paho.mqtt.golang"
	"github.com/gojekfarm/xtools/generic"
	"github.com/gojekfarm/xtools/generic/slice"
	"github.com/gojekfarm/xtools/generic/xmap"
)

// TCPAddress specifies Host and Port for remote broker
type TCPAddress struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
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
	if c.options.multiConnectionMode {
		return c.attemptMultiConnections(addrs)
	}

	return c.attemptSingleConnection(addrs)
}

func (c *Client) attemptMultiConnections(addrs []TCPAddress) error {
	c.clientMu.Lock()
	c.reloadClients(c.multipleClients(addrs))
	c.clientMu.Unlock()

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

func (c *Client) reloadClients(clients map[string]mqtt.Client) {
	oldClients := xmap.Values(c.mqttClients)
	ocs := slice.Map(oldClients, func(is *internalState) mqtt.Client { return is.client })

	if len(clients) > 0 {
		ncs := make(map[string]*internalState, len(clients))

		for k, cc := range clients {
			ncs[k] = &internalState{client: cc, subsCalled: generic.NewSet[string]()}
		}

		c.mqttClients = ncs
	}

	c.options.logger.Info(context.Background(), "reloading clients", map[string]any{
		"oldIds": slice.Map(ocs, clientIDMapper),
		"newIds": slice.Map(xmap.Values(clients), clientIDMapper),
	})

	go func(oldClients []mqtt.Client, newClientsLen int) {
		if len(oldClients) == 0 || newClientsLen == 0 {
			c.options.logger.Info(context.Background(), "skipping disconnections", map[string]any{})

			return
		}

		c.disconnectAll(oldClients)
	}(ocs, len(clients))
}

func (c *Client) disconnectAll(cls []mqtt.Client) {
	slice.MapConcurrent(cls, func(cc mqtt.Client) error {
		c.options.logger.Info(context.Background(), "disconnecting client", map[string]any{
			"clientID": clientIDMapper(cc),
		})

		cc.Disconnect(uint(c.options.gracefulShutdownPeriod / time.Millisecond))

		return nil
	})
}

type indexAddress struct {
	index int
	addr  TCPAddress
}

func (c *Client) multipleClients(addrs []TCPAddress) map[string]mqtt.Client {
	clients := &sync.Map{}
	currRev := c.multiConnRevision.Load()

	i := &atomicCounter{}
	iaddrs := slice.Map(addrs, func(a TCPAddress) indexAddress { return indexAddress{index: int(i.next()), addr: a} })

	slice.MapConcurrentWithContext(context.Background(), iaddrs, func(ctx context.Context, ia indexAddress) error {
		opts := *c.options
		opts.brokerAddress = ia.addr.String()

		pOpts := toClientOptions(c, &opts, fmt.Sprintf("-%d-%d", ia.index, currRev+1))
		cc := newClientFunc.Load().(func(*mqtt.ClientOptions) mqtt.Client)(pOpts)

		clients.Store(fmt.Sprintf("%s-%d", ia.addr.String(), ia.index), cc)

		c.options.logger.Info(ctx, "attempting connection", map[string]any{
			"multiConnRevision": currRev,
			"clientID":          pOpts.ClientID,
		})

		t := cc.Connect()

		if !t.WaitTimeout(c.options.connectTimeout) {
			c.options.logger.Error(ctx, ErrConnectTimeout, map[string]any{
				"clientID": pOpts.ClientID,
			})

			return ErrConnectTimeout
		}

		err := t.Error()
		if err != nil {
			c.options.logger.Error(ctx, err, map[string]any{"clientID": pOpts.ClientID})
		}

		return err
	})

	res := map[string]mqtt.Client{}
	anyConnected := false

	clients.Range(func(key, value interface{}) bool {
		// nolint: errcheck
		cc := value.(mqtt.Client)

		if cc.IsConnectionOpen() {
			anyConnected = true
		}

		// nolint: errcheck
		res[key.(string)] = cc

		return true
	})

	if !anyConnected {
		c.options.logger.Info(context.Background(), "no clients connected", map[string]any{
			"multiConnRevision": currRev,
		})

		go c.disconnectAll(xmap.Values(res))

		res = nil
	}

	if len(res) > 0 {
		if c.multiConnRevision.CompareAndSwap(currRev, currRev+1) {
			c.options.logger.Info(context.Background(), "multiConnRevision incremented", map[string]any{
				"old":               currRev,
				"new":               currRev + 1,
				"multiConnRevision": c.multiConnRevision.Load(),
			})
		}
	}

	return res
}

func (c *Client) newClient(addrs []TCPAddress, attempt int) mqtt.Client {
	addr := addrs[attempt%len(addrs)]

	opts := *c.options
	opts.brokerAddress = fmt.Sprintf("%s:%d", addr.Host, addr.Port)

	cc := newClientFunc.Load().(func(*mqtt.ClientOptions) mqtt.Client)(toClientOptions(c, &opts, ""))

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

func clientIDMapper(cc mqtt.Client) string {
	if cc == nil {
		return "<nil-client>"
	}

	r := cc.OptionsReader()

	if reflect.ValueOf(r).FieldByName("options").IsNil() {
		return "<nil-options>"
	}

	return r.ClientID()
}
