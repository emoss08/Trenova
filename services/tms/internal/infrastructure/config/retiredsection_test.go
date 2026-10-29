package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An operator upgrading past the merge has documentIntelligence: in their
// file. An exact unmarshal already refuses the unknown key, but its message
// reads as a typo; the settings were moved, and saying so is the difference
// between a five-minute edit and an afternoon.
func TestLoad_SaysWhereTheDocumentIntelligenceSectionWent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.test.yaml")
	require.NoError(t, os.WriteFile(path, []byte(
		"documentIntelligence:\n  enableAI: true\n  aiTimeout: 20s\n",
	), 0o600))

	loader := config.NewLoader(
		config.WithConfigPath(dir),
		config.WithEnvironment("test"),
	)
	_, err := loader.Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "documentIntelligence")
	assert.Contains(t, err.Error(), "has been removed")
	assert.Contains(t, err.Error(), "ai.documentExtraction")
}
