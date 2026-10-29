package assistantservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newProposalService(
	runs *assistanttest.StubRunRepo,
	proposals *assistanttest.StubProposalRepo,
	conversations *assistanttest.StubConversationRepo,
) *Service {
	return newProposalServiceBehind(runs, proposals, conversations, assistanttest.ShadowSwitches{})
}

func newProposalServiceBehind(
	runs *assistanttest.StubRunRepo,
	proposals *assistanttest.StubProposalRepo,
	conversations *assistanttest.StubConversationRepo,
	switches assistanttest.ShadowSwitches,
) *Service {
	return &Service{
		logger:        zap.NewNop(),
		recorder:      proposalrecorder.NewWithStores(zap.NewNop(), runs, proposals),
		proposals:     proposals,
		conversations: conversations,
		shadow: agentshadow.New(agentshadow.Params{
			Control:     assistanttest.StubControlRepo{Paused: switches.OrganizationPaused},
			Runs:        runs,
			Definitions: assistanttest.StubDefinitionRepo{Definition: switches.Definition},
		}),
	}
}

func proposalTestParams(
	actions []agentruntimeports.PendingAction,
	saved []conversation.Message,
) persistProposalsParams {
	return persistProposalsParams{
		Definition: &agentdefinition.Definition{
			ID:              pulid.MustNew("agd_"),
			Name:            "Dispatch helper",
			Template:        agenttemplate.TemplateDispatchAssistant,
			AutonomyCeiling: agent.TierPropose,
			Version:         3,
		},
		Thread:  &conversation.Thread{ID: pulid.MustNew("thr_")},
		Actor:   assistanttest.Actor(),
		Saved:   saved,
		Actions: actions,
		Model:   "test-model",
		Input:   "Reassign move mv_1",
	}
}

func TestPersistProposals_SavesNothingWhenTheTurnProposedNothing(t *testing.T) {
	t.Parallel()

	runs := &assistanttest.StubRunRepo{}
	proposals := &assistanttest.StubProposalRepo{}
	svc := newProposalService(runs, proposals, &assistanttest.StubConversationRepo{})

	saved, err := svc.persistProposals(t.Context(), proposalTestParams(nil, nil))
	require.NoError(t, err)

	assert.Nil(t, saved)
	assert.Empty(t, runs.Created, "a turn that proposed nothing must not open an agent run")
	assert.Empty(t, proposals.Created)
}

// A turn that failed or was stopped after a write ran used to record nothing:
// the write happened, but left no audit row, counted against no cap and
// earned no trust. It is recorded now, on a run marked as failed.
func TestPersistProposals_RecordsTheWritesOfATurnThatFailed(t *testing.T) {
	t.Parallel()

	runs := &assistanttest.StubRunRepo{}
	proposals := &assistanttest.StubProposalRepo{}
	svc := newProposalService(runs, proposals, &assistanttest.StubConversationRepo{})

	params := proposalTestParams([]agentruntimeports.PendingAction{
		{
			ToolName:  "hold_shipment",
			Rationale: "Consignee closed",
			Tier:      agent.TierAutoExecute,
			Executed:  true,
		},
	}, nil)
	params.Failed = true

	_, err := svc.persistProposals(t.Context(), params)
	require.NoError(t, err)

	require.Len(t, runs.Created, 1)
	assert.Equal(t, agent.RunStatusFailed, runs.Created[0].Status)
	assert.Len(t, proposals.Created, 1)
}

