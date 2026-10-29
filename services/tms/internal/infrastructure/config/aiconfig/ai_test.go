package aiconfig_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/stretchr/testify/assert"
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

	var unset aiconfig.Config

	assert.True(t, unset.AIEnabled(), "omitting the key must not disable the assistant")
}

func TestAIConfig_HonoursAnExplicitChoice(t *testing.T) {
	t.Parallel()

	off, on := false, true

	assert.False(t, (&aiconfig.Config{Enabled: &off}).AIEnabled())
	assert.True(t, (&aiconfig.Config{Enabled: &on}).AIEnabled())
}

// The message has to name the key. "AI features are disabled" on its own sent
// somebody looking through a configuration file for a switch whose name gives
// no hint that it is the one.
func TestAIEnabledKey_IsTheKeyOperatorsSearchFor(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ai.enabled", aiconfig.EnabledKey)
}

// A service built without the section at all must not panic inside the gate. A
// test constructing the router directly did exactly that, and a gate that can
// take the process down is worse than the misconfiguration it guards against.
func TestAIConfig_NilReceiverIsEnabled(t *testing.T) {
	t.Parallel()

	var missing *aiconfig.Config

	assert.NotPanics(t, func() {
		assert.True(t, missing.AIEnabled())
	})
}

// Cache diagnostics ask the Claude API to keep a fingerprint of every call, so
// they stay off unless an operator turns them on, including when there is no
// AI section at all.
func TestAIConfig_CacheDiagnosticsAreOffUnlessTurnedOn(t *testing.T) {
	t.Parallel()

	var absent *aiconfig.Config
	assert.False(t, absent.CacheDiagnosticsEnabled())
	assert.False(t, (&aiconfig.Config{}).CacheDiagnosticsEnabled())
	assert.True(t, (&aiconfig.Config{CacheDiagnostics: true}).CacheDiagnosticsEnabled())
}

func TestAIConfig_ToolDisclosureThresholdDefaultsToTwelve(t *testing.T) {
	t.Parallel()

	var absent *aiconfig.Config
	assert.Equal(t, aiconfig.DefaultToolDisclosureThreshold, absent.GetToolDisclosureThreshold())
	assert.Equal(t, 12, (&aiconfig.Config{}).GetToolDisclosureThreshold())

	zero, thirty, negative := 0, 30, -1
	assert.Equal(
		t,
		0,
		(&aiconfig.Config{ToolDisclosureThreshold: &zero}).GetToolDisclosureThreshold(),
	)
	assert.Equal(
		t,
		30,
		(&aiconfig.Config{ToolDisclosureThreshold: &thirty}).GetToolDisclosureThreshold(),
	)
	assert.Equal(t, 12,
		(&aiconfig.Config{ToolDisclosureThreshold: &negative}).GetToolDisclosureThreshold())
}
