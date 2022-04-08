package xds

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/gojekfarm/courier-go"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/mock"
	"reflect"
	"testing"
)

func TestNewResolver(t *testing.T) {
	rc := &mockReceiver{mock.Mock{}}
	done := make(chan struct{})
	close(done)

	receiveChan := make(chan []*v3endpointpb.ClusterLoadAssignment)
	close(receiveChan)

	rc.On("Done").Return(done)
	rc.On("Receive").Return(receiveChan)

	tests := []struct {
		name string
		want *Resolver
	}{
		{
			name: "success",
			want: &Resolver{
				rc: rc,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if got := NewResolver(rc); !reflect.DeepEqual(got.rc, tt.want.rc) {
				t.Errorf("NewResolver() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_Done(t *testing.T) {
	rc := &mockReceiver{mock.Mock{}}
	done := make(chan struct{})
	rc.On("Done").Return(done)
	r := &Resolver{
		rc: rc,
	}

	ch := r.Done()
	var a interface{}

	go func() {
		a = <-ch
	}()
	done <- struct{}{}

	if a != struct{}{} {
		t.Errorf("Done(), no value received")
	}
}

func TestResolver_UpdateChan(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		rc := &mockReceiver{mock.Mock{}}
		r := &Resolver{
			rc: rc,
			ch: make(chan []courier.TCPAddress),
		}

		go func() {
			r.ch <- []courier.TCPAddress{{
				Host: "host_2",
				Port: 1,
			},
				{
					Host: "host_2",
					Port: 2,
				}}
		}()

		got := r.UpdateChan()
		val := <-got
		if !reflect.DeepEqual(val, []courier.TCPAddress{{
			Host: "host_2",
			Port: 1,
		},
			{
				Host: "host_2",
				Port: 2,
			}}) {
			t.Errorf("UpdateChan() expected value not received from chan")
		}
	})
}

func TestResolver_Run(t *testing.T) {
	tests := []struct {
		name             string
		done             bool
		mockReceiverFunc func(chan []*v3endpointpb.ClusterLoadAssignment, chan struct{}) *mockReceiver
		update           []*v3endpointpb.ClusterLoadAssignment
		address          []courier.TCPAddress
	}{
		{
			name: "received_on_done_chan",
			mockReceiverFunc: func(updates chan []*v3endpointpb.ClusterLoadAssignment, done chan struct{}) *mockReceiver {
				m := &mockReceiver{mock.Mock{}}
				close(done)

				m.On("Done").Return(done)
				m.On("Receive").Return(updates)
				return m
			},
			done: true,
		},
		{
			name: "received_on_update_chan",
			mockReceiverFunc: func(updates chan []*v3endpointpb.ClusterLoadAssignment, done chan struct{}) *mockReceiver {
				m := &mockReceiver{mock.Mock{}}
				m.On("Receive").Return(updates)
				m.On("Done").Return(done)

				return m
			},
			update: []*v3endpointpb.ClusterLoadAssignment{{
				ClusterName: "xds:///broker.domain",
				Endpoints: []*v3endpointpb.LocalityLbEndpoints{{
					LoadBalancingWeight: &wrappers.UInt32Value{Value: 1},
					LbEndpoints: []*v3endpointpb.LbEndpoint{{
						HostIdentifier: &v3endpointpb.LbEndpoint_Endpoint{
							Endpoint: &v3endpointpb.Endpoint{
								Address: &core.Address{
									Address: &core.Address_SocketAddress{
										SocketAddress: &core.SocketAddress{
											Protocol: core.SocketAddress_TCP,
											Address:  "localhost",
											PortSpecifier: &core.SocketAddress_PortValue{
												PortValue: 1883,
											},
										},
									},
								},
							},
						},
					}},
				},
					{
						LoadBalancingWeight: &wrappers.UInt32Value{Value: 2},
						LbEndpoints: []*v3endpointpb.LbEndpoint{{
							HostIdentifier: &v3endpointpb.LbEndpoint_Endpoint{
								Endpoint: &v3endpointpb.Endpoint{
									Address: &core.Address{
										Address: &core.Address_SocketAddress{
											SocketAddress: &core.SocketAddress{
												Protocol: core.SocketAddress_TCP,
												Address:  "localhost",
												PortSpecifier: &core.SocketAddress_PortValue{
													PortValue: 8888,
												},
											},
										},
									},
								},
							},
						}},
					},
				},
			},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan []*v3endpointpb.ClusterLoadAssignment)
			doneChan := make(chan struct{})
			receiver := tt.mockReceiverFunc(ch, doneChan)
			r := NewResolver(receiver)

			go func() {
				if tt.update != nil {
					ch <- tt.update
					close(doneChan)
				}
			}()

			var done bool
			var update []courier.TCPAddress

			select {
			case <-r.Done():
				done = true
			case update = <-r.UpdateChan():
			}

			if done != tt.done {
				t.Errorf("run() returned on done channel")
			}

			if !done && !reflect.DeepEqual(update, []courier.TCPAddress{{
				Host: "localhost",
				Port: 1883,
			},
			{
				Host: "localhost",
				Port: 8888,
			}}) {
				t.Errorf("UpdateChan() expected value not received from chan")
			}
		})

	}
}

type mockReceiver struct {
	mock.Mock
}

func (m *mockReceiver) Receive() <-chan []*v3endpointpb.ClusterLoadAssignment {
	return m.Called().Get(0).(chan []*v3endpointpb.ClusterLoadAssignment)
}

func (m *mockReceiver) Done() <-chan struct{} {
	return m.Called().Get(0).(chan struct{})
}