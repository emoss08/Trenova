package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/flake"
)

// jumpySource fails on every second run: the persisted counter makes the
// flakiness deterministic, so the hunt must catch it without luck.
const jumpySource = `package jumpy

import (
	"os"
	"testing"
)

func TestJumpy(t *testing.T) {
	data, _ := os.ReadFile("counter.txt")
	if err := os.WriteFile("counter.txt", append(data, 'x'), 0o644); err != nil {
		t.Fatal(err)
	}
	if len(data)%2 == 1 {
		t.Fatal("deliberately failing every second run")
	}
}

func TestSteady(t *testing.T) {}
`

// TestHuntCatchesADeterministicallyFlakyTest pins hunt end to end: build the
// binary once, run it repeatedly at one tree state, attribute per-test
// verdicts, and let detection find the same-fingerprint conflict — while the
// steady test stays clean.
func TestHuntCatchesADeterministicallyFlakyTest(t *testing.T) {
	t.Setenv("GOWORK", "off")

	root := t.TempDir()
	dir := filepath.Join(root, "jumpy")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.com/hunt\n\ngo 1.26\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jumpy_test.go"), []byte(jumpySource), 0o644))

	observations, err := huntPackage(t.Context(), huntRun{
		root:             root,
		importPath:       "example.com/hunt/jumpy",
		dir:              dir,
		workdir:          t.TempDir(),
		runs:             4,
		timeout:          time.Minute,
		buildParallelism: 2,
		fingerprint:      "fp-hunt",
	})
	require.NoError(t, err)
	require.NotEmpty(t, observations)

	flakes := flake.Detect(observations)
	require.Len(t, flakes, 1, "the alternating test is flaky, the steady one is not")
	assert.Equal(t, "TestJumpy", flakes[0].Test)
	assert.Equal(t, 2, flakes[0].Passes())
	assert.Equal(t, 2, flakes[0].Fails())
}

// TestHuntSkipsPackagesWithoutTests pins the empty case: a package whose build
// produces no test binary records nothing rather than erroring the hunt.
func TestHuntSkipsPackagesWithoutTests(t *testing.T) {
	t.Setenv("GOWORK", "off")

	root := t.TempDir()
	dir := filepath.Join(root, "bare")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.com/hunt\n\ngo 1.26\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bare.go"),
		[]byte("package bare\n\nfunc Value() int { return 1 }\n"), 0o644))

	observations, err := huntPackage(t.Context(), huntRun{
		root:             root,
		importPath:       "example.com/hunt/bare",
		dir:              dir,
		workdir:          t.TempDir(),
		runs:             3,
		timeout:          time.Minute,
		buildParallelism: 2,
		fingerprint:      "fp-bare",
	})

	require.NoError(t, err)
	assert.Empty(t, observations)
}
