package compression

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/pkg/errors"
)

// Compress data and return the result.
func Compress(data []byte) ([]byte, error) {
	var out bytes.Buffer

	w := gzip.NewWriter(&out)

	if _, err := io.Copy(w, bytes.NewBuffer(data)); err != nil {
		return nil, errors.WithStack(err)
	}

	if err := w.Close(); err != nil {
		return nil, errors.WithStack(err)
	}

	return out.Bytes(), nil
}

// Decompress data and return the result.
func Decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	data, err = io.ReadAll(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return data, nil
}
