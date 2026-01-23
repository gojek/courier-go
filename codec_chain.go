package courier

import (
	"bytes"
	"context"
	"io"
)

type chainedEncoders struct {
	context  context.Context
	encoders []EncoderFunc
	writer   io.Writer
}

func (c *chainedEncoders) Encode(v interface{}) error {
	if len(c.encoders) == 0 {
		return nil
	}

	var currentBuff bytes.Buffer
	firstEncoder := c.encoders[0](c.context, &currentBuff)

	if err := firstEncoder.Encode(v); err != nil {
		return err
	}

	for _, encFunc := range c.encoders[1:] {
		var nextBuff bytes.Buffer
		enc := encFunc(c.context, &nextBuff)

		if err := enc.Encode(currentBuff.Bytes()); err != nil {
			return err
		}

		currentBuff = nextBuff
	}

	_, err := c.writer.Write(currentBuff.Bytes())

	return err
}

func ChainEncoders(encoders ...EncoderFunc) EncoderFunc {
	if len(encoders) == 0 {
		return DefaultEncoderFunc
	}

	if len(encoders) == 1 {
		return encoders[0]
	}

	return func(ctx context.Context, w io.Writer) Encoder {
		return &chainedEncoders{
			context:  ctx,
			encoders: encoders,
			writer:   w,
		}
	}
}

func ChainDecoders(decoders ...DecoderFunc) DecoderFunc {
	if len(decoders) == 0 {
		return DefaultDecoderFunc
	}

	if len(decoders) == 1 {
		return decoders[0]
	}

	return func(ctx context.Context, r io.Reader) Decoder {
		return &chainedDecoders{
			context:  ctx,
			decoders: decoders,
			reader:   r,
		}
	}
}

type chainedDecoders struct {
	context  context.Context
	decoders []DecoderFunc
	reader   io.Reader
}

func (c *chainedDecoders) Decode(v interface{}) error {
	if len(c.decoders) == 0 {
		return nil
	}

	input, err := io.ReadAll(c.reader)
	if err != nil {
		return err
	}

	currentBuff := bytes.NewBuffer(input)

	for _, decFunc := range c.decoders[:len(c.decoders)-1] {
		var nextBuff bytes.Buffer

		dec := decFunc(c.context, currentBuff)

		if err := dec.Decode(&nextBuff); err != nil {
			return err
		}

		currentBuff = &nextBuff
	}

	lastDecoder := c.decoders[len(c.decoders)-1](c.context, currentBuff)

	return lastDecoder.Decode(v)
}
