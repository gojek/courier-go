package updatehandler

import (
	"github.com/gojekfarm/courier-go/xds/types"
	"reflect"
	"sync"
	"testing"
)

func Test_compareSlices(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{
			name: "identical_slices",
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "b", "c"},
			want: true,
		},
		{
			name: "different_length",
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "b"},
		},
		{
			name: "same_elements_different_arrangement",
			a:    []string{"a", "b", "c"},
			b:    []string{"b", "c", "a"},
			want: true,
		},
		{
			name: "nil_slice",
			a:    nil,
			b:    []string{"b", "c", "a"},
		},
		{
			name: "different_slices",
			a:    []string{"b", "c"},
			b:    []string{"x", "y"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareSlices(tt.a, tt.b); got != tt.want {
				t.Errorf("compareSlices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_endpointUpdateHandler_AddEndpointWatcher(t *testing.T) {
	watchFunc := func(strings []string) {}
	tests := []struct {
		name    string
		cfg     Config
		watcher types.EndpointWatcher
	}{
		{
			name: "add_endpoint_success_when_endpoint_not_present",
			cfg:  Config{},
			watcher: types.EndpointWatcher{
				Endpoint: "cluster",
				Callback: func(strings []string) {},
			},
		},
		{
			name: "add_endpoint_success_when_endpoint_present",
			cfg: Config{
				Epw: []types.EndpointWatcher{
					{
						"cluster", watchFunc,
					},
				},
			},
			watcher: types.EndpointWatcher{
				Endpoint: "cluster",
				Callback: func(strings []string) {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(tt.cfg)
			h.AddEndpointWatcher(tt.watcher)

			if h.(*endpointUpdateHandler).callbackMap[tt.watcher.Endpoint] == nil {
				t.Errorf("endpoint not added")
				return
			}

			if !reflect.DeepEqual(h.(*endpointUpdateHandler).callbackMap[tt.watcher.Endpoint], tt.watcher.Callback) {
				t.Errorf("wrong endpoint added")
			}
		})
	}
}

func Test_endpointUpdateHandler_NewConnectionError(t *testing.T) {
	type fields struct {
		endpoints       map[string][]string
		callbackMap     map[string]types.CallbackFunc
		connErrCallback func(error)
		mu              sync.Mutex
	}
	type args struct {
		err error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{

		}
	},
		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//h := &endpointUpdateHandler{
			//	endpoints:       tt.fields.endpoints,
			//	callbackMap:     tt.fields.callbackMap,
			//	connErrCallback: tt.fields.connErrCallback,
			//	mu:              tt.fields.mu,
			//}
		})
	}
}

func Test_endpointUpdateHandler_NewEndpoints(t *testing.T) {
	type fields struct {
		endpoints       map[string][]string
		callbackMap     map[string]types.CallbackFunc
		connErrCallback func(error)
		mu              sync.Mutex
	}
	type args struct {
		endpoints map[string][]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//h := &endpointUpdateHandler{
			//	endpoints:       tt.fields.endpoints,
			//	callbackMap:     tt.fields.callbackMap,
			//	connErrCallback: tt.fields.connErrCallback,
			//	mu:              tt.fields.mu,
			//}
		})
	}
}

func Test_endpointUpdateHandler_RemoveEndpointWatcher(t *testing.T) {
	type fields struct {
		endpoints       map[string][]string
		callbackMap     map[string]types.CallbackFunc
		connErrCallback func(error)
		mu              sync.Mutex
	}
	type args struct {
		watcher types.EndpointWatcher
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//h := &endpointUpdateHandler{
			//	endpoints:       tt.fields.endpoints,
			//	callbackMap:     tt.fields.callbackMap,
			//	connErrCallback: tt.fields.connErrCallback,
			//	mu:              tt.fields.mu,
			//}
		})
	}
}

func Test_endpointUpdateHandler_getDiff(t *testing.T) {
	type fields struct {
		endpoints       map[string][]string
		callbackMap     map[string]types.CallbackFunc
		connErrCallback func(error)
		mu              sync.Mutex
	}
	type args struct {
		updatedEndpoints map[string][]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string][]string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &endpointUpdateHandler{
				endpoints:       tt.fields.endpoints,
				callbackMap:     tt.fields.callbackMap,
				connErrCallback: tt.fields.connErrCallback,
				mu:              tt.fields.mu,
			}
			if got := h.getDiff(tt.args.updatedEndpoints); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDiff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_endpointUpdateHandler_initiateEndpoints(t *testing.T) {
	type fields struct {
		endpoints       map[string][]string
		callbackMap     map[string]types.CallbackFunc
		connErrCallback func(error)
		mu              sync.Mutex
	}
	type args struct {
		epw []types.EndpointWatcher
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//h := &endpointUpdateHandler{
			//	endpoints:       tt.fields.endpoints,
			//	callbackMap:     tt.fields.callbackMap,
			//	connErrCallback: tt.fields.connErrCallback,
			//	mu:              tt.fields.mu,
			//}
		})
	}
}
