package vcs_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/vcs"
)

func churnRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	// Commits are dated ten days in the past so window assertions are about
	// arithmetic, not about racing the wall clock.
	when := time.Now().Add(-10 * 24 * time.Hour).Format(time.RFC3339)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
			"GIT_AUTHOR_DATE="+when, "GIT_COMMITTER_DATE="+when,
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}
	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}

	git("init", "-q")

	write("pkg/hot.go", "package pkg\n\nfunc Hot() int { return 1 }\n")
	write("pkg/cold.go", "package pkg\n\nfunc Cold() int { return 1 }\n")
	write("pkg/hot_test.go", "package pkg\n")
	write("README.md", "readme\n")
	git("add", "-A")
	git("commit", "-qm", "initial")

	write("pkg/hot.go", "package pkg\n\nfunc Hot() int { return 2 }\n")
	git("add", "-A")
	git("commit", "-qm", "touch hot")

	write("pkg/hot.go", "package pkg\n\nfunc Hot() int { return 3 }\n")
	write("pkg/hot_test.go", "package pkg\n\n// changed\n")
	write("README.md", "readme changed\n")
	git("add", "-A")
	git("commit", "-qm", "touch hot again, plus noise")

	return root
}

// TestChurnCountsProductionGoFilesOnly pins the filter: churn in tests is not
// production exposure, and churn in docs is not this tool's business.
func TestChurnCountsProductionGoFilesOnly(t *testing.T) {
	root := churnRepo(t)

	churn, err := vcs.Churn(t.Context(), root, 30*24*time.Hour)
	require.NoError(t, err)

	hot := churn["pkg/hot.go"]
	assert.Equal(t, 3, hot.Commits, "every commit touched hot.go")
	assert.Positive(t, hot.LinesChanged)
	assert.Positive(t, hot.LastUnix)

	assert.Equal(t, 1, churn["pkg/cold.go"].Commits, "cold.go only existed in the initial commit")
	assert.NotContains(t, churn, "pkg/hot_test.go")
	assert.NotContains(t, churn, "README.md")
}

// TestChurnHonoursTheWindow pins the time bound: commits older than the window
// contribute nothing, and an empty history is a result, not an error.
func TestChurnHonoursTheWindow(t *testing.T) {
	root := churnRepo(t)

	churn, err := vcs.Churn(t.Context(), root, 24*time.Hour)
	require.NoError(t, err)

	assert.Empty(t, churn, "every commit is dated ten days back, outside a one-day window")
}
