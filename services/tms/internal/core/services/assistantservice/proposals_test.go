package assistantservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubRunRepo struct {
	created []*agent.AgentRun
	err     error
}

func (r *stubRunRepo) Create(
	_ context.Context,
	entity *agent.AgentRun,
) (*agent.AgentRun, error) {
	if r.err != nil {
		return nil, r.err
	}

	entity.ID = pulid.MustNew("ar_")
	r.created = append(r.created, entity)

	return entity, nil
}

type stubProposalRepo struct {
	created  []*agent.AgentProposal
	byThread []*agent.AgentProposal
	err      error
	listErr  error
	lastList repositories.ListAgentProposalsByThreadRequest
}

func (r *stubProposalRepo) Create(
	_ context.Context,
	entity *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	if r.err != nil {
		return nil, r.err
	}

	entity.ID = pulid.MustNew("ap_")
	r.created = append(r.created, entity)

	return entity, nil
}

func (r *stubProposalRepo) ListByThread(
	_ context.Context,
	req repositories.ListAgentProposalsByThreadRequest,
) ([]*agent.AgentProposal, error) {
	r.lastList = req
	if r.listErr != nil {
		return nil, r.listErr
	}

	return r.byThread, nil
}

// The conversation repository is embedded rather than implemented: this suite
// only reads a thread, and an unexpected call panics on the nil interface, which
// is exactly the failure a test wants.
type stubConversationRepo struct {
	repositories.ConversationRepository

	thread *conversation.Thread
	getErr error
}

func (r *stubConversationRepo) GetThread(
	_ context.Context,
	_ repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}

	return r.thread, nil
}

func newProposalService(
	runs *stubRunRepo,
	proposals *stubProposalRepo,
	conversations *stubConversationRepo,
) *Service {
	return &Service{
		logger:        zap.NewNop(),
		recorder:      proposalrecorder.NewWithStores(zap.NewNop(), runs, proposals),
		proposals:     proposals,
		conversations: conversations,
	}
}

func proposalTestParams(actions []serviceports.PendingAction, saved []conversation.Message) persistProposalsParams {
	return persistProposalsParams{
		Definition: &agentdefinition.Definition{
			ID:              pulid.MustNew("agd_"),
			Name:            "Dispatch helper",
			Template:        agentdefinition.TemplateDispatchAssistant,
			AutonomyCeiling: agent.TierPropose,
			Version:         3,
		},
		Thread:  &conversation.Thread{ID: pulid.MustNew("thr_")},
		Actor:   testActor(),
		Saved:   saved,
		Actions: actions,
		Model:   "test-model",
		Input:   "Reassign move mv_1",
	}
}

func TestPersistProposals_SavesNothingWhenTheTurnProposedNothing(t *testing.T) {
	t.Parallel()

	runs := &stubRunRepo{}
	proposals := &stubProposalRepo{}
	svc := newProposalService(runs, proposals, &stubConversationRepo{})

	saved, err := svc.persistProposals(t.Context(), proposalTestParams(nil, nil))
	require.NoError(t, err)

	assert.Nil(t, saved)
	assert.Empty(t, runs.created, "a turn that proposed nothing must not open an agent run")
	assert.Empty(t, proposals.created)
}

func TestPersistProposals_OpensOneChatRunForTheTurn(t *testing.T) {
	t.Parallel()

	runs := &stubRunRepo{}
	proposals := &stubProposalRepo{}
	svc := newProposalService(runs, proposals, &stubConversationRepo{})

	params := proposalTestParams([]serviceports.PendingAction{
		{ToolName: "reassign_move", Rationale: "Driver is out of hours", Tier: agent.TierPropose},
		{ToolName: "hold_shipment", Rationale: "Consignee closed", Tier: agent.TierPropose},
	}, nil)

	_, err := svc.persistProposals(t.Context(), params)
	require.NoError(t, err)

	require.Len(t, runs.created, 1, "both proposals belong to the one turn that raised them")
	run := runs.created[0]
	assert.Equal(t, agent.TypeAssistantChat, run.AgentType)
	assert.Equal(t, agent.SubjectAssistantThread, run.SubjectType)
	assert.Equal(t, params.Thread.ID, run.SubjectID)
	assert.Equal(t, agent.RunTriggerChat, run.Trigger)
	assert.Equal(t, params.Definition.ID, run.AgentDefinitionID)
	assert.Equal(t, "test-model", run.ModelIdentifier)
	assert.NotEmpty(t, run.InputContextHash)

	require.Len(t, proposals.created, 2)
	for _, proposal := range proposals.created {
		assert.Equal(t, run.ID, proposal.RunID)
		assert.Equal(t, agent.ProposalStatusPending, proposal.Status)
	}
}

