package courier

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testData struct {
	Message string `json:"message"`
	Value   int    `json:"value"`
}

func gzipEncoderFunc(_ context.Context, w io.Writer) Encoder {
	return &gzipEncoder{w: gzip.NewWriter(w)}
}

type gzipEncoder struct {
	w *gzip.Writer
}

func (e *gzipEncoder) Encode(v interface{}) error {
	data, ok := v.([]byte)
	if !ok {
		return json.NewEncoder(e.w).Encode(v)
	}

	if _, err := e.w.Write(data); err != nil {
		return err
	}

	return e.w.Close()
}

func gzipDecoderFunc(_ context.Context, r io.Reader) Decoder {
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return &errorDecoder{err: err}
	}

	return &gzipDecoder{r: gzipReader}
}

type gzipDecoder struct {
	r *gzip.Reader
}

func (d *gzipDecoder) Decode(v interface{}) error {
	defer func() { _ = d.r.Close() }()

	data, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}

	if p, ok := v.(*[]byte); ok {
		*p = data

		return nil
	}

	if w, ok := v.(io.Writer); ok {
		_, err := w.Write(data)

		return err
	}

	return json.NewDecoder(bytes.NewReader(data)).Decode(v)
}

type errorDecoder struct {
	err error
}

func (d *errorDecoder) Decode(v interface{}) error {
	return d.err
}

func base64EncoderFunc(_ context.Context, w io.Writer) Encoder {
	return &base64Encoder{w: base64.NewEncoder(base64.StdEncoding, w)}
}

type base64Encoder struct {
	w io.WriteCloser
}

func (e *base64Encoder) Encode(v interface{}) error {
	data, ok := v.([]byte)
	if !ok {
		return json.NewEncoder(e.w).Encode(v)
	}

	if _, err := e.w.Write(data); err != nil {
		return err
	}

	return e.w.Close()
}

func base64DecoderFunc(_ context.Context, r io.Reader) Decoder {
	return &base64DecoderWrapper{r: base64.NewDecoder(base64.StdEncoding, r)}
}

type base64DecoderWrapper struct {
	r io.Reader
}

func (d *base64DecoderWrapper) Decode(v interface{}) error {
	data, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}

	if p, ok := v.(*[]byte); ok {
		*p = data
		return nil
	}

	if w, ok := v.(io.Writer); ok {
		_, err := w.Write(data)
		return err
	}

	return json.NewDecoder(bytes.NewReader(data)).Decode(v)
}

func TestChainEncoders(t *testing.T) {
	tests := []struct {
		name     string
		encoders []EncoderFunc
		input    interface{}
		validate func(t *testing.T, output []byte)
	}{
		{
			name:     "TestEncoder",
			encoders: []EncoderFunc{DefaultEncoderFunc},
			input:    testData{Message: "hello", Value: 1},
			validate: func(t *testing.T, output []byte) {
				var result testData
				err := json.Unmarshal(output, &result)
				require.NoError(t, err)
				assert.Equal(t, "hello", result.Message)
				assert.Equal(t, 1, result.Value)
			},
		},
		{
			name:     "NoEncoders",
			encoders: []EncoderFunc{},
			input:    testData{Message: "test", Value: 1},
			validate: func(t *testing.T, output []byte) {
				var result testData
				err := json.Unmarshal(output, &result)
				require.NoError(t, err)
				assert.Equal(t, "test", result.Message)
			},
		},
		{
			name:     "JSONGzip",
			encoders: []EncoderFunc{DefaultEncoderFunc, gzipEncoderFunc},
			input:    testData{Message: "testJSONGzip", Value: 100},
			validate: func(t *testing.T, output []byte) {
				gzipReader, err := gzip.NewReader(bytes.NewReader(output))
				require.NoError(t, err)
				defer func() { _ = gzipReader.Close() }()

				var result testData
				err = json.NewDecoder(gzipReader).Decode(&result)
				require.NoError(t, err)
				assert.Equal(t, "testJSONGzip", result.Message)
				assert.Equal(t, 100, result.Value)
			},
		},
		{
			name:     "JSONBase64",
			encoders: []EncoderFunc{DefaultEncoderFunc, base64EncoderFunc},
			input:    testData{Message: "testJSONBase64", Value: 200},
			validate: func(t *testing.T, output []byte) {
				decoded, err := base64.StdEncoding.DecodeString(string(output))
				require.NoError(t, err)

				var result testData
				err = json.Unmarshal(decoded, &result)
				require.NoError(t, err)
				assert.Equal(t, "testJSONBase64", result.Message)
				assert.Equal(t, 200, result.Value)
			},
		},
		{
			name: "JSONGzipBase64",
			encoders: []EncoderFunc{
				DefaultEncoderFunc,
				gzipEncoderFunc,
				base64EncoderFunc,
			},
			input: testData{Message: "testJSONGzipBase64", Value: 999},
			validate: func(t *testing.T, output []byte) {
				decoded, err := base64.StdEncoding.DecodeString(string(output))
				require.NoError(t, err)

				gzipReader, err := gzip.NewReader(bytes.NewReader(decoded))
				require.NoError(t, err)
				defer func() { _ = gzipReader.Close() }()

				var result testData
				err = json.NewDecoder(gzipReader).Decode(&result)
				require.NoError(t, err)
				assert.Equal(t, "testJSONGzipBase64", result.Message)
				assert.Equal(t, 999, result.Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.Buffer{}

			chainedEncoder := ChainEncoders(tt.encoders...)
			encoder := chainedEncoder(context.Background(), &buf)

			err := encoder.Encode(tt.input)
			require.NoError(t, err)

			tt.validate(t, buf.Bytes())
		})
	}
}

func TestChainDecoders(t *testing.T) {
	tests := []struct {
		name     string
		decoders []DecoderFunc
		prepare  func(t *testing.T) []byte
		expected testData
	}{
		{
			name:     "TestDecoder",
			decoders: []DecoderFunc{DefaultDecoderFunc},
			prepare: func(t *testing.T) []byte {
				data, err := json.Marshal(testData{Message: "test", Value: 42})
				require.NoError(t, err)
				return data
			},
			expected: testData{Message: "test", Value: 42},
		},
		{
			name:     "NoDecoders",
			decoders: []DecoderFunc{},
			prepare: func(t *testing.T) []byte {
				data, err := json.Marshal(testData{Message: "test", Value: 1})
				require.NoError(t, err)
				return data
			},
			expected: testData{Message: "test", Value: 1},
		},
		{
			name:     "JSONGzip",
			decoders: []DecoderFunc{gzipDecoderFunc, DefaultDecoderFunc},
			prepare: func(t *testing.T) []byte {
				data, err := json.Marshal(testData{Message: "testJSONGzip", Value: 100})
				require.NoError(t, err)

				var buf bytes.Buffer
				gzipWriter := gzip.NewWriter(&buf)
				_, err = gzipWriter.Write(data)
				require.NoError(t, err)
				require.NoError(t, gzipWriter.Close())

				return buf.Bytes()
			},
			expected: testData{Message: "testJSONGzip", Value: 100},
		},
		{
			name:     "JSONBase64",
			decoders: []DecoderFunc{base64DecoderFunc, DefaultDecoderFunc},
			prepare: func(t *testing.T) []byte {
				data, err := json.Marshal(testData{Message: "testJSONBase64", Value: 200})
				require.NoError(t, err)

				encoded := base64.StdEncoding.EncodeToString(data)
				return []byte(encoded)
			},
			expected: testData{Message: "testJSONBase64", Value: 200},
		},
		{
			name: "JSONGzipBase64",
			decoders: []DecoderFunc{
				base64DecoderFunc,
				gzipDecoderFunc,
				DefaultDecoderFunc,
			},
			prepare: func(t *testing.T) []byte {
				data, err := json.Marshal(testData{Message: "testJSONGzipBase64", Value: 999})
				require.NoError(t, err)

				var gzipBuf bytes.Buffer
				gzipWriter := gzip.NewWriter(&gzipBuf)
				_, err = gzipWriter.Write(data)
				require.NoError(t, err)
				require.NoError(t, gzipWriter.Close())

				encoded := base64.StdEncoding.EncodeToString(gzipBuf.Bytes())
				return []byte(encoded)
			},
			expected: testData{Message: "testJSONGzipBase64", Value: 999},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.prepare(t)

			chainedDecoder := ChainDecoders(tt.decoders...)
			decoder := chainedDecoder(context.Background(), bytes.NewReader(input))

			var result testData
			err := decoder.Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.Message, result.Message)
			assert.Equal(t, tt.expected.Value, result.Value)
		})
	}
}

