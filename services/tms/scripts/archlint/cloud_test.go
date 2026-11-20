package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnlyBootstrapImportsTheCommercialDirectory(t *testing.T) {
	t.Parallel()

	problems, err := cloudImporters(filepath.Join("..", ".."))
	require.NoError(t, err)

	assert.Empty(t, problems,
		"public code never imports internal/cloud (its own license); compose it from "+
			"internal/bootstrap and reach it through a port in core:\n%s",
		strings.Join(problems, "\n"))
}

func TestACoreImportOfTheCommercialDirectoryIsReported(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	}
	write("internal/modules/kernel/app/leak/leak.go",
		"package leak\n\nimport _ \""+cloudImport+"/tenantlifecycle\"\n")
	write("internal/bootstrap/core/core.go",
		"package core\n\nimport _ \""+cloudImport+"\"\n")
	write("internal/cloud/tenantlifecycle/service.go",
		"package tenantlifecycle\n\nimport _ \""+cloudImport+"/cloudtenant\"\n")

	problems, err := cloudImporters(root)
	require.NoError(t, err)
	require.Len(t, problems, 1)
	assert.Contains(t, problems[0], "internal/modules/kernel/app/leak/leak.go:3")
}
