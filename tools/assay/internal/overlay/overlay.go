package overlay

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeebo/blake3"
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

// UniqueName renders an identifier as a filesystem-safe name that is injective:
// two distinct identifiers can never map to the same file. Flattening path
// separators alone made example.com/a/b and example.com/a_b collide, which let
// one package's tests silently execute another package's binary; the hash
// suffix carries the uniqueness, the readable prefix is for humans.
func UniqueName(identifier string) string {
	sum := blake3.Sum256([]byte(identifier))

	var b strings.Builder
	b.Grow(len(identifier) + 14)
	for i := 0; i < len(identifier); i++ {
		c := identifier[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '-', c == '_':
			b.WriteByte(c)
		default:
			b.WriteByte('_')
		}
	}
	b.WriteByte('-')
	b.WriteString(hex.EncodeToString(sum[:6]))

	return b.String()
}
