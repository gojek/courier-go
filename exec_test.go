package courier

import (
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
)

func TestOptionsImplementInterface(t *testing.T) {
	testcases := []struct {
		name string
		opt  any
	}{
		{
			name: "execOptConst",
			opt:  execOptConst(0),
		},
		{
			name: "execOptWithState",
			opt:  execOptWithState(nil),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			eo, ok := tc.opt.(execOpt)
			if !ok {
				t.Errorf("expected %T to implement execOpt", tc.opt)
			}
			eo.isExecOpt()
		})
	}
}

type invalidExecOpt bool

func (ieo invalidExecOpt) isExecOpt() {}

func TestClient_execMultiConn_invalidExecOption(t *testing.T) {
	assert.EqualError(t, new(Client).execMultiConn(func(mqtt.Client) error {
		return nil
	}, invalidExecOpt(true)), errInvalidExecOpt.Error())
}
