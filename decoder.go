package courier

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// DecoderFunc is used to create a Decoder from io.Reader stream
// of message bytes before calling MessageHandler;
// the context.Context value may be used to select appropriate Decoder.
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

// ChainDecoderFunc creates a DecoderFunc that tries multiple decoders in sequence.
// It attempts each decoder in order; if successful, it stops. If all fail, it returns
// a combined error containing all individual errors.
func ChainDecoderFunc(decoders ...DecoderFunc) DecoderFunc {
	return func(ctx context.Context, r io.Reader) Decoder {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil {
			return &chainDecoder{decoders: nil}
		}

		decs := make([]Decoder, 0, len(decoders))
		for _, fn := range decoders {
			decs = append(decs, fn(ctx, bytes.NewReader(buf.Bytes())))
		}

		return &chainDecoder{decoders: decs}
	}
}

type chainDecoder struct {
	decoders []Decoder
}

func (f *chainDecoder) Decode(v interface{}) error {
	var errs []error

	for _, dec := range f.decoders {
		if err := dec.Decode(v); err != nil {
			errs = append(errs, err)

			continue
		}

		return nil
	}

	return fmt.Errorf("all decoders failed: %w", errors.Join(errs...))
}

func (f DecoderFunc) apply(o *clientOptions) { o.newDecoder = f }
