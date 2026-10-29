package runstepledger

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunstep"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeStepRepo struct {
	claimed []*agentrunstep.AgentRunStep
	settled  []agentruntimeports.SettleAgentRunStepRequest
	recorded []*agentrunstep.AgentRunStep
	rows     []*agentrunstep.AgentRunStep
}

func (f *fakeStepRepo) Claim(
	_ context.Context,
	step *agentrunstep.AgentRunStep,
) (*agentrunstep.AgentRunStep, error) {
	f.claimed = append(f.claimed, step)

	return step, nil
}

func (f *fakeStepRepo) Settle(
	_ context.Context,
	req *agentruntimeports.SettleAgentRunStepRequest,
) error {
	f.settled = append(f.settled, *req)

	return nil
}

func (f *fakeStepRepo) Record(_ context.Context, step *agentrunstep.AgentRunStep) error {
	f.recorded = append(f.recorded, step)

	return nil
}

func (f *fakeStepRepo) List(
	context.Context,
	*agentruntimeports.ListAgentRunStepsRequest,
) ([]*agentrunstep.AgentRunStep, error) {
	return f.rows, nil
}

func (f *fakeStepRepo) Get(
	context.Context,
	*agentruntimeports.GetAgentRunStepRequest,
) (*agentrunstep.AgentRunStep, error) {
	return nil, errors.New("not used")
}

func (f *fakeStepRepo) Prune(
	context.Context,
	agentruntimeports.PruneAgentRunStepsRequest,
) (int, error) {
	return 0, nil
}

func TestClaim_WritesWhereAndAsWhomTheStepRan(t *testing.T) {
	t.Parallel()

	repo := &fakeStepRepo{}
	ledger := New(Params{Logger: zap.NewNop(), Repo: repo})
	version := int64(12)
	definitionID := pulid.MustNew("agdef_")
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	verdict, err := ledger.Claim(t.Context(), tenant, &agentruntimeports.RunStep{
		OwnerKind:         agentruntimeports.RunStepOwnerAssistantTurn,
		OwnerID:           pulid.MustNew("atrn_"),
		Kind:              agentruntimeports.RunStepTool,
		Key:               "step-1",
		ToolName:          "dispatch_assign_move",
		TraceID:           "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:            "00f067aa0ba902b7",
		DefinitionID:      definitionID,
		DefinitionVersion: &version,
		DelegateCallID:    "call_1",
	})
	require.NoError(t, err)
	assert.Equal(t, agentruntimeports.StepFresh, verdict.State)

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

	require.NoError(
		t,
		ledger.Settle(t.Context(), pagination.TenantInfo{}, &agentruntimeports.RunStep{
			OwnerID: pulid.MustNew("ar_"),
			Key:     "step-2",
			Status:  agentruntimeports.RunStepFailed,
			Outcome: agentruntimeports.RunStepOutcome{
				Content: "Tool \"dispatch_assign_move\" is not permitted.",
				Failed:  true,
				Reason:  "lacks update access to shipment_move",
				Verdict: "denied",
			},
		}),
	)

	require.Len(t, repo.settled, 1)
	assert.Equal(t, "lacks update access to shipment_move", repo.settled[0].Outcome["reason"])
	assert.Equal(t, "denied", repo.settled[0].Outcome["verdict"])
}

func TestLoaded_ReadsTheProvenanceBack(t *testing.T) {
	t.Parallel()

	version := int64(4)
	repo := &fakeStepRepo{rows: []*agentrunstep.AgentRunStep{{
		OwnerKind:              string(agentruntimeports.RunStepOwnerAgentRun),
		OwnerID:                pulid.MustNew("ar_"),
		Kind:                   string(agentruntimeports.RunStepTool),
		Status:                 string(agentruntimeports.RunStepCompleted),
		StepKey:                "step-3",
		TraceID:                "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:                 "00f067aa0ba902b7",
		AgentDefinitionID:      pulid.MustNew("agdef_"),
		AgentDefinitionVersion: &version,
		DelegateCallID:         "call_2",
		Outcome:                map[string]any{"verdict": "ran"},
	}}}
	ledger := New(Params{Logger: zap.NewNop(), Repo: repo})

	steps, err := ledger.Loaded(
		t.Context(),
		pagination.TenantInfo{},
		agentruntimeports.RunStepOwner{},
	)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "00f067aa0ba902b7", steps[0].SpanID)
	assert.Equal(t, "call_2", steps[0].DelegateCallID)
	require.NotNil(t, steps[0].DefinitionVersion)
	assert.Equal(t, int64(4), *steps[0].DefinitionVersion)
	assert.Equal(t, "ran", steps[0].Outcome.Verdict)
}

