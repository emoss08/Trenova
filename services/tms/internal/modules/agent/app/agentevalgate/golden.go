package agentevalgate

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
)

var prettyJSON = sonic.Config{SortMapKeys: true}.Froze()

func MarshalJSON(value any) ([]byte, error) {
	encoded, err := prettyJSON.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(encoded, '\n'), nil
}

func WriteFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}

	return os.WriteFile(path, content, 0o600)
}

func Golden(t testing.TB, path string, got []byte, update bool, refresh string) {
	t.Helper()

	if update {
		if err := WriteFile(path, got); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("%s does not exist; create it with: %s", path, refresh)
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if bytes.Equal(want, got) {
		return
	}

	t.Errorf("%s is stale; refresh it with: %s\n%s", path, refresh, FirstDifference(want, got))
}

func FirstDifference(want, got []byte) string {
	wantLines := strings.Split(string(want), "\n")
	gotLines := strings.Split(string(got), "\n")
	for idx := range max(len(wantLines), len(gotLines)) {
		var wantLine, gotLine string
		if idx < len(wantLines) {
			wantLine = wantLines[idx]
		}
		if idx < len(gotLines) {
			gotLine = gotLines[idx]
		}
		if wantLine != gotLine {
			return fmt.Sprintf("first difference at line %d:\n  committed: %q\n  now:       %q",
				idx+1, wantLine, gotLine)
		}
	}

	return "the files differ only in line endings"
}
