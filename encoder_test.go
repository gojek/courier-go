package courier

import (
	"bytes"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/suite"
)

type jsonEncoderSuite struct {
	suite.Suite
}

func Test_jsonEncoderSuite(t *testing.T) {
	suite.Run(t, new(jsonEncoderSuite))
}

func (s *jsonEncoderSuite) TestEncode() {
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
		s.Run(tt.name, func() {
			wantBuf, wantErr := tt.want()

			buf := bytes.Buffer{}
			j := jsonEncoder{w: &buf}
			err := j.Encode(tt.obj)

			if wantErr != nil {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			s.Equal(wantBuf.Bytes(), buf.Bytes())
		})
	}
}
