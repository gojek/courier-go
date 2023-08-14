package courier

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestClient_newClient(t *testing.T) {
	tests := []struct {
		name             string
		addrs            []TCPAddress
		newClientFunc    func(*mqtt.ClientOptions) mqtt.Client
		onConnLostAssert func(*testing.T, error)
	}{
		{
			name: "success_attempt_1",
			addrs: []TCPAddress{
				{
					Host: "localhost",
					Port: 1883,
				},
				{
					Host: "localhost",
					Port: 8888,
				},
			},
			newClientFunc: func(o *mqtt.ClientOptions) mqtt.Client {
				if !reflect.DeepEqual(o.Servers[0].String(), "tcp://localhost:1883") {
					panic(o.Servers)
				}

				m := &mockClient{}
				tkn := &mockToken{}
				tkn.On("WaitTimeout", o.ConnectTimeout).Return(true)
				tkn.On("Error").Return(nil)
				m.On("Connect").Return(tkn)
				return m
			},
		},
		{
			name: "success_attempt_2",
			addrs: []TCPAddress{
				{
					Host: "localhost",
					Port: 1883,
				},
				{
					Host: "localhost",
					Port: 8888,
				},
			},
			newClientFunc: func(o *mqtt.ClientOptions) mqtt.Client {
				if o.Servers[0].String() != "tcp://localhost:1883" {
					panic(o.Servers)
				}

				m := &mockClient{}
				tkn1 := &mockToken{}
				tkn1.On("WaitTimeout", o.ConnectTimeout).Return(false)

				m.On("Connect").Return(tkn1).Run(func(args mock.Arguments) {
					newClientFunc.Store(func(o *mqtt.ClientOptions) mqtt.Client {
						if o.Servers[0].String() != "tcp://localhost:8888" {
							panic(o.Servers[0].String())
						}
						tkn2 := &mockToken{}
						tkn2.On("WaitTimeout", o.ConnectTimeout).Return(true)
						tkn2.On("Error").Return(nil)
						m.On("Connect").Return(tkn2).Once()

						return m
					})
				}).Once()

				return m
			},
		},
		{
			name: "token_error",
			addrs: []TCPAddress{
				{
					Host: "localhost",
					Port: 1883,
				},
				{
					Host: "localhost",
					Port: 8888,
				},
			},
			onConnLostAssert: func(t *testing.T, err error) {
				assert.EqualError(t, err, "some error")
			},
			newClientFunc: func(o *mqtt.ClientOptions) mqtt.Client {
				if o.Servers[0].String() != "tcp://localhost:1883" {
					panic(o.Servers)
				}

				m := &mockClient{}
				tkn1 := &mockToken{}
				tkn1.On("WaitTimeout", o.ConnectTimeout).Return(true)
				tkn1.On("Error").Return(errors.New("some error"))

				m.On("Connect").Return(tkn1).Run(func(args mock.Arguments) {
					newClientFunc.Store(func(o *mqtt.ClientOptions) mqtt.Client {
						if o.Servers[0].String() != "tcp://localhost:8888" {
							panic(o.Servers[0].String())
						}
						tkn2 := &mockToken{}
						tkn2.On("WaitTimeout", o.ConnectTimeout).Return(true)
						tkn2.On("Error").Return(nil)
						m.On("Connect").Return(tkn2).Once()

						return m
					})
				}).Once()

				return m
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultClientOptions()
			if tt.onConnLostAssert != nil {
				opts.onConnectionLostHandler = func(err error) {
					tt.onConnLostAssert(t, err)
				}
			}

			c := &Client{options: opts}
			newClientFunc.Store(tt.newClientFunc)
			got := c.newClient(tt.addrs, 0)

			got.(*mockClient).AssertExpectations(t)
		})
	}

	newClientFunc.Store(mqtt.NewClient)
}

func TestClient_watchAddressUpdates(t *testing.T) {
	tests := []struct {
		name          string
		sender        func(chan []TCPAddress, chan struct{}, *sync.WaitGroup)
		closeDoneChan bool
	}{
		{
			name: "close_on_done_chan",
			sender: func(c chan []TCPAddress, c2 chan struct{}, wg *sync.WaitGroup) {
				wg.Done()
				close(c2)
			},
			closeDoneChan: true,
		},
		{
			name: "update_sent",
			sender: func(c chan []TCPAddress, c2 chan struct{}, wg *sync.WaitGroup) {
				c <- []TCPAddress{{
					Host: "localhost",
					Port: 8888,
				}}
				wg.Done()
			},
		},
		{
			name: "empty_update_sent",
			sender: func(c chan []TCPAddress, c2 chan struct{}, wg *sync.WaitGroup) {
				c <- []TCPAddress{}
				wg.Done()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newClient := &mockClient{}

			oldClient := &mockClient{}
			oldClient.On("Disconnect", uint(defaultClientOptions().gracefulShutdownPeriod/time.Millisecond)).Return()

			c := &Client{
				options:    defaultClientOptions(),
				mqttClient: oldClient,
			}
			uc := make(chan []TCPAddress)
			dc := make(chan struct{})
			r := &resolver{
				updateChan: uc,
				doneChan:   dc,
			}

			wg := &sync.WaitGroup{}
			wg.Add(1)
			go tt.sender(uc, dc, wg)

			newClientFunc.Store(func(o *mqtt.ClientOptions) mqtt.Client {
				tkn1 := &mockToken{}
				tkn1.On("WaitTimeout", o.ConnectTimeout).Return(true)
				tkn1.On("Error").Return(nil)
				newClient.On("Connect").Return(tkn1).Once()
				return newClient
			})

			go c.watchAddressUpdates(r)

			wg.Wait()
			if !tt.closeDoneChan {
				close(dc)
				close(uc)
			}

			newClient.AssertExpectations(t)
		})
	}
}

type resolver struct {
	updateChan chan []TCPAddress
	doneChan   chan struct{}
}

func (r resolver) UpdateChan() <-chan []TCPAddress {
	return r.updateChan
}

func (r resolver) Done() <-chan struct{} {
	return r.doneChan
}
