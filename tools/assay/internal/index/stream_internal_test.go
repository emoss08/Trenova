package index

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunHarnessSurvivesAnOversizedLine pins the deadlock fix: a single line
// over the scanner's 1MiB limit kills the reader, the child fills the pipe and
// blocks, and the old code had already disarmed the deadline before Wait — so
// `assay index` hung forever. The fix cancels and drains when the scanner dies,
// which must resolve the run promptly and fold the reason into the error.
func TestRunHarnessSurvivesAnOversizedLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub harness is a shell script")
	}

	dir := t.TempDir()
	stub := filepath.Join(dir, "harness.sh")
	script := "#!/bin/sh\n" +
		// One line well past the 1MiB scanner limit kills the reader.
		"head -c 2097152 /dev/zero | tr '\\0' 'x'\n" +
		"printf '\\n'\n" +
		// Then enough output to fill the now-unread pipe and block the writer.
		"head -c 1048576 /dev/zero | tr '\\0' 'y'\n" +
		"sleep 600\n"
	require.NoError(t, os.WriteFile(stub, []byte(script), 0o755))

	started := time.Now()
	result := runHarness(
		t.Context(),
		Options{Timeout: 5 * time.Minute},
		stub, dir, filepath.Join(dir, "list"), dir,
		[]string{"TestA"},
	)

	assert.Less(t, time.Since(started), 30*time.Second,
		"a dead reader must cancel and drain, not wait out the deadline")
	require.Error(t, result.err)
	assert.Equal(t, "TestA", result.stalled)
	assert.Contains(t, result.err.Error(), "unreadable",
		"the scanner's death is the reason and must not be swallowed")
}

// TestParseHarnessLineSurvivesPartialLineGlue pins the fix for tests that print
// without a trailing newline: the harness's progress line is glued onto their
// fragment, and requiring the prefix at the line start silently pushed such
// packages onto the salvage path.
func TestParseHarnessLineSurvivesPartialLineGlue(t *testing.T) {
	glued := "some test output without a newline" + harnessLinePrefix + "TestX exit=0 ns=1200"

	progress, ok := parseHarnessLine(glued, "TestX")

	require.True(t, ok)
	assert.Equal(t, "TestX", progress.name)
	assert.Zero(t, progress.exit)
	assert.Equal(t, time.Duration(1200), progress.duration)
}

func TestParseHarnessLineRejectsForgedNames(t *testing.T) {
	forged := harnessLinePrefix + "TestOther exit=0 ns=5"

	_, ok := parseHarnessLine(forged, "TestExpected")

	assert.False(t, ok, "a test printing a progress line for the wrong name must not desync the collector")
}

// TestListTestsDedupesInternalAndExternalNames pins the fix for a top-level name
// declared in both the internal and the external test package: -test.list prints
// it twice, and keeping both ran one counter window for two tests and recorded
// the result twice under one name.
func TestListTestsDedupesInternalAndExternalNames(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub binary is a shell script")
	}

	dir := t.TempDir()
	stub := filepath.Join(dir, "binary.sh")
	script := "#!/bin/sh\n" +
		"printf 'TestShared\\nTestShared\\nTestOnly\\nnotatest\\nok  example.com/x 0.01s\\n'\n"
	require.NoError(t, os.WriteFile(stub, []byte(script), 0o755))

	names, err := listTests(t.Context(), Options{}, stub, dir)

	require.NoError(t, err)
	assert.Equal(t, []string{"TestOnly", "TestShared"}, names)
}
