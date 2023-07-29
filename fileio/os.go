package fileio

import (
	"os"

	"github.com/pkg/errors"
)

// OSFile provides a storage based on Go's standard "os" package for filesystem support.
type OSFile struct{}

func (*OSFile) MkdirAll(path string, perm os.FileMode) error {
	return errors.WithStack(os.MkdirAll(path, perm))
}

func (*OSFile) ReadFile(name string) ([]byte, error) {
	data, err := os.ReadFile(name)
	return data, errors.WithStack(err)
}

func (*OSFile) WriteFile(name string, data []byte, perm os.FileMode) error {
	return errors.WithStack(os.WriteFile(name, data, perm))
}

func (*OSFile) NotExist(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return true, nil
	}
	return false, errors.WithStack(err)
}
