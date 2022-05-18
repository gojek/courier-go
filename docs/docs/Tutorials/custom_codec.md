---
title: Custom Message Codec
description: Tutorial on using custom message encoder and decoders.
---

### Writing Custom Encoder Decoder

First write a custom encoder and decoder to work with the data types your project uses.

One common codec is `protobuf`.

```go title="decoder.go"
var bufPool = &sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// NewDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// A Decoder reads and decodes proto values from an input stream.
type Decoder struct {
	r io.Reader
}

// Decode reads the proto-encoded value from its
// input and stores it in the value pointed to by v.
func (d *Decoder) Decode(v interface{}) error {
	m, ok := v.(proto.Message)
	if !ok {
		return errors.New("value should be a proto.Message")
	}

	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)

	buf.Reset()

	if _, err := buf.ReadFrom(d.r); err != nil {
		return err
	}

	if err := proto.Unmarshal(buf.Bytes(), m); err != nil {
		return err
	}

	return nil
}
```

```go title="encoder.go"
// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// An Encoder writes proto values to an output stream.
type Encoder struct {
	w io.Writer
}

// Encode writes the proto encoding of v to the stream.
func (e *Encoder) Encode(v interface{}) error {
	m, ok := v.(proto.Message)
	if !ok {
		return errors.New("value should be a proto.Message")
	}

	b, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	if _, err := e.w.Write(b); err != nil {
		return err
	}

	return nil
}
```

### Registering Encoder Decoder

You can then use this codec to encode/decode protobuf messages with courier client.

```go title="client.go"
client, _ := courier.NewClient(
    courier.WithCustomEncoder(func(w io.Writer) courier.Encoder { return NewEncoder(w) }),
    courier.WithCustomDecoder(func(r io.Reader) courier.Decoder { return NewDecoder(r) }),
)
```
