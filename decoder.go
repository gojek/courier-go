package courier

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
)

// DecoderFunc is used to create a Decoder from io.Reader stream
// of message bytes before calling MessageHandler
type DecoderFunc func(context.Context, io.Reader) Decoder

// Decoder helps to decode message bytes into the desired object
type Decoder interface {
	// Decode decodes message bytes into the passed object
	Decode(v interface{}) error
}

// DefaultDecoderFunc is a DecoderFunc that uses a json.Decoder as the Decoder.
func DefaultDecoderFunc(_ context.Context, r io.Reader) Decoder {
	return json.NewDecoder(r)
}

func base64JsonDecoder(_ context.Context, r io.Reader) Decoder {
	return json.NewDecoder(base64.NewDecoder(base64.StdEncoding, r))
}

func (f DecoderFunc) apply(o *clientOptions) { o.newDecoder = f }
