package cache

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}

	return root
}

func baseTree() map[string]string {
	return map[string]string{
		"go.mod":         "module example.com/x\n\ngo 1.26\n",
		"svc/svc.go":     "package svc\n",
		"repo/repo.go":   "package repo\n",
		"README.md":      "# docs\n",
		"svc/notes.txt":  "scratch\n",
		".git/HEAD":      "ref: refs/heads/main\n",
		".git/objects/a": "junk\n",
	}
}

func compute(t *testing.T, root string, tags ...string) Fingerprint {
	t.Helper()

	fp, err := Compute(t.Context(), Inputs{Root: root, Tags: tags, Env: []string{"GOOS=linux"}})
	require.NoError(t, err)

	return fp
}

func TestFingerprintIsDeterministic(t *testing.T) {
	root := writeTree(t, baseTree())

	assert.Equal(t, compute(t, root), compute(t, root))
}

func TestFingerprintChangesWhenGoFileChanges(t *testing.T) {
	root := writeTree(t, baseTree())
	before := compute(t, root)

	require.NoError(t, os.WriteFile(
		filepath.Join(root, "svc", "svc.go"), []byte("package svc\n\nfunc F() {}\n"), 0o644))

	assert.NotEqual(t, before, compute(t, root))
}

func TestFingerprintChangesWhenModuleFileChanges(t *testing.T) {
	root := writeTree(t, baseTree())
	before := compute(t, root)

	require.NoError(t, os.WriteFile(
		filepath.Join(root, "go.mod"), []byte("module example.com/x\n\ngo 1.26\n\nrequire x v1.0.0\n"), 0o644))

	assert.NotEqual(t, before, compute(t, root))
}

func TestFingerprintChangesWhenGoFileAppears(t *testing.T) {
	root := writeTree(t, baseTree())
	before := compute(t, root)

	require.NoError(t, os.WriteFile(filepath.Join(root, "svc", "extra.go"), []byte("package svc\n"), 0o644))

	assert.NotEqual(t, before, compute(t, root))
}

func TestFingerprintChangesWhenGoFileDisappears(t *testing.T) {
	root := writeTree(t, baseTree())
	before := compute(t, root)

	require.NoError(t, os.Remove(filepath.Join(root, "repo", "repo.go")))

	assert.NotEqual(t, before, compute(t, root))
}

func TestFingerprintIgnoresIrrelevantFiles(t *testing.T) {
	root := writeTree(t, baseTree())
	before := compute(t, root)

	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("# rewritten\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "svc", "notes.txt"), []byte("other\n"), 0o644))

	assert.Equal(t, before, compute(t, root),
		"documentation and scratch files cannot change the package graph")
}

func TestFingerprintSkipsDotDirectoriesAndNodeModules(t *testing.T) {
	root := writeTree(t, baseTree())
	before := compute(t, root)

	require.NoError(t, os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "node_modules", "pkg", "index.go"), []byte("package pkg\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "extra.go"), []byte("package git\n"), 0o644))

	assert.Equal(t, before, compute(t, root))
}

func TestFingerprintSeparatesBuildTags(t *testing.T) {
	root := writeTree(t, baseTree())

	plain := compute(t, root)
	tagged := compute(t, root, "integration")

	assert.NotEqual(t, plain, tagged)
	assert.Equal(t, tagged, compute(t, root, "integration"))
}

func TestFingerprintIgnoresTagOrder(t *testing.T) {
	root := writeTree(t, baseTree())

	assert.Equal(t, compute(t, root, "a", "b"), compute(t, root, "b", "a"))
}

func TestFingerprintSeparatesToolchainEnvironment(t *testing.T) {
	root := writeTree(t, baseTree())

	linux, err := Compute(t.Context(), Inputs{Root: root, Env: []string{"GOOS=linux"}})
	require.NoError(t, err)
	windows, err := Compute(t.Context(), Inputs{Root: root, Env: []string{"GOOS=windows"}})
	require.NoError(t, err)

	assert.NotEqual(t, linux, windows, "build constraints depend on GOOS")
}

func TestFingerprintSeparatesRoots(t *testing.T) {
	first := writeTree(t, baseTree())
	second := writeTree(t, baseTree())

	assert.NotEqual(t, compute(t, first), compute(t, second),
		"the graph stores absolute directories, so the root belongs in the key")
}

func TestFingerprintStringIsHex(t *testing.T) {
	root := writeTree(t, baseTree())

	assert.Len(t, compute(t, root).String(), 64)
}

func TestBuildIdentityIsNonEmptyAndPinsDevBuilds(t *testing.T) {
	got := BuildIdentity()

	assert.NotEmpty(t, got)
	assert.Contains(t, got, "exe=",
		"a dev build has no release version, so the executable itself must pin the key")
}

func TestScanExposesPerFileDigests(t *testing.T) {
	root := writeTree(t, baseTree())

	manifest, err := Scan(t.Context(), Inputs{Root: root, Env: []string{"GOOS=linux"}})
	require.NoError(t, err)

	rels := make([]string, 0, len(manifest.Files))
	for _, f := range manifest.Files {
		rels = append(rels, f.RelPath)
	}
	assert.Equal(t, []string{"go.mod", "repo/repo.go", "svc/svc.go"}, rels,
		"only graph-relevant files are digested, in sorted order")

	svc := filepath.Join(root, "svc", "svc.go")
	digest, ok := manifest.DigestFor(svc)
	require.True(t, ok)
	assert.NotEqual(t, Digest{}, digest)

	_, ok = manifest.DigestFor(filepath.Join(root, "README.md"))
	assert.False(t, ok)
}

func TestScanDigestsTrackContent(t *testing.T) {
	root := writeTree(t, baseTree())
	svc := filepath.Join(root, "svc", "svc.go")

	before, err := Scan(t.Context(), Inputs{Root: root})
	require.NoError(t, err)
	beforeDigest, ok := before.DigestFor(svc)
	require.True(t, ok)

	require.NoError(t, os.WriteFile(svc, []byte("package svc\n\nfunc F() {}\n"), 0o644))

	after, err := Scan(t.Context(), Inputs{Root: root})
	require.NoError(t, err)
	afterDigest, ok := after.DigestFor(svc)
	require.True(t, ok)

	assert.NotEqual(t, beforeDigest, afterDigest)
	repoDigestBefore, _ := before.DigestFor(filepath.Join(root, "repo", "repo.go"))
	repoDigestAfter, _ := after.DigestFor(filepath.Join(root, "repo", "repo.go"))
	assert.Equal(t, repoDigestBefore, repoDigestAfter, "untouched files keep their digest")
}

func TestScanKeyMatchesCompute(t *testing.T) {
	root := writeTree(t, baseTree())

	manifest, err := Scan(t.Context(), Inputs{Root: root, Env: []string{"GOOS=linux"}})
	require.NoError(t, err)

	assert.Equal(t, compute(t, root), manifest.Key)
}

func TestComputeFailsOnMissingRoot(t *testing.T) {
	_, err := Compute(t.Context(), Inputs{Root: filepath.Join(t.TempDir(), "absent")})

	require.Error(t, err)
}

func TestScanStopsOnCancelledContext(t *testing.T) {
	root := writeTree(t, baseTree())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := Scan(ctx, Inputs{Root: root})

	require.ErrorIs(t, err, context.Canceled,
		"a cancelled run must surface the cancellation, not a partial fingerprint")
}
