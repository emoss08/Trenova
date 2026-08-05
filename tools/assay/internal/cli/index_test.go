package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPatternMatching pins --packages semantics. The documented example
// `--packages ./internal/...` did not match anything at all: relative patterns
// were prefix-matched against import paths, and the prefix had no segment
// boundary, so `.../cover` also swallowed `.../coverage`.
func TestPatternMatching(t *testing.T) {
	cwd := filepath.Join("/repo", "tools", "assay")
	pkg := "github.com/emoss08/assay/internal/cover"
	dir := filepath.Join(cwd, "internal", "cover")

	cases := []struct {
		name    string
		pattern string
		want    bool
	}{
		{"relative wildcard from the caller's directory", "./internal/...", true},
		{"relative exact directory", "./internal/cover", true},
		{"relative to a different directory", "./cmd/...", false},
		{"bare dot-dot-dot matches everything", "...", true},
		{"import path exact", "github.com/emoss08/assay/internal/cover", true},
		{"import path wildcard", "github.com/emoss08/assay/internal/...", true},
		{"import path wildcard includes its own base", "github.com/emoss08/assay/internal/cover/...", true},
		{"no segment boundary bleed", "github.com/emoss08/assay/internal/cov/...", false},
		{"prefix without wildcard is not a match", "github.com/emoss08/assay/internal", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, patternMatches(tc.pattern, cwd, pkg, dir))
		})
	}
}

func TestPatternDoesNotBleedAcrossDirectorySegments(t *testing.T) {
	cwd := "/repo"
	coverage := filepath.Join("/repo", "internal", "coverage")

	assert.False(t, patternMatches("./internal/cover/...", cwd, "example.com/x/internal/coverage", coverage),
		"a directory wildcard must stop at path segments, not raw byte prefixes")
}