func TestPersistProposals_OpensOneChatRunForTheTurn(t *testing.T) {
	t.Parallel()

	runs := &assistanttest.StubRunRepo{}
	proposals := &assistanttest.StubProposalRepo{}
	svc := newProposalService(runs, proposals, &assistanttest.StubConversationRepo{})

	params := proposalTestParams([]agentruntimeports.PendingAction{
		{ToolName: "reassign_move", Rationale: "Driver is out of hours", Tier: agent.TierPropose},
		{ToolName: "hold_shipment", Rationale: "Consignee closed", Tier: agent.TierPropose},
	}, nil)

	_, err := svc.persistProposals(t.Context(), params)
	require.NoError(t, err)

	require.Len(t, runs.Created, 1, "both proposals belong to the one turn that raised them")
	run := runs.Created[0]
	assert.Equal(t, agent.TypeAssistantChat, run.AgentType)
	assert.Equal(t, agent.SubjectAssistantThread, run.SubjectType)
	assert.Equal(t, params.Thread.ID, run.SubjectID)
	assert.Equal(t, agent.RunTriggerChat, run.Trigger)
	assert.Equal(t, params.Definition.ID, run.AgentDefinitionID)
	assert.Equal(t, "test-model", run.ModelIdentifier)
	assert.NotEmpty(t, run.InputContextHash)

	require.Len(t, proposals.Created, 2)
	for _, proposal := range proposals.Created {
		assert.Equal(t, run.ID, proposal.RunID)
		assert.Equal(t, agent.ProposalStatusPending, proposal.Status)
	}
}

// A persisted proposal has to be pending, because the whole point of the record
// is that a person has not decided yet.
func TestPersistProposals_RecordsThePendingStatusAndTier(t *testing.T) {
	t.Parallel()

	proposals := &assistanttest.StubProposalRepo{}
	svc := newProposalService(
		&assistanttest.StubRunRepo{},
		proposals,
		&assistanttest.StubConversationRepo{},
	)

	_, err := svc.persistProposals(
		t.Context(),
		proposalTestParams([]agentruntimeports.PendingAction{
			{
				ToolName:  "reassign_move",
				Arguments: map[string]any{"moveId": "mv_1"},
				Rationale: "Driver is out of hours",
				Tier:      agent.TierActWithApproval,
			},
		}, nil),
	)
	require.NoError(t, err)

	require.Len(t, proposals.Created, 1)
	saved := proposals.Created[0]
	assert.Equal(t, agent.ProposalStatusPending, saved.Status)
	assert.Equal(t, agent.TierActWithApproval, saved.AutonomyTier)
	assert.Equal(t, "mv_1", saved.ToolParams["moveId"])
	assert.Equal(t, "Driver is out of hours", saved.Rationale)
}

func TestPersistProposals_TiesEachProposalToTheMessageThatAskedForIt(t *testing.T) {
	t.Parallel()

	firstTurn := conversation.Message{
		ID:        pulid.MustNew("amsg_"),
		Role:      conversation.RoleAssistant,
		ToolCalls: []conversation.ToolCallRecord{{ID: "call_1", Name: "reassign_move"}},
	}
	secondTurn := conversation.Message{
		ID:        pulid.MustNew("amsg_"),
		Role:      conversation.RoleAssistant,
		ToolCalls: []conversation.ToolCallRecord{{ID: "call_2", Name: "hold_shipment"}},
	}

	proposals := &assistanttest.StubProposalRepo{}
	svc := newProposalService(
		&assistanttest.StubRunRepo{},
		proposals,
		&assistanttest.StubConversationRepo{},
	)

	_, err := svc.persistProposals(t.Context(), proposalTestParams(
		[]agentruntimeports.PendingAction{
			{ToolName: "hold_shipment", Rationale: "Consignee closed", ToolCallID: "call_2"},
			{ToolName: "reassign_move", Rationale: "Out of hours", ToolCallID: "call_1"},
		},
		[]conversation.Message{
			{ID: pulid.MustNew("amsg_"), Role: conversation.RoleUser},
			firstTurn,
			secondTurn,
		},
	))
	require.NoError(t, err)

	require.Len(t, proposals.Created, 2)
	assert.Equal(t, secondTurn.ID, proposals.Created[0].SourceMessageID)
	assert.Equal(t, firstTurn.ID, proposals.Created[1].SourceMessageID)
}

