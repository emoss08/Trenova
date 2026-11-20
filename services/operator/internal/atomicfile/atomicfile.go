package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

const Mode = 0o600

func Write(path string, body []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	name := tmp.Name()
	cleanup := func() { _ = os.Remove(name) }
	if err = tmp.Chmod(Mode); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	if _, err = tmp.Write(body); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	if err = tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	if err = os.Rename(name, path); err != nil {
		cleanup()
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	if d, derr := os.Open(dir); derr == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}
