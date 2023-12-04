package courier

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockEmitter struct {
	mock.Mock
}

func newMockEmitter(t *testing.T) *mockEmitter {
	m := &mockEmitter{}
	m.Test(t)
	return m
}

func (m *mockEmitter) Emit(ctx context.Context, info MQTTClientInfo) { m.Called(ctx, info) }

func TestClient_ClientInfoEmitter(t *testing.T) {
	tests := []struct {
		name string
		mock func(*sync.WaitGroup, *mock.Mock)
		opts func(*testing.T) []ClientOption
	}{
		{
			name: "single connection mode",
			opts: func(t *testing.T) []ClientOption { return nil },
			mock: func(wg *sync.WaitGroup, m *mock.Mock) {
				wg.Add(1)

				m.On("Emit", mock.Anything, mock.Anything).Return().Run(func(args mock.Arguments) {
					wg.Done()
				}).Once()
			},
		},
		{
			name: "multi connection mode",
			opts: func(t *testing.T) []ClientOption {
				ch := make(chan []TCPAddress, 1)
				dCh := make(chan struct{})
				mr := newMockResolver(t)

				mr.On("UpdateChan").Return(ch)
				mr.On("Done").Return(dCh)

				go func() {
					ch <- []TCPAddress{testBrokerAddress, testBrokerAddress}

					<-time.After(2 * time.Second)
					close(ch)

					dCh <- struct{}{}
				}()

				return []ClientOption{
					WithResolver(mr),
					UseMultiConnectionMode,
				}
			},
			mock: func(wg *sync.WaitGroup, m *mock.Mock) {
				wg.Add(2)

				m.On("Emit", mock.Anything, mock.Anything).Return().Run(func(args mock.Arguments) {
					wg.Done()
				}).Twice()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eOpts := tt.opts(t)
			mc := newMockEmitter(t)
			eOpts = append(eOpts, &ClientInfoEmitterConfig{
				Interval: time.Second,
				Emitter:  mc,
			})

			wg := &sync.WaitGroup{}
			if tt.mock != nil {
				tt.mock(wg, &mc.Mock)
			}

			c, err := NewClient(append(defOpts, eOpts...)...)
			assert.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				_ = c.Run(ctx)
			}()

			assert.True(t, WaitForConnection(c, 2*time.Second, 100*time.Millisecond))

			wg.Wait()

			cancel()

			mc.AssertExpectations(t)
		})
	}

	t.Run("NewClientWithLessThanOneSecondEmitterIntervalError", func(t *testing.T) {
		_, err := NewClient(append(defOpts, &ClientInfoEmitterConfig{
			Interval: 100 * time.Millisecond,
			Emitter:  newMockEmitter(t),
		})...)
		assert.EqualError(t, err, "client info emitter interval must be greater than or equal to 1s")
	})
}
