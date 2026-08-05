package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNameStatusZ(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repo")

	cases := []struct {
		name    string
		payload string
		want    []Change
	}{
		{
			name:    "single modification",
			payload: "M\x00svc/svc.go\x00",
			want:    []Change{{Path: filepath.Join(root, "svc/svc.go"), Status: "M"}},
		},
		{
			name:    "addition and deletion",
			payload: "A\x00a.go\x00D\x00b.go\x00",
			want: []Change{
				{Path: filepath.Join(root, "a.go"), Status: "A"},
				{Path: filepath.Join(root, "b.go"), Status: "D"},
			},
		},
		{
			name:    "rename yields both paths",
			payload: "R100\x00old/x.go\x00new/x.go\x00",
			want: []Change{
				{Path: filepath.Join(root, "old/x.go"), Status: "R100"},
				{Path: filepath.Join(root, "new/x.go"), Status: "R100"},
			},
		},
		{
			name:    "copy yields both paths",
			payload: "C75\x00src.go\x00dst.go\x00",
			want: []Change{
				{Path: filepath.Join(root, "src.go"), Status: "C75"},
				{Path: filepath.Join(root, "dst.go"), Status: "C75"},
			},
		},
		{
			name:    "truncated rename is dropped",
			payload: "M\x00a.go\x00R100\x00only-old.go\x00",
			want:    []Change{{Path: filepath.Join(root, "a.go"), Status: "M"}},
		},
		{
			name:    "empty payload",
			payload: "",
			want:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseNameStatusZ(tc.payload, root))
		})
	}
}

func TestDedupeChangesSortsAndDropsDuplicates(t *testing.T) {
	got := dedupeChanges([]Change{
		{Path: "/b", Status: "M"},
		{Path: "/a", Status: "A"},
		{Path: "/b", Status: "D"},
	})

	assert.Equal(t, []Change{{Path: "/a", Status: "A"}, {Path: "/b", Status: "M"}}, got)
}

func initRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=assay", "GIT_AUTHOR_EMAIL=assay@example.com",
			"GIT_COMMITTER_NAME=assay", "GIT_COMMITTER_EMAIL=assay@example.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}

	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}

	run("init", "--initial-branch=main")
	write("base.go", "package base\n")
	run("add", ".")
	run("commit", "-m", "base")

	run("checkout", "-b", "feature")
	write("feature.go", "package feature\n")
	run("add", ".")
	run("commit", "-m", "feature")

	write("dirty.go", "package dirty\n")
	write("untracked.txt", "hello\n")
	run("add", "dirty.go")

	return root
}

func relPaths(t *testing.T, root string, changes []Change) []string {
	t.Helper()

	out := make([]string, 0, len(changes))
	for _, c := range changes {
		rel, err := filepath.Rel(root, c.Path)
		require.NoError(t, err)
		out = append(out, filepath.ToSlash(rel))
	}
	slices.Sort(out)

	return out
}

func TestChangesAgainstMergeBase(t *testing.T) {
	root := initRepo(t)

	got, err := Changes(t.Context(), Options{Root: root, Base: "main", IncludeUntracked: true})
	require.NoError(t, err)

	assert.Equal(t, []string{"dirty.go", "feature.go", "untracked.txt"}, relPaths(t, root, got))
}

func TestChangesWithoutBaseSeesWorkingTreeOnly(t *testing.T) {
	root := initRepo(t)

	got, err := Changes(t.Context(), Options{Root: root, IncludeUntracked: true})
	require.NoError(t, err)

	assert.Equal(t, []string{"dirty.go", "untracked.txt"}, relPaths(t, root, got))
}

func TestChangesExcludesUntrackedWhenDisabled(t *testing.T) {
	root := initRepo(t)

	got, err := Changes(t.Context(), Options{Root: root, Base: "main"})
	require.NoError(t, err)

	assert.NotContains(t, relPaths(t, root, got), "untracked.txt")
}

func TestChangesFailsOnUnknownRef(t *testing.T) {
	root := initRepo(t)

	_, err := Changes(t.Context(), Options{Root: root, Base: "no-such-ref"})

	require.Error(t, err)
}

func TestRepoRootResolvesFromSubdirectory(t *testing.T) {
	root := initRepo(t)
	sub := filepath.Join(root, "nested", "deep")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	got, err := RepoRoot(t.Context(), sub)
	require.NoError(t, err)

	wantResolved, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	gotResolved, err := filepath.EvalSymlinks(got)
	require.NoError(t, err)
	assert.Equal(t, wantResolved, gotResolved)
}
