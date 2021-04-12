package courier

import (
	"encoding/json"
	"io"
)

// EncoderFunc is used to create an Encoder from io.Writer
type EncoderFunc func(io.Writer) Encoder

// Encoder helps in transforming objects to message bytes
type Encoder interface {
	// Encode takes any object and encodes it into bytes
	Encode(v interface{}) error
}

func defaultEncoderFunc(w io.Writer) Encoder {
	return &jsonEncoder{w: w}
}

type jsonEncoder struct {
	w io.Writer
}

func (j jsonEncoder) Encode(v interface{}) error {
	return json.NewEncoder(j.w).Encode(v)
}
