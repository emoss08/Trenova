package tooldispatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordLoopRefusals_WritesEachOnceAsAFailedToolStep(t *testing.T) {
	t.Parallel()

	ledger := runtimetest.NewKeyedLedger()
	definition := pulid.MustNew("agdef_")
	req := &RecordLoopRefusalsRequest{
		Ledger: ledger,
		Tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Owner: agentruntimeports.RunStepOwner{
			Kind: agentruntimeports.RunStepOwnerAssistantTurn,
			ID:   pulid.MustNew("atrn_"),
		},
		DefinitionID: definition,
		Attempt:      1,
		Refusals: []agentruntimeports.LoopRefusal{
			{
				CallID:   "call_1",
				ToolName: "worker_list",
				Verdict:  aitrace.OutcomeDenied,
				Reason:   "the agent does not hold the tool",
				Content:  `"worker_list" is not enabled for this agent`,
			},
			{CallID: "", ToolName: "nameless", Verdict: aitrace.OutcomeInvalid},
		},
	}

	assert.Zero(t, RecordLoopRefusals(t.Context(), req))
	require.Len(t, ledger.Steps, 1, "a refusal with no call id has nothing to key it by")

	step := ledger.Steps[loopRefusalKeyPrefix+"call_1"]
	assert.Equal(t, agentruntimeports.RunStepTool, step.Kind)
	assert.Equal(t, agentruntimeports.RunStepFailed, step.Status)
	assert.Equal(t, definition, step.DefinitionID, "the scorecard counts it toward its agent")
	assert.Equal(t, aitrace.OutcomeDenied, step.Outcome.Verdict)
	assert.Equal(t, "the agent does not hold the tool", step.Outcome.Reason)

	req.Attempt = 2
	req.Refusals[0].Reason = "written again"
	assert.Zero(t, RecordLoopRefusals(t.Context(), req))
	assert.Equal(t, "the agent does not hold the tool",
		ledger.Steps[loopRefusalKeyPrefix+"call_1"].Outcome.Reason,
		"a retried finish does not write the refusal twice")
}