// The column is not-null, so a tool taking no arguments must still store an
// object rather than a JSON null.
func TestPersistProposals_StoresEmptyArgumentsAsAnObject(t *testing.T) {
	t.Parallel()

	proposals := &assistanttest.StubProposalRepo{}
	svc := newProposalService(
		&assistanttest.StubRunRepo{},
		proposals,
		&assistanttest.StubConversationRepo{},
	)

	_, err := svc.persistProposals(
		t.Context(),
		proposalTestParams([]agentruntimeports.PendingAction{
			{ToolName: "close_period", Rationale: "Month end"},
		}, nil),
	)
	require.NoError(t, err)

	require.Len(t, proposals.Created, 1)
	assert.NotNil(t, proposals.Created[0].ToolParams)
	assert.Empty(t, proposals.Created[0].ToolParams)
}

// An unset tier must never mean "no restriction". It falls back to the tier that
// requires a person, which is the safe direction to be wrong in.
func TestPersistProposals_FallsBackToTheMostRestrictiveTier(t *testing.T) {
	t.Parallel()

	proposals := &assistanttest.StubProposalRepo{}
	svc := newProposalService(
		&assistanttest.StubRunRepo{},
		proposals,
		&assistanttest.StubConversationRepo{},
	)

	_, err := svc.persistProposals(
		t.Context(),
		proposalTestParams([]agentruntimeports.PendingAction{
			{ToolName: "reassign_move", Rationale: "Out of hours"},
		}, nil),
	)
	require.NoError(t, err)

	require.Len(t, proposals.Created, 1)
	assert.Equal(t, agent.TierPropose, proposals.Created[0].AutonomyTier)
}

// An approver who wants more than the rationale needs a way back to the exchange
// that produced the proposal.
func TestPersistProposals_CitesTheConversationAsEvidence(t *testing.T) {
	t.Parallel()

	sourceTurn := conversation.Message{
		ID:        pulid.MustNew("amsg_"),
		Role:      conversation.RoleAssistant,
		ToolCalls: []conversation.ToolCallRecord{{ID: "call_1", Name: "reassign_move"}},
	}

	proposals := &assistanttest.StubProposalRepo{}
	svc := newProposalService(
		&assistanttest.StubRunRepo{},
		proposals,
		&assistanttest.StubConversationRepo{},
	)

	params := proposalTestParams(
		[]agentruntimeports.PendingAction{{
			ToolName:   "reassign_move",
			Rationale:  "Out of hours",
			Tier:       agent.TierPropose,
			ToolCallID: "call_1",
		}},
		[]conversation.Message{sourceTurn},
	)

	_, err := svc.persistProposals(t.Context(), params)
	require.NoError(t, err)

	require.Len(t, proposals.Created, 1)
	evidence := proposals.Created[0].Evidence
	require.Len(t, evidence, 2)
	assert.Equal(t, evidenceTypeThread, evidence[0].Type)
	assert.Equal(t, params.Thread.ID.String(), evidence[0].ID)
	assert.Equal(t, evidenceTypeMessage, evidence[1].Type)
	assert.Equal(t, sourceTurn.ID.String(), evidence[1].ID)
}

// Citing a message id that is not there would be worse than citing only the
// thread, so an unmatched tool call cites what is actually known.
func TestPersistProposals_CitesOnlyTheThreadWhenTheMessageIsUnknown(t *testing.T) {
	t.Parallel()

	proposals := &assistanttest.StubProposalRepo{}
	svc := newProposalService(
		&assistanttest.StubRunRepo{},
		proposals,
		&assistanttest.StubConversationRepo{},
	)

	_, err := svc.persistProposals(
		t.Context(),
		proposalTestParams([]agentruntimeports.PendingAction{
			{ToolName: "reassign_move", Rationale: "Out of hours", ToolCallID: "call_missing"},
		}, nil),
	)
	require.NoError(t, err)

	require.Len(t, proposals.Created, 1)
	require.Len(t, proposals.Created[0].Evidence, 1)
	assert.Equal(t, evidenceTypeThread, proposals.Created[0].Evidence[0].Type)
	assert.True(t, proposals.Created[0].SourceMessageID.IsNil())
}

