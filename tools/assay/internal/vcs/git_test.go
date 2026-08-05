package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
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

func TestParseHunkHeader(t *testing.T) {
	cases := []struct {
		name string
		line string
		want LineRange
		ok   bool
	}{
		{"single added line", "@@ -0,0 +13 @@", LineRange{Start: 13, End: 13}, true},
		{"multiple added lines", "@@ -12,0 +13,4 @@", LineRange{Start: 13, End: 16}, true},
		{"with section heading", "@@ -8,3 +8,3 @@ func Do() {", LineRange{Start: 8, End: 10}, true},
		{"pure deletion widens", "@@ -20,2 +19,0 @@", LineRange{Start: 19, End: 20}, true},
		{"deletion at file start", "@@ -1,2 +0,0 @@", LineRange{Start: 1, End: 1}, true},
		{"not a hunk", "+++ b/x.go", LineRange{}, false},
		{"malformed", "@@ garbage @@", LineRange{}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseHunkHeader(tc.line)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestParseUnifiedHunks(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repo")
	payload := `diff --git a/svc/svc.go b/svc/svc.go
index 111..222 100644
--- a/svc/svc.go
+++ b/svc/svc.go
@@ -10,1 +10,2 @@ func Do() {
-old
+new
+extra
@@ -30,0 +32,1 @@
+tail
diff --git a/gone.go b/gone.go
deleted file mode 100644
--- a/gone.go
+++ /dev/null
@@ -1,3 +0,0 @@
-a
-b
-c
`

	got := parseUnifiedHunks(payload, root)

	svc := got[filepath.Join(root, "svc/svc.go")]
	assert.Equal(t, []LineRange{{Start: 10, End: 11}, {Start: 32, End: 32}}, svc.new)
	assert.Equal(t, []LineRange{{Start: 10, End: 10}, {Start: 30, End: 31}}, svc.base,
		"index lookups need base-commit coordinates, not working-tree ones")

	gone := got[filepath.Join(root, "gone.go")]
	assert.Empty(t, gone.new, "a file whose new side is /dev/null has no current lines")
}

func TestParseHunkSidesReportsBothCoordinateSystems(t *testing.T) {
	base, added, ok := parseHunkSides("@@ -40,3 +50,5 @@ func Do() {")

	require.True(t, ok)
	assert.Equal(t, LineRange{Start: 40, End: 42}, base)
	assert.Equal(t, LineRange{Start: 50, End: 54}, added)
}

func TestParseHunkSidesWidensZeroLengthSides(t *testing.T) {
	base, added, ok := parseHunkSides("@@ -12,0 +13,2 @@")

	require.True(t, ok)
	assert.Equal(t, LineRange{Start: 12, End: 13}, base,
		"a pure insertion has no base lines, so the insertion point is widened")
	assert.Equal(t, LineRange{Start: 13, End: 14}, added)
}

func TestChangeExposesBothLineNumberSets(t *testing.T) {
	c := Change{
		Lines:     []LineRange{{Start: 10, End: 12}},
		BaseLines: []LineRange{{Start: 7, End: 8}},
	}

	assert.Equal(t, []int{10, 11, 12}, c.LineNumbers())
	assert.Equal(t, []int{7, 8}, c.BaseLineNumbers())
}

func TestChangeWholeFileAndTouches(t *testing.T) {
	whole := Change{Path: "/x.go", Status: "A"}
	assert.True(t, whole.WholeFile())
	assert.False(t, whole.Touches(1, 100))

	partial := Change{Path: "/y.go", Status: "M", Lines: []LineRange{{Start: 10, End: 12}}}
	assert.False(t, partial.WholeFile())
	assert.True(t, partial.Touches(12, 20), "overlap at the boundary counts")
	assert.True(t, partial.Touches(1, 10))
	assert.False(t, partial.Touches(13, 20))
	assert.False(t, partial.Touches(1, 9))
}

func TestLineRangeContains(t *testing.T) {
	r := LineRange{Start: 5, End: 7}

	assert.True(t, r.Contains(5))
	assert.True(t, r.Contains(7))
	assert.False(t, r.Contains(4))
	assert.False(t, r.Contains(8))
}

func TestDedupeChangesSortsAndDropsDuplicates(t *testing.T) {
	got := dedupeChanges([]Change{
		{Path: "/b", Status: "M"},
		{Path: "/a", Status: "A"},
		{Path: "/b", Status: "D"},
	})

	assert.Equal(t, []Change{{Path: "/a", Status: "A"}, {Path: "/b", Status: "M"}}, got)
}

func TestIsWholeFileStatus(t *testing.T) {
	for _, status := range []string{"A", "D", "R100", "C75", "?", ""} {
		assert.True(t, isWholeFileStatus(status), "status %q", status)
	}
	for _, status := range []string{"M", "T"} {
		assert.False(t, isWholeFileStatus(status), "status %q", status)
	}
}

type repo struct {
	root string
	t    *testing.T
}

func (r repo) git(args ...string) {
	r.t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = r.root
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=assay", "GIT_AUTHOR_EMAIL=assay@example.com",
		"GIT_COMMITTER_NAME=assay", "GIT_COMMITTER_EMAIL=assay@example.com",
	)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(r.t, err, "git %v: %s", args, out)
}

func (r repo) gitOut(args ...string) string {
	r.t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = r.root
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=assay", "GIT_AUTHOR_EMAIL=assay@example.com",
		"GIT_COMMITTER_NAME=assay", "GIT_COMMITTER_EMAIL=assay@example.com",
	)
	out, err := cmd.Output()
	require.NoErrorf(r.t, err, "git %v", args)

	return strings.TrimSpace(string(out))
}

func (r repo) write(rel, content string) {
	r.t.Helper()

	full := filepath.Join(r.root, rel)
	require.NoError(r.t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(r.t, os.WriteFile(full, []byte(content), 0o644))
}

func initRepo(t *testing.T) repo {
	t.Helper()

	r := repo{root: t.TempDir(), t: t}

	r.git("init", "--initial-branch=main")
	r.write("base.go", "package base\n\nfunc A() int {\n\treturn 1\n}\n\nfunc B() int {\n\treturn 2\n}\n")
	r.write("doomed.go", "package base\n\nfunc Doomed() {}\n")
	r.git("add", ".")
	r.git("commit", "-m", "base")

	r.git("checkout", "-b", "feature")
	r.write("feature.go", "package feature\n")
	r.write("base.go", "package base\n\nfunc A() int {\n\treturn 99\n}\n\nfunc B() int {\n\treturn 2\n}\n")
	r.git("rm", "-q", "doomed.go")
	r.git("add", ".")
	r.git("commit", "-m", "feature")

	r.write("dirty.go", "package dirty\n")
	r.write("untracked.txt", "hello\n")
	r.git("add", "dirty.go")

	return r
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

func findChange(t *testing.T, res Result, root, rel string) Change {
	t.Helper()

	want := filepath.Join(root, filepath.FromSlash(rel))
	for _, c := range res.Changes {
		if c.Path == want {
			return c
		}
	}
	t.Fatalf("change for %s not found in %v", rel, relPaths(t, root, res.Changes))

	return Change{}
}

func TestChangesAgainstMergeBase(t *testing.T) {
	r := initRepo(t)

	got, err := Changes(t.Context(), Options{Root: r.root, Base: "main", IncludeUntracked: true})
	require.NoError(t, err)

	assert.Equal(t, BaseMergeBase, got.BaseMode)
	assert.Empty(t, got.Note)
	assert.Equal(t, []string{"base.go", "dirty.go", "doomed.go", "feature.go", "untracked.txt"},
		relPaths(t, r.root, got.Changes))
}

func TestChangesReportsModifiedLineRanges(t *testing.T) {
	r := initRepo(t)

	got, err := Changes(t.Context(), Options{Root: r.root, Base: "main"})
	require.NoError(t, err)

	modified := findChange(t, got, r.root, "base.go")
	require.False(t, modified.WholeFile(), "a modified file must carry line ranges")
	assert.True(t, modified.Touches(4, 4), "line 4 is the edited return; got %v", modified.Lines)
	assert.False(t, modified.Touches(8, 8), "func B was untouched; got %v", modified.Lines)
}

func TestChangesTreatsAddedAndDeletedFilesAsWholeFile(t *testing.T) {
	r := initRepo(t)

	got, err := Changes(t.Context(), Options{Root: r.root, Base: "main", IncludeUntracked: true})
	require.NoError(t, err)

	for _, rel := range []string{"feature.go", "doomed.go", "untracked.txt"} {
		assert.True(t, findChange(t, got, r.root, rel).WholeFile(),
			"%s has no usable coverage history and must be treated as a whole-file change", rel)
	}
}

func TestChangesWithoutBaseSeesWorkingTreeOnly(t *testing.T) {
	r := initRepo(t)

	got, err := Changes(t.Context(), Options{Root: r.root, IncludeUntracked: true})
	require.NoError(t, err)

	assert.Equal(t, BaseWorkingTree, got.BaseMode)
	assert.Equal(t, []string{"dirty.go", "untracked.txt"}, relPaths(t, r.root, got.Changes))
}

func TestChangesExcludesUntrackedWhenDisabled(t *testing.T) {
	r := initRepo(t)

	got, err := Changes(t.Context(), Options{Root: r.root, Base: "main"})
	require.NoError(t, err)

	assert.NotContains(t, relPaths(t, r.root, got.Changes), "untracked.txt")
}

func TestChangesFallsBackWhenNoMergeBaseExists(t *testing.T) {
	r := initRepo(t)

	const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	unrelated := r.gitOut("commit-tree", emptyTree, "-m", "unrelated root")
	r.git("branch", "unrelated", unrelated)

	got, err := Changes(t.Context(), Options{Root: r.root, Base: "unrelated"})
	require.NoError(t, err, "an unrelated base must degrade, not fail")

	assert.Equal(t, BaseTwoDot, got.BaseMode)
	assert.Contains(t, got.Note, "no merge base")
	assert.Contains(t, got.Note, "over-selects")
	assert.NotEmpty(t, got.Changes, "the fallback must still produce a change set")
}

func TestChangesFailsOnUnknownRefWithActionableMessage(t *testing.T) {
	r := initRepo(t)

	_, err := Changes(t.Context(), Options{Root: r.root, Base: "origin/nope"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot resolve base ref")
	assert.Contains(t, err.Error(), "origin/nope")
}

func TestChangesMentionsShallowCloneWhenRefIsMissing(t *testing.T) {
	r := initRepo(t)
	shallow := repo{root: t.TempDir(), t: t}
	shallow.git("clone", "--depth=1", "file://"+r.root, ".")

	require.True(t, IsShallow(t.Context(), shallow.root), "clone --depth=1 must be shallow")

	_, err := Changes(t.Context(), Options{Root: shallow.root, Base: "origin/main"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "shallow clone",
		"the error must name the actual cause a CI user will hit")
	assert.Contains(t, err.Error(), "--files")
}

func TestIsShallowIsFalseForNormalRepo(t *testing.T) {
	r := initRepo(t)

	assert.False(t, IsShallow(t.Context(), r.root))
}

func TestRepoRootResolvesFromSubdirectory(t *testing.T) {
	r := initRepo(t)
	sub := filepath.Join(r.root, "nested", "deep")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	got, err := RepoRoot(t.Context(), sub)
	require.NoError(t, err)

	wantResolved, err := filepath.EvalSymlinks(r.root)
	require.NoError(t, err)
	gotResolved, err := filepath.EvalSymlinks(got)
	require.NoError(t, err)
	assert.Equal(t, wantResolved, gotResolved)
}
