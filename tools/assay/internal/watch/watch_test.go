package watch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/cache"
)

const watchModule = "example.com/watched"

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

func (r repo) write(rel, content string) {
	r.t.Helper()

	full := filepath.Join(r.root, filepath.FromSlash(rel))
	require.NoError(r.t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(r.t, os.WriteFile(full, []byte(content), 0o644))
}

func (r repo) path(rel string) string {
	return filepath.Join(r.root, filepath.FromSlash(rel))
}

const alphaSource = `package alpha

func Double(n int) int {
	doubled := n * 2

	return doubled
}
`

const alphaTest = `package alpha

import "testing"

func TestDouble(t *testing.T) {
	if Double(2) != 4 {
		t.Fatal("bad")
	}
}

func TestDoubleZero(t *testing.T) {
	if Double(0) != 0 {
		t.Fatal("bad")
	}
}
`

const betaSource = `package beta

func Greet(name string) string {
	greeting := "hello " + name

	return greeting
}
`

const betaTest = `package beta

import "testing"

func TestGreet(t *testing.T) {
	if Greet("x") != "hello x" {
		t.Fatal("bad")
	}
}
`

func newRepo(t *testing.T) repo {
	t.Helper()
	t.Setenv("GOWORK", "off")

	r := repo{root: t.TempDir(), t: t}
	r.write("go.mod", "module "+watchModule+"\n\ngo 1.26\n")
	r.write("alpha/alpha.go", alphaSource)
	r.write("alpha/alpha_test.go", alphaTest)
	r.write("beta/beta.go", betaSource)
	r.write("beta/beta_test.go", betaTest)
	r.git("init", "--initial-branch=main")
	r.git("add", ".")
	r.git("commit", "-qm", "base")

	return r
}

func planner(r repo) *Planner {
	return &Planner{Root: r.root, Env: append(os.Environ(), "GOWORK=off")}
}

func TestCleanTreePlansNothing(t *testing.T) {
	r := newRepo(t)

	cycle, err := planner(r).Plan(t.Context(), nil)
	require.NoError(t, err)

	assert.Empty(t, cycle.Runs, "an unchanged tree has nothing to run")
}

func TestSaveRunsOnlyTheAffectedPackage(t *testing.T) {
	r := newRepo(t)
	r.write("alpha/alpha.go", strings.Replace(alphaSource, "n * 2", "n + n", 1))
	r.write("beta/beta.go", strings.Replace(betaSource, "hello ", "hi ", 1))

	cycle, err := planner(r).Plan(t.Context(), []string{r.path("alpha/alpha.go")})
	require.NoError(t, err)

	require.Len(t, cycle.Runs, 1, "the save touched alpha; beta stays dirty but deferred")
	assert.Equal(t, watchModule+"/alpha", cycle.Runs[0].ImportPath)
	assert.Equal(t, 1, cycle.Deferred)
}

func TestEmptyBatchRunsTheWholeDirtyPlan(t *testing.T) {
	r := newRepo(t)
	r.write("alpha/alpha.go", strings.Replace(alphaSource, "n * 2", "n + n", 1))
	r.write("beta/beta.go", strings.Replace(betaSource, "hello ", "hi ", 1))

	cycle, err := planner(r).Plan(t.Context(), nil)
	require.NoError(t, err)

	assert.Len(t, cycle.Runs, 2)
	assert.Zero(t, cycle.Deferred)
}

// TestEditInsideOneTestRunsOnlyThatTest pins the refinement that makes watch
// feel right during test-driven work: editing TestDouble's body must run
// TestDouble, not the whole package.
func TestEditInsideOneTestRunsOnlyThatTest(t *testing.T) {
	r := newRepo(t)
	r.write("alpha/alpha_test.go", strings.Replace(alphaTest, "!= 4", "!= 2+2", 1))

	cycle, err := planner(r).Plan(t.Context(), []string{r.path("alpha/alpha_test.go")})
	require.NoError(t, err)

	require.Len(t, cycle.Runs, 1)
	assert.Equal(t, []string{"TestDouble"}, cycle.Runs[0].Tests)
	assert.NotEmpty(t, cycle.Runs[0].Pattern)
}

// TestEditOutsideTestBodiesRunsThePackageInFull pins the refinement's guard: a
// change to a helper or import can affect any test in the package, so the
// refinement must decline rather than guess.
func TestEditOutsideTestBodiesRunsThePackageInFull(t *testing.T) {
	r := newRepo(t)
	r.write("alpha/alpha_test.go", "package alpha\n\nimport \"testing\"\n\nvar shared = 4\n\n"+
		"func TestDouble(t *testing.T) {\n\tif Double(2) != shared {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n\n"+
		"func TestDoubleZero(t *testing.T) {\n\tif Double(0) != 0 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n")

	cycle, err := planner(r).Plan(t.Context(), []string{r.path("alpha/alpha_test.go")})
	require.NoError(t, err)

	require.Len(t, cycle.Runs, 1)
	assert.Empty(t, cycle.Runs[0].Tests, "a package-level edit must run the whole package")
	assert.Empty(t, cycle.Runs[0].Pattern)
}

// TestRefinementDeclinesWhenPackageIsAlsoADependent guards the second condition:
// a test-file edit does not exempt a package that some other change also
// reaches — those tests must run against the changed dependency in full.
func TestRefinementDeclinesWhenPackageIsAlsoADependent(t *testing.T) {
	r := newRepo(t)
	r.write("beta/beta.go", "package beta\n\nimport \"example.com/watched/alpha\"\n\n"+
		"func Greet(name string) string {\n\tgreeting := \"hello \" + name\n\n\treturn greeting\n}\n\n"+
		"func Doubled(n int) int {\n\treturn alpha.Double(n)\n}\n")
	r.git("add", ".")
	r.git("commit", "-qm", "beta depends on alpha")

	r.write("alpha/alpha.go", strings.Replace(alphaSource, "n * 2", "n + n", 1))
	r.write("beta/beta_test.go", strings.Replace(betaTest, "hello x", "hello x", 1)+
		"\nfunc TestGreetEmpty(t *testing.T) {\n\tif Greet(\"\") != \"hello \" {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n")

	cycle, err := planner(r).Plan(t.Context(), []string{r.path("beta/beta_test.go")})
	require.NoError(t, err)

	for _, run := range cycle.Runs {
		if run.ImportPath == watchModule+"/beta" {
			assert.NotEqual(t, "test-function edit", run.Reason,
				"beta is also a dependent of the alpha change and must not be narrowed by the test edit")
		}
	}
}

func TestBinaryCacheReusesUntilTheClosureChanges(t *testing.T) {
	r := newRepo(t)
	c, err := NewBinaryCache()
	require.NoError(t, err)
	t.Cleanup(c.Close)

	req := BuildRequest{
		Root:        r.root,
		Env:         append(os.Environ(), "GOWORK=off"),
		ImportPath:  watchModule + "/alpha",
		Fingerprint: cache.Fingerprint{1},
	}

	first, reused, err := c.Ensure(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, reused)
	assert.FileExists(t, first)

	second, reused, err := c.Ensure(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, reused, "an identical fingerprint must not pay a rebuild")
	assert.Equal(t, first, second)

	req.Fingerprint = cache.Fingerprint{2}
	_, reused, err = c.Ensure(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, reused, "a changed dependency closure must rebuild")
}

func TestWatcherBatchesDebouncedSaves(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o644))

	w, err := NewWatcher(dir, 100*time.Millisecond)
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go w.Run(ctx)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n\nvar X = 1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go~"), []byte("backup"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("irrelevant"), 0o644))

	select {
	case batch := <-w.Batches():
		require.Len(t, batch.Paths, 1, "editor noise and non-Go files must not appear: %v", batch.Paths)
		assert.Equal(t, filepath.Join(dir, "a.go"), batch.Paths[0])
		assert.False(t, batch.Resync)
	case <-time.After(5 * time.Second):
		t.Fatal("no batch arrived")
	}
}

func TestWatcherSeesFilesInNewDirectories(t *testing.T) {
	dir := t.TempDir()

	w, err := NewWatcher(dir, 100*time.Millisecond)
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go w.Run(ctx)

	sub := filepath.Join(dir, "newpkg")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	time.Sleep(150 * time.Millisecond)
	require.NoError(t, os.WriteFile(filepath.Join(sub, "new.go"), []byte("package newpkg\n"), 0o644))

	select {
	case batch := <-w.Batches():
		assert.Contains(t, batch.Paths, filepath.Join(sub, "new.go"),
			"a package created mid-session must be watched")
	case <-time.After(5 * time.Second):
		t.Fatal("no batch arrived for the new directory")
	}
}
