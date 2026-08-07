package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/report"
	"github.com/emoss08/assay/internal/testfixture"
)

func TestReadFileListResolvesRelativePaths(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "changed.txt")
	absolute := filepath.Join(root, "already", "absolute.go")

	content := "svc/svc.go\n\n  repo/repo.go  \n" + absolute + "\n"
	require.NoError(t, os.WriteFile(list, []byte(content), 0o644))

	got, err := readFileList(list, root, nil)
	require.NoError(t, err)

	paths := make([]string, 0, len(got))
	for _, c := range got {
		paths = append(paths, c.Path)
	}
	assert.Equal(t, []string{
		filepath.Join(root, "svc", "svc.go"),
		filepath.Join(root, "repo", "repo.go"),
		absolute,
	}, paths)
}

func TestReadFileListReadsStdin(t *testing.T) {
	root := t.TempDir()

	got, err := readFileList("-", root, strings.NewReader("a/a.go\nb/b.go\n"))
	require.NoError(t, err)

	require.Len(t, got, 2)
	assert.Equal(t, filepath.Join(root, "a", "a.go"), got[0].Path)
}

func TestReadFileListRejectsMissingFile(t *testing.T) {
	_, err := readFileList(filepath.Join(t.TempDir(), "nope.txt"), t.TempDir(), nil)

	require.Error(t, err)
}

func parseGoTestArgs(t *testing.T, argv []string) ([]string, error) {
	t.Helper()

	var got []string
	var gotErr error
	cmd := &cobra.Command{
		Use:  "x",
		Args: cobra.ArbitraryArgs,
		RunE: func(c *cobra.Command, args []string) error {
			got, gotErr = goTestArgs(c, args)

			return nil
		},
	}
	cmd.Flags().String("since", "", "")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(argv)
	require.NoError(t, cmd.Execute())

	return got, gotErr
}

func TestGoTestArgsForwardsEverythingAfterDash(t *testing.T) {
	got, err := parseGoTestArgs(t, []string{"--since", "main", "--", "-count=1", "-race"})

	require.NoError(t, err)
	assert.Equal(t, []string{"-count=1", "-race"}, got)
}

func TestGoTestArgsAllowsNone(t *testing.T) {
	got, err := parseGoTestArgs(t, []string{"--since", "main"})

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestGoTestArgsRejectsPositionalArguments(t *testing.T) {
	_, err := parseGoTestArgs(t, []string{"./..."})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "use -- to pass flags to go test")
}

