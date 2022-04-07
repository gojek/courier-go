package courier

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"math/rand"
	"time"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

type TCPAddress struct {
	Host string
	Port uint16
}

type Resolver interface {
	UpdateChan() <-chan []TCPAddress
	Done() <-chan struct{}
}

func WithResolver(resolver Resolver) ClientOption {
	return func(c *clientOptions) {
		c.resolver = resolver
	}
}

func (c *Client) watchAddressUpdates(r Resolver) {
	for {
		select {
		case <-r.Done():
			return
		case addrs := <-r.UpdateChan():
			fmt.Printf("New address received: %v \n", addrs)
			if len(addrs) == 0 {
				break
			}
			// try to start new client first, iff it starts, replace current client
			cc := c.newClient(addrs)
			c.reloadClient(cc)
		}
	}
}

func (c *Client) newClient(addrs []TCPAddress) mqtt.Client {
	addr := addrs[rnd.Intn(len(addrs))]

	c.options.brokerAddress = fmt.Sprintf("tcp://%s:%d", addr.Host, addr.Port)
	cc := newClientFunc(toClientOptions(c, c.options))

	t := cc.Connect()
	if !t.WaitTimeout(c.options.connectTimeout) {
		return c.newClient(addrs)
	}

	if err := t.Error(); err != nil {
		// TODO: add retry backoff or use ExponentialStartStrategy utility
		return c.newClient(addrs)
	}

	return cc
}

func (c *Client) reloadClient(cc mqtt.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldClient := c.mqttClient
	c.mqttClient = cc

	go func() {
		oldClient.Disconnect(uint(c.options.gracefulShutdownPeriod / time.Millisecond))
	}()
}
