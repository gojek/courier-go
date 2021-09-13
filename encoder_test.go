package courier

import (
	"bytes"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_defaultEncoderFunc(t *testing.T) {
	validObj := struct {
		Key   string `json:"key"`
		Value int64  `json:"value"`
	}{
		Key:   "number",
		Value: 1e6,
	}
	invalidObj := math.Inf(1)

	tests := []struct {
		name string
		obj  interface{}
		want func() (bytes.Buffer, error)
	}{
		{
			name: "Success",
			obj:  validObj,
			want: func() (bytes.Buffer, error) {
				return *bytes.NewBufferString(`{"key":"number","value":1000000}` + "\n"), nil
			},
		},
		{
			name: "Failure",
			obj:  invalidObj,
			want: func() (bytes.Buffer, error) {
				return bytes.Buffer{}, errors.New("json.UnsupportedValueError")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantBuf, wantErr := tt.want()

			buf := bytes.Buffer{}
			j := defaultEncoderFunc(&buf)
			err := j.Encode(tt.obj)

			if wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, wantBuf.Bytes(), buf.Bytes())
		})
	}
}