func TestGoTestArgsRejectsArgumentsBeforeDash(t *testing.T) {
	_, err := parseGoTestArgs(t, []string{"./...", "--", "-race"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "before --")
}

func execute(t *testing.T, argv ...string) (string, string, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(argv)
	err := cmd.ExecuteContext(t.Context())

	return stdout.String(), stderr.String(), err
}

func TestVersionCommand(t *testing.T) {
	stdout, _, err := execute(t, "version")

	require.NoError(t, err)
	assert.Contains(t, stdout, "assay "+Version)
}

func TestUnknownCommandFails(t *testing.T) {
	_, _, err := execute(t, "frobnicate")

	require.Error(t, err)
}

func writeChangeList(t *testing.T, rels ...string) string {
	t.Helper()

	list := filepath.Join(t.TempDir(), "changed.txt")
	require.NoError(t, os.WriteFile(list, []byte(strings.Join(rels, "\n")+"\n"), 0o644))

	return list
}

func TestSelectCommandEmitsJSON(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := testfixture.Write(t)

	stdout, _, err := execute(t,
		"select", "--root", root, "--files", writeChangeList(t, "repo/repo.go"), "--json")
	require.NoError(t, err)

	var summary report.Summary
	require.NoError(t, json.Unmarshal([]byte(stdout), &summary))

	selected := make([]string, 0, len(summary.SelectedPackages))
	for _, pkg := range summary.SelectedPackages {
		selected = append(selected, pkg.ImportPath)
	}
	assert.Equal(t, []string{
		testfixture.Module + "/app",
		testfixture.Module + "/repo",
		testfixture.Module + "/svc",
		testfixture.Module + "/tagged",
	}, selected)
	assert.False(t, summary.SelectAll)
}

func TestSelectCommandVerboseText(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := testfixture.Write(t)

	stdout, _, err := execute(t,
		"select", "--root", root, "--files", writeChangeList(t, "repo/repo.go"), "-v", "--no-color")
	require.NoError(t, err)

	assert.Contains(t, stdout, "with tests")
	assert.Contains(t, stdout, "full")
	assert.Contains(t, stdout, testfixture.Module+"/repo")
	assert.NotContains(t, stdout, "\x1b[", "--no-color must suppress ANSI escapes")
}

func TestSelectCommandAllReportsFallback(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := testfixture.Write(t)

	stdout, _, err := execute(t, "select", "--root", root, "--all", "--json")
	require.NoError(t, err)

	var summary report.Summary
	require.NoError(t, json.Unmarshal([]byte(stdout), &summary))
	assert.True(t, summary.SelectAll)
	assert.Equal(t, "--all requested", summary.SelectAllReason)
}

func TestSelectCommandHonoursBuildTags(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := testfixture.Write(t)

	stdout, _, err := execute(t,
		"select", "--root", root, "--tags", "integration",
		"--files", writeChangeList(t, "repo/repo.go"), "--json")
	require.NoError(t, err)

	var summary report.Summary
	require.NoError(t, json.Unmarshal([]byte(stdout), &summary))

	var found bool
	for _, pkg := range summary.SelectedPackages {
		if pkg.ImportPath == testfixture.Module+"/tagged" {
			found = true
		}
	}
	assert.True(t, found, "the build-tagged package must be selected when its tag is enabled")
}

func TestSelectCommandCachesTheGraph(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := testfixture.Write(t)
	cacheDir := t.TempDir()
	list := writeChangeList(t, "repo/repo.go")

	summaryFor := func() report.Summary {
		t.Helper()
		stdout, _, err := execute(t,
			"select", "--root", root, "--files", list, "--cache-dir", cacheDir, "--json")
		require.NoError(t, err)

		var summary report.Summary
		require.NoError(t, json.Unmarshal([]byte(stdout), &summary))

		return summary
	}

	cold := summaryFor()
	assert.False(t, cold.GraphFromCache, "the first run has nothing to reuse")

	warm := summaryFor()
	assert.True(t, warm.GraphFromCache, "the second run must reuse the cached graph")
	assert.Equal(t, cold.SelectedPackages, warm.SelectedPackages,
		"a cached graph must select exactly what a fresh one selects")
}

func TestSelectCommandInvalidatesCacheWhenSourceChanges(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := testfixture.Write(t)
	cacheDir := t.TempDir()
	list := writeChangeList(t, "repo/repo.go")

	run := func() report.Summary {
		t.Helper()
		stdout, _, err := execute(t,
			"select", "--root", root, "--files", list, "--cache-dir", cacheDir, "--json")
		require.NoError(t, err)

		var summary report.Summary
		require.NoError(t, json.Unmarshal([]byte(stdout), &summary))

		return summary
	}

	run()
	require.True(t, run().GraphFromCache)

	require.NoError(t, os.WriteFile(
		filepath.Join(root, "repo", "repo.go"),
		[]byte("package repo\n\nfunc Get() int { return 2 }\n\nfunc Extra() {}\n"), 0o644))

	assert.False(t, run().GraphFromCache, "editing a Go file must invalidate the cached graph")
}

func TestSelectCommandNoCacheAlwaysReloads(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := testfixture.Write(t)
	cacheDir := t.TempDir()
	list := writeChangeList(t, "repo/repo.go")

	for range 2 {
		stdout, _, err := execute(t,
			"select", "--root", root, "--files", list, "--cache-dir", cacheDir, "--no-cache", "--json")
		require.NoError(t, err)

		var summary report.Summary
		require.NoError(t, json.Unmarshal([]byte(stdout), &summary))
		assert.False(t, summary.GraphFromCache)
	}

	entries, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "--no-cache must not write cache entries either")
}

func TestConflictingChangeSourcesAreRejected(t *testing.T) {
	cases := [][]string{
		{"--since", "main", "--files", "-"},
		{"--since", "main", "--all"},
		{"--files", "-", "--all"},
		{"--since", "main", "--files", "-", "--all"},
	}

	for _, flags := range cases {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			_, _, err := execute(t, append([]string{"select", "--root", t.TempDir()}, flags...)...)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "mutually exclusive")
		})
	}
}

func TestSingleChangeSourceIsAccepted(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := testfixture.Write(t)

	for _, flags := range [][]string{
		{"--files", writeChangeList(t, "repo/repo.go")},
		{"--all"},
	} {
		t.Run(flags[0], func(t *testing.T) {
			_, _, err := execute(t, append([]string{"select", "--root", root, "--json"}, flags...)...)

			require.NoError(t, err)
		})
	}
}

func TestRunCommandRejectsPositionalArguments(t *testing.T) {
	_, _, err := execute(t, "run", "./...")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "use -- to pass flags to go test")
}

func TestExitErrorCarriesCode(t *testing.T) {
	err := &ExitError{Code: 2}

	assert.Equal(t, "exit status 2", err.Error())
}
