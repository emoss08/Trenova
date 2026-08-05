package overlay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteUnderBasename writes content to workdir/group/<basename(original)>,
// keeping the original file name because the go command derives build
// constraints — _linux, _amd64, _test — from the name it is given, not from the
// path being replaced.
func WriteUnderBasename(workdir, group, original string, content []byte) (string, error) {
	dir := filepath.Join(workdir, group)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create overlay directory: %w", err)
	}

	path := filepath.Join(dir, filepath.Base(original))
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}

	return path, nil
}

// WriteFile marshals {"Replace": replace} into workdir/overlay.json and returns
// its path. A key whose on-disk path does not exist injects a brand-new file
// into the package — the compiler sees a file the source tree never contains.
func WriteFile(workdir string, replace map[string]string) (string, error) {
	payload, err := json.Marshal(map[string]map[string]string{"Replace": replace})
	if err != nil {
		return "", fmt.Errorf("encode overlay: %w", err)
	}

	path := filepath.Join(workdir, "overlay.json")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return "", fmt.Errorf("write overlay: %w", err)
	}

	return path, nil
}
