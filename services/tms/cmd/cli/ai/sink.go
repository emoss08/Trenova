package ai

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	datasetDirMode  = 0o700
	datasetFileMode = 0o600
)

var errUnsafeDatasetName = errors.New("dataset file names must be plain names")

type directorySink struct {
	dir     string
	staging string
}

func newDirectorySink(dir string) (*directorySink, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, errors.New("--out is required")
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", dir, err)
	}
	if _, statErr := os.Stat(absolute); statErr == nil {
		return nil, fmt.Errorf("%s already exists; choose a new directory", absolute)
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return nil, fmt.Errorf("check %s: %w", absolute, statErr)
	}

	parent := filepath.Dir(absolute)
	if err = os.MkdirAll(parent, datasetDirMode); err != nil {
		return nil, fmt.Errorf("create %s: %w", parent, err)
	}
	staging, err := os.MkdirTemp(parent, "."+filepath.Base(absolute)+".partial-")
	if err != nil {
		return nil, fmt.Errorf("create a staging directory in %s: %w", parent, err)
	}

	return &directorySink{dir: absolute, staging: staging}, nil
}

func (s *directorySink) Create(name string) (io.WriteCloser, error) {
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return nil, fmt.Errorf("%w: %q", errUnsafeDatasetName, name)
	}

	file, err := os.OpenFile(
		filepath.Join(s.staging, name),
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		datasetFileMode,
	)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", name, err)
	}

	return file, nil
}

func (s *directorySink) commit() error {
	if err := os.Rename(s.staging, s.dir); err != nil {
		return fmt.Errorf("move the datasets into %s: %w", s.dir, err)
	}

	return nil
}

func (s *directorySink) abort() {
	_ = os.RemoveAll(s.staging)
}