func TestPersistProposals_FailsWhenTheRunCannotBeOpened(t *testing.T) {
	t.Parallel()

	proposals := &assistanttest.StubProposalRepo{}
	svc := newProposalService(
		&assistanttest.StubRunRepo{Err: errors.New("database is down")},
		proposals,
		&assistanttest.StubConversationRepo{},
	)

	_, err := svc.persistProposals(
		t.Context(),
		proposalTestParams([]agentruntimeports.PendingAction{
			{ToolName: "reassign_move", Rationale: "Out of hours"},
		}, nil),
	)

	require.Error(t, err)
	assert.Empty(t, proposals.Created, "no proposal may exist without the run it belongs to")
}

// A proposal made while the organization's pause is on arrives on hold, naming
// the pause, so the card never offers an approval the server will refuse.
func TestPersistProposals_HoldsAProposalBehindTheOrganizationPause(t *testing.T) {
	t.Parallel()

	svc := newProposalServiceBehind(
		&assistanttest.StubRunRepo{},
		&assistanttest.StubProposalRepo{},
		&assistanttest.StubConversationRepo{},
		assistanttest.ShadowSwitches{OrganizationPaused: true},
	)

	saved, err := svc.persistProposals(
		t.Context(),
		proposalTestParams([]agentruntimeports.PendingAction{
			{
				ToolName:  "reassign_move",
				Arguments: map[string]any{"moveId": "mv_1"},
				Rationale: "Driver is out of hours",
			},
		}, nil),
	)
	require.NoError(t, err)

	require.Len(t, saved, 1)
	require.NotNil(t, saved[0].Hold)
	assert.Equal(t, string(agentshadow.CauseOrganization), saved[0].Hold.Reason)
	assert.Empty(t, saved[0].Hold.AgentName)
}

// The agent's own switch names the agent, because that is where to go to
// change it.
func TestPersistProposals_HoldsAProposalBehindTheAgentSwitch(t *testing.T) {
	t.Parallel()

	params := proposalTestParams([]agentruntimeports.PendingAction{
		{
			ToolName:  "reassign_move",
			Arguments: map[string]any{"moveId": "mv_1"},
			Rationale: "Driver is out of hours",
		},
	}, nil)
	params.Definition.ShadowMode = true
	svc := newProposalServiceBehind(
		&assistanttest.StubRunRepo{},
		&assistanttest.StubProposalRepo{},
		&assistanttest.StubConversationRepo{},
		assistanttest.ShadowSwitches{},
	)

	saved, err := svc.persistProposals(t.Context(), params)
	require.NoError(t, err)

	require.Len(t, saved, 1)
	require.NotNil(t, saved[0].Hold)
	assert.Equal(t, string(agentshadow.CauseDefinition), saved[0].Hold.Reason)
	assert.Equal(t, "Dispatch helper", saved[0].Hold.AgentName)
}

func TestPersistProposals_LeavesALiveProposalUnheld(t *testing.T) {
	t.Parallel()

	svc := newProposalService(
		&assistanttest.StubRunRepo{},
		&assistanttest.StubProposalRepo{},
		&assistanttest.StubConversationRepo{},
	)

	saved, err := svc.persistProposals(
		t.Context(),
		proposalTestParams([]agentruntimeports.PendingAction{
			{
				ToolName:  "reassign_move",
				Arguments: map[string]any{"moveId": "mv_1"},
				Rationale: "Driver is out of hours",
			},
		}, nil),
	)
	require.NoError(t, err)

	require.Len(t, saved, 1)
	assert.Nil(t, saved[0].Hold)
}
