package courier

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type jsonDecoderSuite struct {
	suite.Suite
}

func Test_jsonDecoderSuite(t *testing.T) {
	suite.Run(t, new(jsonDecoderSuite))
}

func (s *jsonDecoderSuite) TestDecode() {
	type obj struct {
		Key   string `json:"key"`
		Value int64  `json:"value"`
	}

	tests := []struct {
		name    string
		payload string
		result  interface{}
		wantErr bool
	}{
		{
			name:    "Success",
			payload: `{"key":"number","value":1000000}`,
			result:  obj{},
		},
		{
			name:    "Failure",
			payload: "payload",
			result:  map[string]interface{}{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := DefaultDecoderFunc(context.TODO(), strings.NewReader(tt.payload))

			err := d.Decode(&tt.result)

			if tt.wantErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *jsonDecoderSuite) TestDecodeWithBase64() {
	type obj struct {
		Key   string `json:"key"`
		Value int64  `json:"value"`
	}

	tests := []struct {
		name    string
		payload string
		result  interface{}
		wantErr bool
	}{
		{
			name:    "Success",
			payload: "eyJrZXkiOiJudW1iZXIiLCJ2YWx1ZSI6MTAwMDAwMH0K",
			result:  &obj{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := base64JsonDecoder(context.TODO(), strings.NewReader(tt.payload))

			err := d.Decode(&tt.result)

			if tt.wantErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}
