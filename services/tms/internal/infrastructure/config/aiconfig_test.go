package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
An absent switch means enabled.

The gate used to be documentIntelligence.enableAI, which ships false. Somebody
who configured a provider under AI Control got "AI features are disabled" from
a key named after a feature they were not using, with nothing in the message to
say which switch was refusing them.

A master switch that defaults off makes the product broken on arrival, and it
guards nothing: the router can only reach a provider row an organization added
and enabled, with its own credentials. So absent means available, and this is
the switch for an operator who wants to stop all of it at once.
*/
func TestAIConfig_AbsentMeansEnabled(t *testing.T) {
	t.Parallel()

	var unset config.AIConfig

	assert.True(t, unset.AIEnabled(), "omitting the key must not disable the assistant")
}

func TestAIConfig_HonoursAnExplicitChoice(t *testing.T) {
	t.Parallel()

	off, on := false, true

	assert.False(t, (&config.AIConfig{Enabled: &off}).AIEnabled())
	assert.True(t, (&config.AIConfig{Enabled: &on}).AIEnabled())
}

// The message has to name the key. "AI features are disabled" on its own sent
// somebody looking through a configuration file for a switch whose name gives
// no hint that it is the one.
func TestAIEnabledKey_IsTheKeyOperatorsSearchFor(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ai.enabled", config.AIEnabledKey)
}

// A service built without the section at all must not panic inside the gate. A
// test constructing the router directly did exactly that, and a gate that can
// take the process down is worse than the misconfiguration it guards against.
func TestAIConfig_NilReceiverIsEnabled(t *testing.T) {
	t.Parallel()

	var missing *config.AIConfig

	assert.NotPanics(t, func() {
		assert.True(t, missing.AIEnabled())
	})
}

// An operator upgrading past the merge has documentIntelligence: in their
// file. An exact unmarshal already refuses the unknown key, but its message
// reads as a typo; the settings were moved, and saying so is the difference
// between a five-minute edit and an afternoon.
func TestLoad_SaysWhereTheDocumentIntelligenceSectionWent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.test.yaml")
	require.NoError(t, os.WriteFile(path, []byte(
		"documentIntelligence:\n  enableAI: true\n  ocrCommand: tesseract\n",
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
