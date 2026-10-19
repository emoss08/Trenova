package runstepledger

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeStepRepo struct {
	claimed []*agent.AgentRunStep
	settled []agentports.SettleAgentRunStepRequest
	rows    []*agent.AgentRunStep
}

func (f *fakeStepRepo) Claim(
	_ context.Context,
	step *agent.AgentRunStep,
) (*agent.AgentRunStep, error) {
	f.claimed = append(f.claimed, step)

	return step, nil
}

func (f *fakeStepRepo) Settle(_ context.Context, req agentports.SettleAgentRunStepRequest) error {
	f.settled = append(f.settled, req)

	return nil
}

func (f *fakeStepRepo) Record(context.Context, *agent.AgentRunStep) error { return nil }

func (f *fakeStepRepo) List(
	context.Context,
	agentports.ListAgentRunStepsRequest,
) ([]*agent.AgentRunStep, error) {
	return f.rows, nil
}

func (f *fakeStepRepo) Prune(context.Context, agentports.PruneAgentRunStepsRequest) (int, error) {
	return 0, nil
}

func TestClaim_WritesWhereAndAsWhomTheStepRan(t *testing.T) {
	t.Parallel()

	repo := &fakeStepRepo{}
	ledger := New(Params{Logger: zap.NewNop(), Repo: repo})
	version := int64(12)
	definitionID := pulid.MustNew("agdef_")
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	verdict, err := ledger.Claim(t.Context(), tenant, agentports.RunStep{
		OwnerKind:         agentports.RunStepOwnerAssistantTurn,
		OwnerID:           pulid.MustNew("atrn_"),
		Kind:              agentports.RunStepTool,
		Key:               "step-1",
		ToolName:          "dispatch_assign_move",
		TraceID:           "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:            "00f067aa0ba902b7",
		DefinitionID:      definitionID,
		DefinitionVersion: &version,
		DelegateCallID:    "call_1",
	})
	require.NoError(t, err)
	assert.Equal(t, agentports.StepFresh, verdict.State)

	require.Len(t, repo.claimed, 1)
	row := repo.claimed[0]
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", row.TraceID)
	assert.Equal(t, "00f067aa0ba902b7", row.SpanID)
	assert.Equal(t, definitionID, row.AgentDefinitionID)
	require.NotNil(t, row.AgentDefinitionVersion)
	assert.Equal(t, int64(12), *row.AgentDefinitionVersion)
	assert.Equal(t, "call_1", row.DelegateCallID)
	version = 13
	assert.Equal(t, int64(12), *row.AgentDefinitionVersion, "the row keeps its own copy")
}

func TestSettle_KeepsTheVerdictAndWhyACallWasRefused(t *testing.T) {
	t.Parallel()

	repo := &fakeStepRepo{}
	ledger := New(Params{Logger: zap.NewNop(), Repo: repo})

	require.NoError(t, ledger.Settle(t.Context(), pagination.TenantInfo{}, agentports.RunStep{
		OwnerID: pulid.MustNew("ar_"),
		Key:     "step-2",
		Status:  agentports.RunStepFailed,
		Outcome: agentports.RunStepOutcome{
			Content: "Tool \"dispatch_assign_move\" is not permitted.",
			Failed:  true,
			Reason:  "lacks update access to shipment_move",
			Verdict: "denied",
		},
	}))

	require.Len(t, repo.settled, 1)
	assert.Equal(t, "lacks update access to shipment_move", repo.settled[0].Outcome["reason"])
	assert.Equal(t, "denied", repo.settled[0].Outcome["verdict"])
}

func TestLoaded_ReadsTheProvenanceBack(t *testing.T) {
	t.Parallel()

	version := int64(4)
	repo := &fakeStepRepo{rows: []*agent.AgentRunStep{{
		OwnerKind:              string(agentports.RunStepOwnerAgentRun),
		OwnerID:                pulid.MustNew("ar_"),
		Kind:                   string(agentports.RunStepTool),
		Status:                 string(agentports.RunStepCompleted),
		StepKey:                "step-3",
		TraceID:                "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:                 "00f067aa0ba902b7",
		AgentDefinitionID:      pulid.MustNew("agdef_"),
		AgentDefinitionVersion: &version,
		DelegateCallID:         "call_2",
		Outcome:                map[string]any{"verdict": "ran"},
	}}}
	ledger := New(Params{Logger: zap.NewNop(), Repo: repo})

	steps, err := ledger.Loaded(t.Context(), pagination.TenantInfo{}, agentports.RunStepOwner{})
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "00f067aa0ba902b7", steps[0].SpanID)
	assert.Equal(t, "call_2", steps[0].DelegateCallID)
	require.NotNil(t, steps[0].DefinitionVersion)
	assert.Equal(t, int64(4), *steps[0].DefinitionVersion)
	assert.Equal(t, "ran", steps[0].Outcome.Verdict)
}
