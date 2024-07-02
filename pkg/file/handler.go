package file

import (
	"os"
)

// FileHandler defines methods for file operations.
type FileHandler interface {
	Open(name string) (*os.File, error)
	Create(name string) (*os.File, error)
	Stat(name string) (os.FileInfo, error)
}

// OSFileHandler implements FileHandler using the os package.
type OSFileHandler struct{}

func (OSFileHandler) Open(name string) (*os.File, error) {
	return os.Open(name)
}

func (OSFileHandler) Create(name string) (*os.File, error) {
	return os.Create(name)
}

func (OSFileHandler) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
