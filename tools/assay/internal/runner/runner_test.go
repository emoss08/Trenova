package runner

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunksSplitsEvenly(t *testing.T) {
	var got [][]string
	for chunk := range chunks([]string{"a", "b", "c", "d", "e"}, 2) {
		got = append(got, slices.Clone(chunk))
	}

	assert.Equal(t, [][]string{{"a", "b"}, {"c", "d"}, {"e"}}, got)
}

func TestChunksHandlesEmptyInput(t *testing.T) {
	var count int
	for range chunks(nil, 4) {
		count++
	}

	assert.Zero(t, count)
}

func TestChunksStopsOnEarlyBreak(t *testing.T) {
	var seen int
	for range chunks([]string{"a", "b", "c", "d"}, 1) {
		seen++

		break
	}

	assert.Equal(t, 1, seen)
}

func TestRunWithNoPackagesIsANoop(t *testing.T) {
	code, err := Run(t.Context(), Options{Root: t.TempDir()})

	require.NoError(t, err)
	assert.Zero(t, code)
}

func writeModule(t *testing.T, passing, failing int) string {
	t.Helper()

	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}

	write("go.mod", "module example.com/run\n\ngo 1.26\n")
	for i := range passing {
		name := "ok" + strconv.Itoa(i)
		write(name+"/"+name+"_test.go",
			"package "+name+"\n\nimport \"testing\"\n\nfunc TestOK(t *testing.T) {}\n")
	}
	for i := range failing {
		name := "bad" + strconv.Itoa(i)
		write(name+"/"+name+"_test.go",
			"package "+name+"\n\nimport \"testing\"\n\nfunc TestBad(t *testing.T) { t.Fatal(\"boom\") }\n")
	}

	return root
}

func packagePaths(prefix string, n int) []string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, "example.com/run/"+prefix+strconv.Itoa(i))
	}

	return out
}

func TestRunReportsSuccess(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := writeModule(t, 2, 0)

	var out strings.Builder
	code, err := Run(t.Context(), Options{
		Root:   root,
		Groups: []Group{{Packages: packagePaths("ok", 2)}},
		Stdout: &out,
		Stderr: io.Discard,
	})

	require.NoError(t, err)
	assert.Zero(t, code)
	assert.Contains(t, out.String(), "ok")
}

func TestRunPropagatesFailureExitCode(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := writeModule(t, 1, 1)

	code, err := Run(t.Context(), Options{
		Root:   root,
		Groups: []Group{{Packages: append(packagePaths("ok", 1), packagePaths("bad", 1)...)}},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})

	require.NoError(t, err, "a failing test suite is a non-zero exit code, not a runner error")
	assert.Equal(t, 1, code)
}

func TestRunSurfacesWorstCodeAcrossChunks(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := writeModule(t, 1, 1)

	packages := append(packagePaths("ok", 1), packagePaths("bad", 1)...)
	slices.Sort(packages)

	code, err := Run(t.Context(), Options{
		Root:   root,
		Groups: []Group{{Packages: packages}},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})

	require.NoError(t, err)
	assert.NotZero(t, code, "a failure in any chunk must surface")
}

// TestSignalDeathIsAnAbortNotAVerdict pins the fix for the worst pair of
// misreadings in the tool: a signal-killed go test reported ExitCode -1, which
// `assay run` folded into "worst code 0" (success) and `assay verify` read as a
// failing excluded test — printing its UNSOUND banner for a Ctrl-C.
func TestSignalDeathIsAnAbortNotAVerdict(t *testing.T) {
	killed := exec.Command("sh", "-c", "kill -KILL $$")
	runErr := killed.Run()
	require.Error(t, runErr, "the shell must have died by signal for this test to mean anything")

	code, err := exitVerdict(runErr)

	require.Error(t, err, "signal death is an abort, not a test verdict")
	assert.Zero(t, code)
}

func TestExitVerdictKeepsRealFailureCodes(t *testing.T) {
	failing := exec.Command("sh", "-c", "exit 3")

	code, err := exitVerdict(failing.Run())

	require.NoError(t, err)
	assert.Equal(t, 3, code)
}