func TestClaim_StoresNoOutcomeForAStepNotYetRun(t *testing.T) {
	t.Parallel()

	repo := &fakeStepRepo{}
	ledger := New(Params{Logger: zap.NewNop(), Repo: repo})

	_, err := ledger.Claim(t.Context(), pagination.TenantInfo{}, &agentruntimeports.RunStep{
		OwnerKind: agentruntimeports.RunStepOwnerAgentRun,
		OwnerID:   pulid.MustNew("ar_"),
		Kind:      agentruntimeports.RunStepTool,
		Key:       "step-claim",
		Outcome:   agentruntimeports.RunStepOutcome{Content: "not yet"},
	})
	require.NoError(t, err)

	require.Len(t, repo.claimed, 1)
	assert.Equal(t, string(agentruntimeports.RunStepStarted), repo.claimed[0].Status)
	require.NotNil(t, repo.claimed[0].Outcome)
	assert.Empty(t, repo.claimed[0].Outcome)
	assert.Nil(t, repo.claimed[0].SettledAtMs)
}

func TestRecord_StoresTheOutcomeAndReadsItBack(t *testing.T) {
	t.Parallel()

	repo := &fakeStepRepo{}
	ledger := New(Params{Logger: zap.NewNop(), Repo: repo})
	owner := pulid.MustNew("atrn_")
	outcome := agentruntimeports.RunStepOutcome{
		Content: "Shipment S-1001 is billed.",
		Data:    map[string]any{"status": "Billed"},
		RanWith: map[string]any{"shipmentId": "shp_1"},
		Verdict: "ran",
	}

	require.NoError(t, ledger.Record(t.Context(), pagination.TenantInfo{}, &agentruntimeports.RunStep{
		OwnerKind: agentruntimeports.RunStepOwnerAssistantTurn,
		OwnerID:   owner,
		Kind:      agentruntimeports.RunStepCompletion,
		Status:    agentruntimeports.RunStepCompleted,
		Key:       "reply-1",
		Outcome:   outcome,
	}))

	require.Len(t, repo.recorded, 1)
	row := repo.recorded[0]
	assert.Equal(t, string(agentruntimeports.RunStepCompleted), row.Status)
	assert.Equal(t, "Shipment S-1001 is billed.", row.Outcome["content"])
	assert.Equal(t, "ran", row.Outcome["verdict"])
	require.NotNil(t, row.SettledAtMs, "a recorded step is already finished")

	repo.rows = repo.recorded
	steps, err := ledger.Loaded(t.Context(), pagination.TenantInfo{}, agentruntimeports.RunStepOwner{
		Kind: agentruntimeports.RunStepOwnerAssistantTurn,
		ID:   owner,
	})
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, outcome, steps[0].Outcome)
}

func TestRecord_StoresAnEmptyOutcomeAsAnObject(t *testing.T) {
	t.Parallel()

	repo := &fakeStepRepo{}
	ledger := New(Params{Logger: zap.NewNop(), Repo: repo})

	require.NoError(t, ledger.Record(t.Context(), pagination.TenantInfo{}, &agentruntimeports.RunStep{
		OwnerKind: agentruntimeports.RunStepOwnerAgentRun,
		OwnerID:   pulid.MustNew("ar_"),
		Kind:      agentruntimeports.RunStepCompletion,
		Status:    agentruntimeports.RunStepCompleted,
		Key:       "reply-empty",
	}))

	require.Len(t, repo.recorded, 1)
	require.NotNil(t, repo.recorded[0].Outcome)
	require.NotNil(t, repo.recorded[0].Arguments)
}

