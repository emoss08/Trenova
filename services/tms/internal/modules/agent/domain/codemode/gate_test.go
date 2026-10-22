package codemode_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/stretchr/testify/assert"
)

var (
	off     = aiprovider.ScriptSettings{}
	reads   = aiprovider.ScriptSettings{Scripts: true}
	batches = aiprovider.ScriptSettings{Scripts: true, Writes: true}
)

func TestCodeModeIsOffUntilAProviderTurnsItOn(t *testing.T) {
	t.Parallel()

	gate := codemode.Gate{}
	assert.False(t, gate.Offered(codemode.OverrideDefault, false))
	assert.False(t, gate.Offered(codemode.OverrideOn, true), "forcing it needs a development deployment")
	for _, model := range []string{"claude-haiku-5-5", "claude-sonnet-5-5", "gpt-6-luna"} {
		assert.False(t, gate.Decide(codemode.OverrideDefault, model, off).Allowed)
	}
}

func TestGateDecidesOnTheProviderThatAnswered(t *testing.T) {
	t.Parallel()

	gate := codemode.Gate{}
	assert.True(t, gate.Offered(codemode.OverrideDefault, true))

	allowed := gate.Decide(codemode.OverrideDefault, "claude-sonnet-5-5", reads)
	assert.True(t, allowed.Allowed)
	assert.False(t, allowed.Writes, "scripts read only unless the provider lets them write")

	denied := gate.Decide(codemode.OverrideDefault, "claude-haiku-5-5", off)
	assert.False(t, denied.Allowed, "a failover to a provider without scripts runs no code")
	assert.Contains(t, denied.Reason, "claude-haiku-5-5")

	assert.False(t, gate.Decide(codemode.OverrideOff, "claude-sonnet-5-5", batches).Allowed)
	assert.False(t, gate.Offered(codemode.OverrideOff, true))
}

func TestGateLetsScriptsWriteOnlyWhereTheProviderSaysSo(t *testing.T) {
	t.Parallel()

	gate := codemode.Gate{}
	assert.True(t, gate.Decide(codemode.OverrideDefault, "claude-sonnet-5-5", batches).Writes)
	assert.False(t, gate.Decide(codemode.OverrideDefault, "claude-sonnet-5-5", reads).Writes)
}

func TestGateIsForcedOnlyWhereTheBenchRuns(t *testing.T) {
	t.Parallel()

	shut := codemode.Gate{}
	assert.False(t, shut.Decide(codemode.OverrideOn, "any", off).Allowed)
	assert.False(t, shut.Decide(codemode.OverrideWrites, "any", batches).Allowed)

	forced := codemode.Gate{Forced: true}
	assert.True(t, forced.Offered(codemode.OverrideOn, false))
	assert.True(t, forced.Decide(codemode.OverrideOn, "any", off).Allowed)
	assert.False(t, forced.Decide(codemode.OverrideOn, "any", off).Writes)
	assert.True(t, forced.Decide(codemode.OverrideWrites, "any", off).Writes)
	assert.False(t, forced.Offered(codemode.OverrideDefault, false))
	assert.False(t, forced.Decide(codemode.OverrideOff, "any", batches).Allowed)
}
