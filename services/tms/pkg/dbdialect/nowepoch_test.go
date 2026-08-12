package dbdialect_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNowEpoch(t *testing.T) {
	assert.Equal(t, "extract(epoch from current_timestamp)::bigint", dbdialect.Postgres.NowEpoch())
	assert.Equal(t, "unixepoch()", dbdialect.SQLite.NowEpoch())
}

// TestNoHardcodedEpochExpression guards the class of bug that broke login on
// SQLite: a Postgres-only epoch expression written straight into query SQL.
// Model `default:` tags are exempt because bun only emits them for zero values,
// and BeforeAppendModel sets those fields before every insert.
const postgresOnlyMarker = "dialect:postgres-only"

func TestNoHardcodedEpochExpression(t *testing.T) {
	root := moduleRoot(t)

	var offenders []string

	// The helper itself is where the expression is allowed to live.
	helper := filepath.Join(root, "pkg", "dbdialect", "dbdialect.go")

	for _, dir := range []string{"internal", "pkg"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if path == helper {
				return nil
			}

			contents, readErr := os.ReadFile(path) //nolint:gosec // walking our own source tree
			if readErr != nil {
				return readErr
			}

			// Files that are deliberately Postgres-only opt out with a marker.
			// They must gate themselves on a capability instead.
			if strings.Contains(string(contents), postgresOnlyMarker) {
				return nil
			}

			for i, line := range strings.Split(string(contents), "\n") {
				lower := strings.ToLower(line)
				if !strings.Contains(lower, "extract(epoch") {
					continue
				}
				if strings.Contains(lower, "default:extract(epoch") {
					continue
				}

				rel, _ := filepath.Rel(root, path)
				offenders = append(offenders, rel+":"+itoa(i+1))
			}

			return nil
		})
		require.NoError(t, err)
	}

	assert.Emptyf(
		t,
		offenders,
		"use Kind.NowEpoch() or dbdialect.NowEpochFromBun() instead of a hardcoded epoch expression:\n%s",
		strings.Join(offenders, "\n"),
	)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	digits := make([]byte, 0, 8)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	return string(digits)
}

func moduleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for range 10 {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}

	t.Fatal("could not locate module root")

	return ""
}
