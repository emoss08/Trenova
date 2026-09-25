package proposalrecorder

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	provenanceTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	provenanceSpanID  = "00f067aa0ba902b7"
)

func TestRecord_KeepsTheProposalTheCallWasDecidedAs(t *testing.T) {
	t.Parallel()

	minted := pulid.MustNew("ap_")
	proposal := recordOne(t, serviceports.PendingAction{
		ToolName:   "assign_move",
		Arguments:  map[string]any{},
		Rationale:  "cover it",
		Tier:       agent.TierPropose,
		ProposalID: minted,
		TraceID:    provenanceTraceID,
		SpanID:     provenanceSpanID,
		StepKey:    "step-key-1",
		TierSource: agent.TierSourcePersonSetting,
	})

	assert.Equal(t, minted, proposal.ID, "the step, the trace and the proposal name one record")
	assert.Equal(t, provenanceTraceID, proposal.TraceID)
	assert.Equal(t, provenanceSpanID, proposal.SpanID)
	assert.Equal(t, "step-key-1", proposal.StepKey)
	assert.True(t, proposal.ExecutedByUserID.IsNil(), "nothing ran")
	assert.Nil(t, proposal.ExecutedTargetVersion)
}

func TestRecord_MintsAsBeforeForAnActionRecordedWithoutAnID(t *testing.T) {
	t.Parallel()

	proposal := recordOne(t, serviceports.PendingAction{
		ToolName:  "assign_move",
		Arguments: map[string]any{},
		Rationale: "cover it",
		Tier:      agent.TierPropose,
	})

	assert.True(t, proposal.ID.IsNil(),
		"left to the insert to mint, as every action from before the id was kept is")
	assert.Empty(t, proposal.TraceID)
}

func TestRecord_KeepsWhoAnAutomaticWriteRanAsAndWhen(t *testing.T) {
	t.Parallel()

	version := int64(9)
	proposal := recordOne(t, serviceports.PendingAction{
		ToolName:        "assign_move",
		Arguments:       map[string]any{},
		Rationale:       "cover it",
		Tier:            agent.TierAutoExecute,
		Executed:        true,
		ExecutedAt:      1_790_000_000,
		ExecutedVersion: &version,
	})

	require.NotNil(t, proposal.ExecutedAt)
	assert.Equal(t, int64(1_790_000_000), *proposal.ExecutedAt,
		"when the write ran, not when the turn was filed")
	assert.True(t, proposal.ExecutedByUserID.IsNotNil(), "it ran as the person in the conversation")
	require.NotNil(t, proposal.ExecutedTargetVersion)
	assert.Equal(t, int64(9), *proposal.ExecutedTargetVersion)
	version = 10
	assert.Equal(t, int64(9), *proposal.ExecutedTargetVersion)
}

func TestRecord_AnUnattendedWriteRanAsNoPerson(t *testing.T) {
	t.Parallel()

	store := &capturingStore{}
	_, err := NewWithStores(nil, nil, store).Record(t.Context(), &RecordRequest{
		Actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeAgent,
			PrincipalID:    serviceports.AgentPrincipalID,
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		Run: &agent.AgentRun{ID: pulid.MustNew("arun_")},
		Actions: []serviceports.PendingAction{{
			ToolName:  "assign_move",
			Arguments: map[string]any{},
			Rationale: "cover it",
			Tier:      agent.TierAutoExecute,
			Executed:  true,
		}},
		Evidence: messageEvidence,
	})
	require.NoError(t, err)
	require.Len(t, store.created, 1)
	assert.True(t, store.created[0].ExecutedByUserID.IsNil())
	require.NotNil(t, store.created[0].ExecutedAt, "a write filed without its time is stamped now")
}

func TestRecord_OpensADelegatesRunUnderItsParentAndTask(t *testing.T) {
	t.Parallel()

	runs := &openedRuns{}
	turnID := pulid.MustNew("atrn_")
	_, err := NewWithStores(nil, runs, &capturingStore{}).Record(t.Context(), &RecordRequest{
		Actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeUser,
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		Open: &OpenRunRequest{
			AgentType:        agent.TypeAssistantChat,
			SubjectType:      agent.SubjectAssistantThread,
			SubjectID:        pulid.MustNew("athr_"),
			Trigger:          agent.RunTriggerChat,
			PromptVersion:    "assistant-chat/v2",
			InputContextHash: "hash",
			TraceID:          provenanceTraceID,
			TurnID:           turnID,
			ParentOwnerKind:  agent.RunOwnerAssistantTurn,
			ParentOwnerID:    turnID,
			DelegateCallID:   "call_task_1",
		},
		Actions: []serviceports.PendingAction{
			{ToolName: "assign_move", Rationale: "cover it", Tier: agent.TierPropose},
		},
		Evidence: messageEvidence,
	})
	require.NoError(t, err)

	require.Len(t, runs.opened, 1)
	run := runs.opened[0]
	assert.Equal(t, provenanceTraceID, run.TraceID)
	assert.Equal(t, turnID, run.TurnID)
	assert.Equal(t, agent.RunOwnerAssistantTurn, run.ParentOwnerKind)
	assert.Equal(t, turnID, run.ParentOwnerID)
	assert.Equal(t, "call_task_1", run.DelegateCallID)
}
