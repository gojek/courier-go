package courier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type errorEncoder struct {
	err error
}

func (fe *errorEncoder) Encode(v interface{}) error {
	return fe.err
}

type errorDecoder struct {
	err error
}

func (fd *errorDecoder) Decode(v interface{}) error {
	return fd.err
}

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
			j := DefaultEncoderFunc(context.TODO(), &buf)
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

func TestFallbackEncoder_FirstSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := context.Background()

	encoder1 := func(_ context.Context, w io.Writer) Encoder {
		return json.NewEncoder(w)
	}
	encoder2 := func(_ context.Context, w io.Writer) Encoder {
		return &errorEncoder{err: errors.New("should not be called")}
	}

	fallback := FallbackEncoderFunc(encoder1, encoder2)
	enc := fallback(ctx, buf)

	data := map[string]string{"key": "value"}
	if err := enc.Encode(data); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := `{"key":"value"}` + "\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestFallbackEncoder_SecondSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := context.Background()

	encoder1 := func(_ context.Context, w io.Writer) Encoder {
		return &errorEncoder{err: errors.New("first encoder fails")}
	}
	encoder2 := func(_ context.Context, w io.Writer) Encoder {
		return json.NewEncoder(w)
	}

	fallback := FallbackEncoderFunc(encoder1, encoder2)
	enc := fallback(ctx, buf)

	data := map[string]string{"key": "value"}
	if err := enc.Encode(data); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := `{"key":"value"}` + "\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestFallbackEncoder_AllFail(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := context.Background()

	err1 := errors.New("first fails")
	err2 := errors.New("second fails")
	err3 := errors.New("third fails")

	encoder1 := func(_ context.Context, w io.Writer) Encoder {
		return &errorEncoder{err: err1}
	}
	encoder2 := func(_ context.Context, w io.Writer) Encoder {
		return &errorEncoder{err: err2}
	}
	encoder3 := func(_ context.Context, w io.Writer) Encoder {
		return &errorEncoder{err: err3}
	}

	fallback := FallbackEncoderFunc(encoder1, encoder2, encoder3)
	enc := fallback(ctx, buf)

	data := map[string]string{"key": "value"}
	err := enc.Encode(data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "all encoders failed") {
		t.Errorf("expected error to contain 'all encoders failed', got: %v", err)
	}
}
