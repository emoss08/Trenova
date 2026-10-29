package tooldispatch

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/loopguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A delegate's step keys are its own, and the turn's own keys are exactly what
// they were before delegation existed, so an in-flight turn's ledger still
// answers for its steps.
func TestStepKey_ScopeSeparatesADelegatesStepsAndLeavesTheOwnersAlone(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("atrn_")
	params := toolcall.StepKeyParams{
		OwnerID:  owner,
		ToolName: "report_create",
		Args:     map[string]any{"name": "On-time"},
	}
	own := toolcall.StepKey(params)

	base, ok := loopguard.CallKey(&llmports.ToolCall{Name: params.ToolName, Arguments: params.Args})
	require.True(t, ok)
	legacy := sha256.Sum256([]byte(owner.String() + "\x00" + base + "\x00" + "0"))
	assert.Equal(t, hex.EncodeToString(legacy[:]), own)

	params.Scope = "delegate-a"
	scoped := toolcall.StepKey(params)
	params.Scope = "delegate-b"
	other := toolcall.StepKey(params)
	assert.NotEqual(t, own, scoped)
	assert.NotEqual(t, scoped, other)
}