// A persisted proposal has to be pending, because the whole point of the record
// is that a person has not decided yet.
func TestPersistProposals_RecordsThePendingStatusAndTier(t *testing.T) {
	t.Parallel()

	proposals := &stubProposalRepo{}
	svc := newProposalService(&stubRunRepo{}, proposals, &stubConversationRepo{})

	_, err := svc.persistProposals(t.Context(), proposalTestParams([]serviceports.PendingAction{
		{
			ToolName:  "reassign_move",
			Arguments: map[string]any{"moveId": "mv_1"},
			Rationale: "Driver is out of hours",
			Tier:      agent.TierActWithApproval,
		},
	}, nil))
	require.NoError(t, err)

	require.Len(t, proposals.created, 1)
	saved := proposals.created[0]
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

	proposals := &stubProposalRepo{}
	svc := newProposalService(&stubRunRepo{}, proposals, &stubConversationRepo{})

	_, err := svc.persistProposals(t.Context(), proposalTestParams(
		[]serviceports.PendingAction{
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

	require.Len(t, proposals.created, 2)
	assert.Equal(t, secondTurn.ID, proposals.created[0].SourceMessageID)
	assert.Equal(t, firstTurn.ID, proposals.created[1].SourceMessageID)
}

// The column is not-null, so a tool taking no arguments must still store an
// object rather than a JSON null.
func TestPersistProposals_StoresEmptyArgumentsAsAnObject(t *testing.T) {
	t.Parallel()

	proposals := &stubProposalRepo{}
	svc := newProposalService(&stubRunRepo{}, proposals, &stubConversationRepo{})

	_, err := svc.persistProposals(t.Context(), proposalTestParams([]serviceports.PendingAction{
		{ToolName: "close_period", Rationale: "Month end"},
	}, nil))
	require.NoError(t, err)

	require.Len(t, proposals.created, 1)
	assert.NotNil(t, proposals.created[0].ToolParams)
	assert.Empty(t, proposals.created[0].ToolParams)
}

// An unset tier must never mean "no restriction". It falls back to the tier that
// requires a person, which is the safe direction to be wrong in.
func TestPersistProposals_FallsBackToTheMostRestrictiveTier(t *testing.T) {
	t.Parallel()

	proposals := &stubProposalRepo{}
	svc := newProposalService(&stubRunRepo{}, proposals, &stubConversationRepo{})

	_, err := svc.persistProposals(t.Context(), proposalTestParams([]serviceports.PendingAction{
		{ToolName: "reassign_move", Rationale: "Out of hours"},
	}, nil))
	require.NoError(t, err)

	require.Len(t, proposals.created, 1)
	assert.Equal(t, agent.TierPropose, proposals.created[0].AutonomyTier)
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

	proposals := &stubProposalRepo{}
	svc := newProposalService(&stubRunRepo{}, proposals, &stubConversationRepo{})

	params := proposalTestParams(
		[]serviceports.PendingAction{{
			ToolName:   "reassign_move",
			Rationale:  "Out of hours",
			Tier:       agent.TierPropose,
			ToolCallID: "call_1",
		}},
		[]conversation.Message{sourceTurn},
	)

	_, err := svc.persistProposals(t.Context(), params)
	require.NoError(t, err)

	require.Len(t, proposals.created, 1)
	evidence := proposals.created[0].Evidence
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

	proposals := &stubProposalRepo{}
	svc := newProposalService(&stubRunRepo{}, proposals, &stubConversationRepo{})

	_, err := svc.persistProposals(t.Context(), proposalTestParams([]serviceports.PendingAction{
		{ToolName: "reassign_move", Rationale: "Out of hours", ToolCallID: "call_missing"},
	}, nil))
	require.NoError(t, err)

	require.Len(t, proposals.created, 1)
	require.Len(t, proposals.created[0].Evidence, 1)
	assert.Equal(t, evidenceTypeThread, proposals.created[0].Evidence[0].Type)
	assert.True(t, proposals.created[0].SourceMessageID.IsNil())
}

func TestPersistProposals_FailsWhenTheRunCannotBeOpened(t *testing.T) {
	t.Parallel()

	proposals := &stubProposalRepo{}
	svc := newProposalService(
		&stubRunRepo{err: errors.New("database is down")},
		proposals,
		&stubConversationRepo{},
	)

	_, err := svc.persistProposals(t.Context(), proposalTestParams([]serviceports.PendingAction{
		{ToolName: "reassign_move", Rationale: "Out of hours"},
	}, nil))

	require.Error(t, err)
	assert.Empty(t, proposals.created, "no proposal may exist without the run it belongs to")
}

func TestListThreadProposals_ScopesToTheThreadAndReportsExecution(t *testing.T) {
	t.Parallel()

	executedAt := int64(1700000000)
	threadID := pulid.MustNew("thr_")
	proposals := &stubProposalRepo{byThread: []*agent.AgentProposal{
		{
			ID:             pulid.MustNew("ap_"),
			ToolName:       "reassign_move",
			ToolParams:     map[string]any{"moveId": "mv_1"},
			Status:         agent.ProposalStatusExecuted,
			ExecutedAt:     &executedAt,
			ExecutionError: "",
		},
		{
			ID:             pulid.MustNew("ap_"),
			ToolName:       "hold_shipment",
			Status:         agent.ProposalStatusExecutionFailed,
			ExecutionError: "shipment is already delivered",
		},
	}}
	conversations := &stubConversationRepo{thread: &conversation.Thread{ID: threadID}}
	svc := newProposalService(&stubRunRepo{}, proposals, conversations)

	result, err := svc.ListThreadProposals(t.Context(), repositories.GetThreadRequest{ID: threadID})
	require.NoError(t, err)

	assert.Equal(t, threadID, proposals.lastList.ThreadID)
	require.Len(t, result, 2)
	assert.Equal(t, agent.ProposalStatusExecuted, result[0].Status)
	require.NotNil(t, result[0].ExecutedAt)
	assert.Equal(t, executedAt, *result[0].ExecutedAt)
	assert.Equal(t, "shipment is already delivered", result[1].ExecutionError)
}

// Reading the thread is the authorization check. Someone else's thread is not
// found, and its proposals must never be reached.
func TestListThreadProposals_RefusesAThreadTheReaderCannotSee(t *testing.T) {
	t.Parallel()

	proposals := &stubProposalRepo{}
	conversations := &stubConversationRepo{getErr: errors.New("thread not found")}
	svc := newProposalService(&stubRunRepo{}, proposals, conversations)

	_, err := svc.ListThreadProposals(
		t.Context(),
		repositories.GetThreadRequest{ID: pulid.MustNew("thr_")},
	)

	require.Error(t, err)
	assert.Empty(t, proposals.lastList.ThreadID, "the proposal query must not run at all")
}