func TestChainEncodersDecoders(t *testing.T) {
	tests := []struct {
		name     string
		encoders []EncoderFunc
		decoders []DecoderFunc
		input    testData
	}{
		{
			name:     "JSON",
			encoders: []EncoderFunc{DefaultEncoderFunc},
			decoders: []DecoderFunc{DefaultDecoderFunc},
			input:    testData{Message: "testJSON", Value: 1},
		},
		{
			name:     "JSONGzip",
			encoders: []EncoderFunc{DefaultEncoderFunc, gzipEncoderFunc},
			decoders: []DecoderFunc{gzipDecoderFunc, DefaultDecoderFunc},
			input:    testData{Message: "testJSONGzip", Value: 555},
		},
		{
			name:     "JSONBase64",
			encoders: []EncoderFunc{DefaultEncoderFunc, base64EncoderFunc},
			decoders: []DecoderFunc{base64DecoderFunc, DefaultDecoderFunc},
			input:    testData{Message: "testJSONBase64", Value: 777},
		},
		{
			name: "JSONGzipBase64",
			encoders: []EncoderFunc{
				DefaultEncoderFunc,
				gzipEncoderFunc,
				base64EncoderFunc,
			},
			decoders: []DecoderFunc{
				base64DecoderFunc,
				gzipDecoderFunc,
				DefaultDecoderFunc,
			},
			input: testData{Message: "testJSONGzipBase64", Value: 12345},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.Buffer{}

			chainedEncoder := ChainEncoders(tt.encoders...)
			encoder := chainedEncoder(context.Background(), &buf)
			err := encoder.Encode(tt.input)
			require.NoError(t, err)

			chainedDecoder := ChainDecoders(tt.decoders...)
			decoder := chainedDecoder(context.Background(), bytes.NewReader(buf.Bytes()))

			var result testData
			err = decoder.Decode(&result)

			require.NoError(t, err)

			assert.Equal(t, tt.input.Message, result.Message)
			assert.Equal(t, tt.input.Value, result.Value)
		})
	}
}

func TestChainErrors(t *testing.T) {
	encoders := []EncoderFunc{
		DefaultEncoderFunc,
		gzipEncoderFunc,
		base64EncoderFunc,
	}

	decoders := []DecoderFunc{
		DefaultDecoderFunc,
		gzipDecoderFunc,
		base64DecoderFunc,
	}

	buf := bytes.Buffer{}

	chainedEncoder := ChainEncoders(encoders...)
	encoder := chainedEncoder(context.Background(), &buf)
	err := encoder.Encode(testData{Message: "testChainErrors", Value: 42})
	require.NoError(t, err)

	chainedDecoder := ChainDecoders(decoders...)
	decoder := chainedDecoder(context.Background(), bytes.NewReader(buf.Bytes()))

	var result testData
	err = decoder.Decode(&result)
	assert.Error(t, err)
}
