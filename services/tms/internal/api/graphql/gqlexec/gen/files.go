package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

const newFileMode fs.FileMode = 0o600

func writeFile(path string, data []byte) error {
	mode := newFileMode
	info, err := os.Stat(path)
	switch {
	case err == nil:
		mode = info.Mode().Perm()
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if err = os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
