package codemode_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeModeShipsOffForEveryModel(t *testing.T) {
	t.Parallel()

	require.NoError(t, codemode.ShippedError())
	ledger := codemode.Shipped()
	assert.False(t, ledger.Any(), "code mode is off for every model until the bench earns one")

	gate := codemode.Gate{Ledger: ledger}
	assert.False(t, gate.Offered(codemode.OverrideDefault))
	assert.False(t, gate.Offered(codemode.OverrideOn), "forcing it needs a development deployment")
	for _, model := range []string{"claude-haiku-5-5", "claude-sonnet-5-5", "gpt-6-luna"} {
		assert.False(t, gate.Decide(codemode.OverrideDefault, aiprovider.KindAnthropicMessages, model).Allowed)
	}
}

func TestLedgerRefusesAModelWithoutTheRunThatEarnedIt(t *testing.T) {
	t.Parallel()

	_, err := codemode.ParseLedger([]byte(
		"models:\n  - provider: AnthropicMessages\n    model: claude-sonnet-5-5\n"))
	require.ErrorIs(t, err, codemode.ErrUnearnedModel)

	ledger, err := codemode.ParseLedger([]byte("models:\n  - provider: AnthropicMessages\n" +
		"    model: claude-sonnet-5-5\n    run: .deskbench/runs/1/summary.md\n    decided: 2026-11-02\n"))
	require.NoError(t, err)
	assert.True(t, ledger.Enabled(aiprovider.KindAnthropicMessages, "Claude-Sonnet-5-5"))
	assert.False(t, ledger.Enabled(aiprovider.KindOpenAIResponses, "claude-sonnet-5-5"))
	assert.False(t, ledger.Enabled(aiprovider.KindAnthropicMessages, "claude-haiku-5-5"))
}

func TestGateDecidesOnTheModelThatAsked(t *testing.T) {
	t.Parallel()

	ledger, err := codemode.ParseLedger([]byte("models:\n  - provider: AnthropicMessages\n" +
		"    model: claude-sonnet-5-5\n    run: r\n    decided: d\n"))
	require.NoError(t, err)
	gate := codemode.Gate{Ledger: ledger}

	assert.True(t, gate.Offered(codemode.OverrideDefault))
	assert.True(t, gate.Decide(codemode.OverrideDefault, aiprovider.KindAnthropicMessages,
		"claude-sonnet-5-5").Allowed)
	denied := gate.Decide(codemode.OverrideDefault, aiprovider.KindAnthropicMessages, "claude-haiku-5-5")
	assert.False(t, denied.Allowed, "a failover to a model the bench never enabled runs no code")
	assert.Contains(t, denied.Reason, "claude-haiku-5-5")
	assert.False(t, gate.Decide(codemode.OverrideOff, aiprovider.KindAnthropicMessages,
		"claude-sonnet-5-5").Allowed)
	assert.False(t, gate.Offered(codemode.OverrideOff))
}

func TestGateIsForcedOnlyWhereTheBenchRuns(t *testing.T) {
	t.Parallel()

	shut := codemode.Gate{}
	assert.False(t, shut.Decide(codemode.OverrideOn, aiprovider.KindAnthropicMessages, "any").Allowed)

	forced := codemode.Gate{Forced: true}
	assert.True(t, forced.Offered(codemode.OverrideOn))
	assert.True(t, forced.Decide(codemode.OverrideOn, aiprovider.KindAnthropicMessages, "any").Allowed)
	assert.False(t, forced.Offered(codemode.OverrideDefault))
}

func TestLedgerRefusesWritesWithoutTheRunThatEarnedThem(t *testing.T) {
	t.Parallel()

	base := "models:\n  - provider: AnthropicMessages\n    model: claude-sonnet-5-5\n" +
		"    run: r\n    decided: d\n"
	_, err := codemode.ParseLedger([]byte(base + "    writes: true\n"))
	require.ErrorIs(t, err, codemode.ErrUnearnedWrites)

	reads, err := codemode.ParseLedger([]byte(base))
	require.NoError(t, err)
	readOnly := codemode.Gate{Ledger: reads}.Decide(
		codemode.OverrideDefault, aiprovider.KindAnthropicMessages, "claude-sonnet-5-5")
	assert.True(t, readOnly.Allowed)
	assert.False(t, readOnly.Writes, "a model earns reads and writes separately")

	writes, err := codemode.ParseLedger([]byte(base + "    writes: true\n    writesRun: w\n"))
	require.NoError(t, err)
	assert.True(t, codemode.Gate{Ledger: writes}.Decide(
		codemode.OverrideDefault, aiprovider.KindAnthropicMessages, "claude-sonnet-5-5").Writes)
	assert.False(t, codemode.Gate{Ledger: writes}.Decide(
		codemode.OverrideDefault, aiprovider.KindAnthropicMessages, "claude-haiku-5-5").Writes)
}

func TestShippedLedgerLetsNoModelWrite(t *testing.T) {
	t.Parallel()

	for _, entry := range codemode.Shipped().Models {
		assert.False(t, entry.Writes, "%s: no model's scripts write until the bench earns it", entry.Model)
	}
	assert.True(t, codemode.Gate{Forced: true}.Decide(
		codemode.OverrideOn, aiprovider.KindAnthropicMessages, "any").Writes,
		"the bench's forced run exercises writes")
}
