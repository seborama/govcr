package govcr

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
)

// compress data and return the result
func compress(data []byte) ([]byte, error) {
	var out bytes.Buffer

	w := gzip.NewWriter(&out)
	if _, err := io.Copy(w, bytes.NewBuffer(data)); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// decompress data and return the result
func decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	data, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return data, nil
}
