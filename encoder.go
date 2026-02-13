package courier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// EncoderFunc is used to create an Encoder from io.Writer;
// the context.Context value may be used to select appropriate Encoder.
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

// ChainEncoderFunc creates an EncoderFunc that tries multiple encoders in sequence.
// It attempts each encoder in order; if successful, it stops. If all fail, it returns
// a combined error containing all individual errors.
func ChainEncoderFunc(encoders ...EncoderFunc) EncoderFunc {
	return func(ctx context.Context, w io.Writer) Encoder {
		encs := make([]Encoder, 0, len(encoders))
		for _, fn := range encoders {
			encs = append(encs, fn(ctx, w))
		}

		return &chainEncoder{encoders: encs}
	}
}

type chainEncoder struct {
	encoders []Encoder
}

func (f *chainEncoder) Encode(v interface{}) error {
	var errs []error

	for _, enc := range f.encoders {
		if err := enc.Encode(v); err != nil {
			errs = append(errs, err)

			continue
		}

		return nil
	}

	return fmt.Errorf("all encoders failed: %w", errors.Join(errs...))
}

func (f EncoderFunc) apply(o *clientOptions) { o.newEncoder = f }