// TestConcurrentInvocationsKeepOrderedOutput pins both halves of the parallel
// runner: distinct -run groups actually overlap in time, and their output still
// reads as one sequential log because later invocations buffer until the front
// finishes.
func TestConcurrentInvocationsKeepOrderedOutput(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := t.TempDir()

	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	write("go.mod", "module example.com/run\n\ngo 1.26\n")
	write("slow/slow_test.go", "package slow\n\nimport (\n\t\"testing\"\n\t\"time\"\n)\n\n"+
		"func TestSlow(t *testing.T) { time.Sleep(700 * time.Millisecond) }\n")
	write("fast/fast_test.go", "package fast\n\nimport \"testing\"\n\nfunc TestFast(t *testing.T) {}\n")

	var out safeBuilder
	code, err := Run(t.Context(), Options{
		Root: root,
		Groups: []Group{
			{Packages: []string{"example.com/run/slow"}, Run: "^TestSlow$"},
			{Packages: []string{"example.com/run/fast"}, Run: "^TestFast$"},
		},
		Jobs:   2,
		Stdout: &out,
		Stderr: io.Discard,
	})

	require.NoError(t, err)
	assert.Zero(t, code)

	text := out.String()
	slowAt := strings.Index(text, "example.com/run/slow")
	fastAt := strings.Index(text, "example.com/run/fast")
	require.GreaterOrEqual(t, slowAt, 0)
	require.GreaterOrEqual(t, fastAt, 0)
	assert.Less(t, slowAt, fastAt,
		"the fast invocation finishes first but must not print before the slower front")
}

type safeBuilder struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *safeBuilder) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.b.Write(p)
}

func (s *safeBuilder) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.b.String()
}

func TestPromotedWriterFlushesBufferInOrder(t *testing.T) {
	var out strings.Builder
	w := &promotedWriter{out: &out}

	_, err := w.Write([]byte("buffered "))
	require.NoError(t, err)
	assert.Empty(t, out.String(), "nothing may reach the shared writer before promotion")

	w.promote()
	assert.Equal(t, "buffered ", out.String())

	_, err = w.Write([]byte("live"))
	require.NoError(t, err)
	assert.Equal(t, "buffered live", out.String())
}

func TestPromoteWithoutWriterDropsTheBuffer(t *testing.T) {
	w := &promotedWriter{}

	_, err := w.Write([]byte("before promotion"))
	require.NoError(t, err)
	w.promote()
	assert.Zero(t, w.buf.Len(), "promotion must release the buffer even with no destination")

	_, err = w.Write([]byte("after promotion"))
	require.NoError(t, err)
	assert.Zero(t, w.buf.Len(), "a promoted writer with no destination must discard, not accumulate")
}

// TestAbortStillFlushesBufferedOutput pins the fix for the cruellest way to lose
// a failure: the front invocation dies by signal, Wait returns an error, and the
// old code returned before the flush loop — throwing away every buffered line
// from invocations that had already finished, which was exactly the output the
// user needed to see.
//
// The front test kills its own `go test` parent, but only after the fast
// invocation has run and had time to land its output in the buffer, so the
// abort is deterministic rather than a wall-clock race.
func TestAbortStillFlushesBufferedOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture aborts its parent with a unix signal")
	}
	t.Setenv("GOWORK", "off")
	root := t.TempDir()

	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	sentinel := filepath.Join(root, "fast-done")
	write("go.mod", "module example.com/run\n\ngo 1.26\n")
	write("abort/abort_test.go", `package abort

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestAbort(t *testing.T) {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(`+strconv.Quote(sentinel)+`); err == nil {
			time.Sleep(500 * time.Millisecond)
			_ = syscall.Kill(os.Getppid(), syscall.SIGKILL)
			os.Exit(0)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("sentinel never appeared")
}
`)
	write("fast/fast_test.go", `package fast

import (
	"os"
	"testing"
)

func TestFast(t *testing.T) {
	if err := os.WriteFile(`+strconv.Quote(sentinel)+`, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}
`)

	var out safeBuilder
	_, err := Run(t.Context(), Options{
		Root: root,
		Groups: []Group{
			{Packages: []string{"example.com/run/abort"}, Run: "^TestAbort$"},
			{Packages: []string{"example.com/run/fast"}, Run: "^TestFast$"},
		},
		Jobs:      2,
		ExtraArgs: []string{"-v"},
		Stdout:    &out,
		Stderr:    io.Discard,
	})

	require.Error(t, err, "a signal-killed go test is an abort, not a verdict")
	assert.Contains(t, out.String(), "--- PASS: TestFast",
		"output buffered behind the aborted front must still be flushed")
}

func TestRunForwardsExtraArguments(t *testing.T) {
	t.Setenv("GOWORK", "off")
	root := writeModule(t, 1, 0)

	var out strings.Builder
	code, err := Run(t.Context(), Options{
		Root:      root,
		Groups:    []Group{{Packages: packagePaths("ok", 1)}},
		ExtraArgs: []string{"-v", "-count=1"},
		Stdout:    &out,
		Stderr:    io.Discard,
	})

	require.NoError(t, err)
	assert.Zero(t, code)
	assert.Contains(t, out.String(), "--- PASS: TestOK", "-v must reach go test")
}
