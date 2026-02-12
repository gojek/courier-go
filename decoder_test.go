package courier

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type errorDecoder struct {
	err error
}

func (fd *errorDecoder) Decode(v interface{}) error {
	return fd.err
}

type jsonDecoderSuite struct {
	suite.Suite
}

func Test_jsonDecoderSuite(t *testing.T) {
	suite.Run(t, new(jsonDecoderSuite))
}

const data = `{"key":"value"}`

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

func TestChainDecoder_FirstSuccess(t *testing.T) {
	ctx := context.Background()
	reader := strings.NewReader(data)

	decoder1 := func(_ context.Context, r io.Reader) Decoder {
		return json.NewDecoder(r)
	}
	decoder2 := func(_ context.Context, r io.Reader) Decoder {
		return &errorDecoder{err: errors.New("should not be called")}
	}

	chain := ChainDecoderFunc(decoder1, decoder2)
	dec := chain(ctx, reader)

	var result map[string]string
	if err := dec.Decode(&result); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("expected value 'value', got %q", result["key"])
	}
}

func TestChainDecoder_SecondSuccess(t *testing.T) {
	ctx := context.Background()
	reader := strings.NewReader(data)

	decoder1 := func(_ context.Context, r io.Reader) Decoder {
		return &errorDecoder{err: errors.New("first decoder fails")}
	}
	decoder2 := func(_ context.Context, r io.Reader) Decoder {
		return json.NewDecoder(r)
	}

	chain := ChainDecoderFunc(decoder1, decoder2)
	dec := chain(ctx, reader)

	var result map[string]string
	if err := dec.Decode(&result); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("expected value 'value', got %q", result["key"])
	}
}

func TestChainDecoder_AllFail(t *testing.T) {
	ctx := context.Background()
	reader := strings.NewReader(data)

	err1 := errors.New("first fails")
	err2 := errors.New("second fails")
	err3 := errors.New("third fails")

	decoder1 := func(_ context.Context, r io.Reader) Decoder {
		return &errorDecoder{err: err1}
	}
	decoder2 := func(_ context.Context, r io.Reader) Decoder {
		return &errorDecoder{err: err2}
	}
	decoder3 := func(_ context.Context, r io.Reader) Decoder {
		return &errorDecoder{err: err3}
	}

	chain := ChainDecoderFunc(decoder1, decoder2, decoder3)
	dec := chain(ctx, reader)

	var result map[string]string
	err := dec.Decode(&result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "all decoders failed") {
		t.Errorf("expected error to contain 'all decoders failed', got: %v", err)
	}
}
