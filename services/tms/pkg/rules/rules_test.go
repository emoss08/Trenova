package rules

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRuntime(t *testing.T) *Runtime {
	t.Helper()
	r := New(wasmmodule.Options{PoolSize: 1})
	t.Cleanup(func() {
		assert.NoError(t, r.Close(context.WithoutCancel(t.Context())))
	})
	return r
}

func TestEmbeddedModuleMatchesItsChecksumAndTheClientCopy(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256(moduleBytes)
	want := hex.EncodeToString(sum[:]) + "  rules.wasm\n"

	sidecar, err := os.ReadFile("rules.wasm.sha256")
	require.NoError(t, err)
	assert.Equal(t, want, string(sidecar))

	clientDir := filepath.Join("..", "..", "..", "..", "client", "packages", "shared", "src", "rules")
	clientModule, err := os.ReadFile(filepath.Join(clientDir, "rules.wasm"))
	require.NoError(t, err)
	assert.True(t, bytes.Equal(moduleBytes, clientModule), "the client copy of rules.wasm differs")

	clientSidecar, err := os.ReadFile(filepath.Join(clientDir, "rules.wasm.sha256"))
	require.NoError(t, err)
	assert.Equal(t, want, string(clientSidecar))
}

func TestVersionReportsTheCrateVersionAndItsOperations(t *testing.T) {
	t.Parallel()

	manifest, err := os.ReadFile(filepath.Join("..", "..", "..", "Cargo.toml"))
	require.NoError(t, err)
	match := regexp.MustCompile(`(?m)^version = "([^"]+)"`).FindSubmatch(manifest)
	require.NotNil(t, match, "services/Cargo.toml names no workspace version")

	info, err := newTestRuntime(t).Version(t.Context())
	require.NoError(t, err)

	assert.Equal(t, string(match[1]), info.Version)
	assert.Contains(t, info.Operations, OpVersion)
	assert.Contains(t, info.Operations, "tariff.evaluate")
}

func TestTheDefaultRuntimeAnswers(t *testing.T) {
	t.Parallel()

	info, err := Version(t.Context())
	require.NoError(t, err)
	assert.Contains(t, info.Operations, OpVersion)

	var version wasmmodule.VersionInfo
	require.NoError(t, Call(t.Context(), OpVersion, nil, &version))
	assert.Equal(t, info, version)

	raw, err := CallRaw(t.Context(), OpVersion, []byte(`{}`))
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(raw, []byte(`{"version":"`)), string(raw))
}
