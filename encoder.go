package courier

import (
	"context"
	"encoding/json"
	"io"
)

// EncoderFunc is used to create an Encoder from io.Writer
type EncoderFunc func(context.Context, io.Writer) Encoder

// Encoder helps in transforming objects to message bytes
type Encoder interface {
	// Encode takes any object and encodes it into bytes
	Encode(v interface{}) error
}

// DefaultEncoderFunc is a EncoderFunc that uses a json.Encoder as the Encoder.
func DefaultEncoderFunc(_ context.Context, w io.Writer) Encoder {
	return json.NewEncoder(w)
}

func (f EncoderFunc) apply(o *clientOptions) { o.newEncoder = f }