func TestRecord_DropsAnAnswerTooLargeToKeep(t *testing.T) {
	t.Parallel()

	repo := &fakeStepRepo{}
	ledger := New(Params{Logger: zap.NewNop(), Repo: repo})

	require.NoError(t, ledger.Record(t.Context(), pagination.TenantInfo{}, &agentruntimeports.RunStep{
		OwnerKind: agentruntimeports.RunStepOwnerAgentRun,
		OwnerID:   pulid.MustNew("ar_"),
		Kind:      agentruntimeports.RunStepTool,
		Status:    agentruntimeports.RunStepCompleted,
		Key:       "step-large",
		Outcome: agentruntimeports.RunStepOutcome{
			Content: strings.Repeat("x", maxStoredOutcomeChars+1),
			Verdict: "ran",
		},
	}))

	require.Len(t, repo.recorded, 1)
	assert.Empty(t, repo.recorded[0].Outcome["content"])
	assert.Equal(t, "ran", repo.recorded[0].Outcome["verdict"])
}

func TestRecord_ClassifiesAFailedToolCall(t *testing.T) {
	t.Parallel()

	repo := &fakeStepRepo{}
	ledger := New(Params{Logger: zap.NewNop(), Repo: repo})

	require.NoError(t, ledger.Record(t.Context(), pagination.TenantInfo{}, &agentruntimeports.RunStep{
		OwnerKind: agentruntimeports.RunStepOwnerAgentRun,
		OwnerID:   pulid.MustNew("ar_"),
		Kind:      agentruntimeports.RunStepTool,
		Status:    agentruntimeports.RunStepFailed,
		Key:       "step-denied",
		Outcome: agentruntimeports.RunStepOutcome{
			Content: "Tool \"dispatch_assign_move\" is not permitted.",
			Failed:  true,
			Reason:  "lacks update access to shipment_move",
			Verdict: "denied",
		},
	}))

	require.Len(t, repo.recorded, 1)
	assert.NotEmpty(t, repo.recorded[0].Outcome["cause"])
}

func TestSettle_StoresTheOutcomeAndReadsItBack(t *testing.T) {
	t.Parallel()

	repo := &fakeStepRepo{}
	ledger := New(Params{Logger: zap.NewNop(), Repo: repo})
	owner := pulid.MustNew("ar_")
	outcome := agentruntimeports.RunStepOutcome{
		Content: "Move assigned to driver D-12.",
		Data:    map[string]any{"driver": "D-12"},
		Verdict: "ran",
	}

	require.NoError(t, ledger.Settle(t.Context(), pagination.TenantInfo{}, &agentruntimeports.RunStep{
		OwnerKind: agentruntimeports.RunStepOwnerAgentRun,
		OwnerID:   owner,
		Kind:      agentruntimeports.RunStepTool,
		Status:    agentruntimeports.RunStepCompleted,
		Key:       "step-4",
		Outcome:   outcome,
	}))

	require.Len(t, repo.settled, 1)
	repo.rows = []*agentrunstep.AgentRunStep{{
		OwnerKind: string(agentruntimeports.RunStepOwnerAgentRun),
		OwnerID:   owner,
		Kind:      string(agentruntimeports.RunStepTool),
		Status:    repo.settled[0].Status,
		StepKey:   repo.settled[0].StepKey,
		Outcome:   repo.settled[0].Outcome,
	}}

	steps, err := ledger.Loaded(t.Context(), pagination.TenantInfo{}, agentruntimeports.RunStepOwner{
		Kind: agentruntimeports.RunStepOwnerAgentRun,
		ID:   owner,
	})
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, agentruntimeports.RunStepCompleted, steps[0].Status)
	assert.Equal(t, outcome, steps[0].Outcome)
}
