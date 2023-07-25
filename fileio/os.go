package fileio

import "os"

// OSFile provides a storage based on Go's standard "os" package for filesystem support.
type OSFile struct{}

func (*OSFile) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (*OSFile) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (*OSFile) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (*OSFile) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}
