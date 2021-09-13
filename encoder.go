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
	return json.NewEncoder(w)
}
