package courier

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestClient_AddressUpdates(t *testing.T) {
	testBrokerAddress2 := TCPAddress{
		Host: testBrokerAddress.Host,
		Port: testBrokerAddress.Port + 1,
	}
	subscribeFunc := func(ctx context.Context, _ PubSub, msg *Message) {
		// do nothing
	}
	gsp := defaultClientOptions().gracefulShutdownPeriod

	newMockClientTokenWithDisconnect := func(t *testing.T, o *mqtt.ClientOptions) (*mockClient, *mockToken) {
		m := newMockClient(t)
		tkn := newMockToken(t)

		tkn.On("WaitTimeout", o.ConnectTimeout).Return(true)
		tkn.On("Error").Return(nil)
		m.On("Connect").Return(tkn).Once()
		m.On("Disconnect", uint(gsp/time.Millisecond)).Return().Once()

		return m, tkn
	}
	storeNewMocks := func(mckCh chan any, mocks ...any) {
		for _, mck := range mocks {
			mckCh <- mck
		}
	}
	aggregateMocks := func(mckCh chan any) []any {
		wg := &sync.WaitGroup{}
		var mcks []any
		wg.Add(1)
		go func() {
			defer wg.Done()
			close(mckCh)

			for mck := range mckCh {
				mcks = append(mcks, mck)
			}
		}()

		wg.Wait()

		return mcks
	}

	t.Run("ResubscribesWhenAddressUpdateHappensInSingleConnectionMode", func(t *testing.T) {
		r := newMockResolver(t)
		ch := make(chan []TCPAddress)
		r.On("UpdateChan").Return(ch)
		doneCh := make(chan struct{})
		r.On("Done").Return(doneCh)

		go func() { ch <- []TCPAddress{testBrokerAddress, testBrokerAddress2} }()

		mckCh := make(chan any, 6)

		c, err := NewClient(WithResolver(r))
		assert.NoError(t, err)

		newClientFunc.Store(func(o *mqtt.ClientOptions) mqtt.Client {
			fmt.Println("newClientFunc")

			m, tkn := newMockClientTokenWithDisconnect(t, o)
			stk := newMockToken(t)

			stk.On("WaitTimeout", 10*time.Second).Return(true)
			stk.On("Error").Return(nil)

			m.On("Subscribe", "topic1", byte(1), mock.AnythingOfType("mqtt.MessageHandler")).
				Return(stk).
				Once()

			storeNewMocks(mckCh, tkn, m, stk)

			return m
		})
		defer newClientFunc.Store(mqtt.NewClient)

		assert.NoError(t, c.Start())

		assert.NoError(t, c.Subscribe(context.Background(), "topic1", subscribeFunc, QOSOne))

		prevAddr := fmt.Sprintf("%p", c.mqttClient.(*mockClient))

		go func() { ch <- []TCPAddress{testBrokerAddress, testBrokerAddress2} }()

		assert.Eventually(t, func() bool {
			c.clientMu.RLock()
			defer c.clientMu.RUnlock()

			return prevAddr != fmt.Sprintf("%p", c.mqttClient.(*mockClient))
		}, time.Second, 100*time.Millisecond)

		mcks := aggregateMocks(mckCh)
		assert.NoError(t, c.stop())
		require.Len(t, mcks, 6)
		mock.AssertExpectationsForObjects(t, mcks...)

		close(doneCh)
	})

	t.Run("ReSubscribesToSharedSubscriptionsWhenUpdateHappens", func(t *testing.T) {
		// given: 2 shared subscriptions are created, 2 clients are created and subscribed
		// to the same shared subscriptions initially
		r := newMockResolver(t)
		ch := make(chan []TCPAddress)
		r.On("UpdateChan").Return(ch)

		mckCh := make(chan any, 12)

		go func() { ch <- []TCPAddress{testBrokerAddress, testBrokerAddress2} }()

		doneCh := make(chan struct{})
		r.On("Done").Return(doneCh)

		c, err := NewClient(WithResolver(r), MultiConnectionMode)
		assert.NoError(t, err)

		newClientFunc.Store(func(o *mqtt.ClientOptions) mqtt.Client {
			m, tkn := newMockClientTokenWithDisconnect(t, o)
			stk := newMockToken(t)

			stk.On("WaitTimeout", 10*time.Second).Return(true)
			stk.On("Error").Return(nil)

			m.On("SubscribeMultiple", map[string]byte{
				"$share/group1/topic1": 1,
				"$share/group1/topic2": 2,
			}, mock.AnythingOfType("mqtt.MessageHandler")).Return(stk).Once()
			m.On("Subscribe",
				"$share/group1/topic3",
				byte(1),
				mock.AnythingOfType("mqtt.MessageHandler"),
			).Return(stk).Once()

			storeNewMocks(mckCh, tkn, m, stk)

			return m
		})
		defer newClientFunc.Store(mqtt.NewClient)

		assert.NoError(t, c.Start())

		assert.NoError(t, c.SubscribeMultiple(context.Background(), map[string]QOSLevel{
			"$share/group1/topic1": QOSOne,
			"$share/group1/topic2": QOSTwo,
		}, subscribeFunc))
		assert.NoError(t, c.Subscribe(context.Background(), "$share/group1/topic3", subscribeFunc, QOSOne))

		// when: the resolver updates the addresses

		newClientFunc.Store(func(o *mqtt.ClientOptions) mqtt.Client {
			m, tkn := newMockClientTokenWithDisconnect(t, o)
			stk := newMockToken(t)

			stk.On("WaitTimeout", 10*time.Second).Return(true)
			stk.On("Error").Return(nil)

			m.On("Subscribe",
				"$share/group1/topic1",
				byte(QOSOne),
				mock.AnythingOfType("mqtt.MessageHandler"),
			).Return(stk).Once()
			m.On("Subscribe",
				"$share/group1/topic2",
				byte(QOSTwo),
				mock.AnythingOfType("mqtt.MessageHandler"),
			).Return(stk).Once()
			m.On("Subscribe",
				"$share/group1/topic3",
				byte(QOSOne),
				mock.AnythingOfType("mqtt.MessageHandler"),
			).Return(stk).Once()

			storeNewMocks(mckCh, tkn, m, stk)

			return m
		})

		go func() { ch <- []TCPAddress{testBrokerAddress, testBrokerAddress2} }()

		wantAddrs := []string{testBrokerAddress.String(), testBrokerAddress2.String()}
		assert.Eventually(t, func() bool {
			c.clientMu.RLock()
			defer c.clientMu.RUnlock()

			var actual []string
			for addr := range c.mqttClients {
				actual = append(actual, addr)
			}

			return assert.ElementsMatch(t, wantAddrs, actual)
		}, time.Second, 100*time.Millisecond)

		mcks := aggregateMocks(mckCh)

		// then: the client that has the address change should resubscribe to the shared subscriptions
		assert.NoError(t, c.stop())
		require.Len(t, mcks, 12)
		mock.AssertExpectationsForObjects(t, mcks...)

		close(doneCh)
	})

	t.Run("ReSubscribesToNormalSubscriptionsRandomlyWhenUpdateHappens", func(t *testing.T) {
		// given 2 normal subscriptions are created, 2 clients are created and subscribed
		r := newMockResolver(t)
		ch := make(chan []TCPAddress)
		r.On("UpdateChan").Return(ch)

		mckCh := make(chan any, 12)

		go func() { ch <- []TCPAddress{testBrokerAddress, testBrokerAddress2} }()

		doneCh := make(chan struct{})
		r.On("Done").Return(doneCh)

		c, err := NewClient(WithResolver(r), MultiConnectionMode)
		assert.NoError(t, err)

		subCallCh := make(chan string, 4)
		topicSubscribed := make([]string, 0, 4)

		newClientFunc.Store(func(o *mqtt.ClientOptions) mqtt.Client {
			m, tkn := newMockClientTokenWithDisconnect(t, o)
			stk := newMockToken(t)

			stk.On("WaitTimeout", 10*time.Second).Maybe().Return(true)
			stk.On("Error").Maybe().Return(nil)
			m.On("Subscribe",
				"topic1",
				byte(1),
				mock.AnythingOfType("mqtt.MessageHandler"),
			).Return(stk).Maybe().Run(func(args mock.Arguments) {
				subCallCh <- "topic1-client1"
			})
			m.On("Subscribe",
				"topic2",
				byte(2),
				mock.AnythingOfType("mqtt.MessageHandler"),
			).Return(stk).Maybe().Run(func(args mock.Arguments) {
				subCallCh <- "topic2-client1"
			})

			storeNewMocks(mckCh, tkn, m, stk)

			return m
		})
		defer newClientFunc.Store(mqtt.NewClient)

		assert.NoError(t, c.Start())

		assert.NoError(t, c.Subscribe(context.Background(), "topic1", subscribeFunc, QOSOne))
		assert.NoError(t, c.Subscribe(context.Background(), "topic2", subscribeFunc, QOSTwo))

		// when: the resolver updates the addresses

		newClientFunc.Store(func(o *mqtt.ClientOptions) mqtt.Client {
			m, tkn := newMockClientTokenWithDisconnect(t, o)
			stk := newMockToken(t)

			stk.On("WaitTimeout", 10*time.Second).Maybe().Return(true)
			stk.On("Error").Maybe().Return(nil)
			m.On("Subscribe",
				"topic1",
				byte(1),
				mock.AnythingOfType("mqtt.MessageHandler"),
			).Return(stk).Maybe().Run(func(args mock.Arguments) {
				subCallCh <- "topic1-client2"
			})
			m.On("Subscribe",
				"topic2",
				byte(2),
				mock.AnythingOfType("mqtt.MessageHandler"),
			).Return(stk).Maybe().Run(func(args mock.Arguments) {
				subCallCh <- "topic2-client2"
			})

			storeNewMocks(mckCh, tkn, m, stk)

			return m
		})

		go func() { ch <- []TCPAddress{testBrokerAddress, testBrokerAddress2} }()

		wantAddrs := []string{testBrokerAddress.String(), testBrokerAddress2.String()}
		assert.Eventually(t, func() bool {
			c.clientMu.RLock()
			defer c.clientMu.RUnlock()

			var actual []string
			for addr := range c.mqttClients {
				actual = append(actual, addr)
			}

			return assert.ElementsMatch(t, wantAddrs, actual)
		}, time.Second, 100*time.Millisecond)

		for i := 0; i < 2; i++ {
			select {
			case <-time.After(time.Second):
				t.Fatal("timeout")
			case topic := <-subCallCh:
				topicSubscribed = append(topicSubscribed, topic)
			}
		}

		// then: the client that has the address change should resubscribe to the shared subscriptions

		for i := 0; i < 2; i++ {
			select {
			case <-time.After(time.Second):
				t.Fatal("timeout")
			case topic := <-subCallCh:
				topicSubscribed = append(topicSubscribed, topic)
			}
		}

		assert.NoError(t, c.stop())
		mcks := aggregateMocks(mckCh)
		require.Len(t, topicSubscribed, 4)
		assert.ElementsMatch(t, []string{"topic1-client1", "topic2-client1", "topic1-client2", "topic2-client2"}, topicSubscribed)
		require.Len(t, mcks, 12)
		mock.AssertExpectationsForObjects(t, mcks...)

		close(doneCh)
	})
}

func Test_accumulateErrors(t *testing.T) {
	tests := []struct {
		name    string
		prev    error
		curr    error
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "no_errors",
			wantErr: assert.NoError,
		},
		{
			name: "prev_error",
			prev: errors.New("prev error"),
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.EqualError(t, err, `1 error occurred: [prev error]`)
			},
		},
		{
			name: "curr_error",
			curr: errors.New("curr error"),
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.EqualError(t, err, `1 error occurred: [curr error]`)
			},
		},
		{
			name: "prev_and_curr_error",
			prev: errors.New("prev error"),
			curr: errors.New("curr error"),
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.EqualError(t, err, `2 errors occurred: [prev error] | [curr error]`)
			},
		},
		{
			name: "prev_multierrors",
			prev: multierror.Append(errors.New("prev error"), errors.New("prev error 2")),
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.EqualError(t, err, `2 errors occurred: [prev error] | [prev error 2]`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, accumulateErrors(tt.prev, tt.curr), fmt.Sprintf("accumulateErrors(%v, %v)", tt.prev, tt.curr))
		})
	}
}
