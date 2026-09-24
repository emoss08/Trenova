package agentjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type evalStore struct {
	repositories.AgentEvaluationRepository
	stored *agent.Evaluation
}

func (s *evalStore) GetByID(
	context.Context,
	repositories.GetAgentEvaluationByIDRequest,
) (*agent.Evaluation, error) {
	copied := *s.stored

	return &copied, nil
}

func (s *evalStore) Update(
	_ context.Context,
	entity *agent.Evaluation,
) (*agent.Evaluation, error) {
	entity.Version++
	copied := *entity
	s.stored = &copied

	return entity, nil
}

type definitionStore struct {
	repositories.AgentDefinitionRepository
	definition *agentdefinition.Definition
}

func (s *definitionStore) GetByID(
	context.Context,
	repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return s.definition, nil
}

type runStore struct {
	repositories.AgentRunRepository
	run *agent.AgentRun
}

func (s *runStore) GetByID(
	context.Context,
	repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return s.run, nil
}

type proposalStore struct {
	repositories.AgentProposalRepository
	proposals []*agent.AgentProposal
}

func (s *proposalStore) ListByRun(
	context.Context,
	repositories.ListAgentProposalsByRunRequest,
) ([]*agent.AgentProposal, error) {
	return s.proposals, nil
}

type decisionStore struct {
	repositories.AgentDecisionRepository
	decisions []*agent.AgentDecision
}

func (s *decisionStore) ListByProposals(
	context.Context,
	repositories.ListAgentDecisionsByProposalsRequest,
) ([]*agent.AgentDecision, error) {
	return s.decisions, nil
}

type conversationStore struct {
	repositories.ConversationRepository
	thread   *conversation.Thread
	messages []conversation.Message
}

func (s *conversationStore) GetThreadOwned(
	context.Context,
	repositories.GetThreadOwnedRequest,
) (*conversation.Thread, error) {
	if s.thread == nil {
		return nil, errortypes.NewNotFoundError("Thread not found")
	}

	return s.thread, nil
}

func (s *conversationStore) ListMessages(
	context.Context,
	repositories.ListMessagesRequest,
) ([]conversation.Message, error) {
	return s.messages, nil
}

type caseStore struct {
	repositories.AgentEvalCaseRepository
	evalCase *agentquality.EvalCase
}

func (s *caseStore) GetByID(
	context.Context,
	repositories.GetAgentEvalCaseByIDRequest,
) (*agentquality.EvalCase, error) {
	if s.evalCase == nil {
		return nil, errortypes.NewNotFoundError("AgentEvalCase not found")
	}

	return s.evalCase, nil
}

type recordingContexts struct {
	requests []*serviceports.RuntimeContextRequest
}

func (c *recordingContexts) Build(
	_ context.Context,
	req *serviceports.RuntimeContextRequest,
) (agentdefinition.RuntimeContext, error) {
	c.requests = append(c.requests, req)

	return agentdefinition.RuntimeContext{Timezone: "America/Chicago"}, nil
}

type replayWorld struct {
	tenant        pagination.TenantInfo
	owner         pulid.ID
	evaluations   *evalStore
	conversations *conversationStore
	cases         *caseStore
	contexts      *recordingContexts
	users         *mocks.MockUserRepository
	activities    *Activities
	payload       *AgentEvaluationPayload
}

func newReplayWorld(t *testing.T) *replayWorld {
	t.Helper()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	owner := pulid.MustNew("usr_")
	definition := &agentdefinition.Definition{
		ID:             pulid.MustNew("agd_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           "Billing desk",
		ToolNames:      []string{"update_rate"},
		Version:        7,
	}
	thread := &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		UserID:            owner,
		AgentDefinitionID: definition.ID,
	}
	run := &agent.AgentRun{
		ID:                pulid.MustNew("ar_"),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		AgentDefinitionID: definition.ID,
		Trigger:           agent.RunTriggerChat,
		SubjectType:       agent.SubjectAssistantThread,
		SubjectID:         thread.ID,
		CreatedAt:         500,
	}
	proposal := &agent.AgentProposal{
		ID:         pulid.MustNew("aprop_"),
		RunID:      run.ID,
		ToolName:   "update_rate",
		ToolParams: map[string]any{"shipmentId": "shp_1", "rate": float64(1200)},
		Status:     agent.ProposalStatusExecuted,
	}
	evaluation := &agent.Evaluation{
		ID:                pulid.MustNew(agent.EvaluationIDPrefix),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		AgentDefinitionID: definition.ID,
		SourceRunID:       run.ID,
		Status:            agent.EvaluationStatusPending,
		Trigger:           agent.RunTriggerChat,
		SubjectType:       agent.SubjectAssistantThread,
		SubjectID:         thread.ID,
	}

	evaluations := &evalStore{stored: evaluation}
	conversations := &conversationStore{
		thread: thread,
		messages: []conversation.Message{
			{Role: conversation.RoleUser, Content: "Rate S-100", CreatedAt: 400},
		},
	}
	cases := &caseStore{}
	contexts := &recordingContexts{}
	users := mocks.NewMockUserRepository(t)

	activities := NewActivities(ActivitiesParams{
		Logger:       zap.NewNop(),
		Definitions:  &definitionStore{definition: definition},
		RunRepo:      &runStore{run: run},
		ProposalRepo: &proposalStore{proposals: []*agent.AgentProposal{proposal}},
		Evaluations:  evaluations,
		EvalCases:    cases,
		Users:        users,
		Decisions: &decisionStore{decisions: []*agent.AgentDecision{{
			ProposalID:    &proposal.ID,
			Decision:      agent.DecisionModified,
			Modifications: map[string]any{"rate": float64(1350)},
		}}},
		Conversations: conversations,
		Contexts:      contexts,
	})

	return &replayWorld{
		tenant:        tenantInfo,
		owner:         owner,
		evaluations:   evaluations,
		conversations: conversations,
		cases:         cases,
		contexts:      contexts,
		users:         users,
		activities:    activities,
		payload: &AgentEvaluationPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
			},
			EvaluationID: evaluation.ID,
		},
	}
}

func (w *replayWorld) member(status domaintypes.Status) {
	w.users.EXPECT().
		GetTenantMember(mock.Anything, mock.MatchedBy(
			func(req repositories.GetTenantMemberRequest) bool {
				return req.UserID == w.owner && req.TenantInfo == w.tenant
			},
		)).
		Return(&tenant.User{ID: w.owner, Status: status}, nil).
		Once()
}

func TestOpenReplay_AChatRunReplaysAsThePersonWhoAsked(t *testing.T) {
	t.Parallel()

	w := newReplayWorld(t)
	w.member(domaintypes.StatusActive)

	opened, err := w.activities.openReplay(t.Context(), w.payload)
	require.NoError(t, err)
	require.NotNil(t, opened)

	actor := opened.request.Actor
	assert.Equal(t, serviceports.PrincipalTypeUser, actor.PrincipalType)
	assert.Equal(t, w.owner, actor.UserID)
	assert.Equal(t, w.owner, actor.PrincipalID)
	assert.Equal(t, w.tenant.OrgID, actor.OrganizationID)
	require.Len(t, w.contexts.requests, 1)
	assert.Equal(t, w.owner, w.contexts.requests[0].Actor.UserID,
		"the tools offered are the person's, resolved now")

	assert.Equal(t, serviceports.AIUsagePurposeEvaluation, opened.request.UsagePurpose)
	assert.Equal(t, serviceports.AIUsagePurposeEvaluation, opened.request.AttributedPurpose())
	assert.True(t, opened.request.Definition.SimulationMode)
	assert.Equal(t, agent.EvaluationStatusRunning, w.evaluations.stored.Status)
	require.NotNil(t, w.evaluations.stored.Fingerprint)
	assert.Equal(t, int64(7), w.evaluations.stored.Fingerprint.DefinitionVersion)
}

func TestOpenReplay_JudgesAgainstWhatThePersonApproved(t *testing.T) {
	t.Parallel()

	w := newReplayWorld(t)
	w.member(domaintypes.StatusActive)

	opened, err := w.activities.openReplay(t.Context(), w.payload)
	require.NoError(t, err)
	require.Len(t, opened.originals, 1)
	original := opened.originals[0]
	assert.Equal(t, float64(1200), original.Params["rate"])
	assert.Equal(t, float64(1350), original.CorrectedParams["rate"])

	_, err = w.activities.storeReplay(t.Context(), w.evaluations.stored, opened.originals,
		&serviceports.RunResult{Actions: []serviceports.PendingAction{{
			ToolName:  "update_rate",
			Arguments: map[string]any{"shipmentId": "shp_1", "rate": 1350},
		}}})
	require.NoError(t, err)

	comparison := w.evaluations.stored.Comparison
	require.NotNil(t, comparison)
	assert.Equal(t, 1, comparison.Agreed, "proposing the corrected rate agrees with the approval")
	assert.Zero(t, comparison.Changed)
}

func TestOpenReplay_SkipsWhenThePersonIsGone(t *testing.T) {
	t.Parallel()

	inactive := newReplayWorld(t)
	inactive.member(domaintypes.StatusInactive)
	opened, err := inactive.activities.openReplay(t.Context(), inactive.payload)
	require.NoError(t, err)
	assert.Nil(t, opened)
	assert.Equal(t, agent.EvaluationStatusSkipped, inactive.evaluations.stored.Status)
	assert.Contains(t,
		inactive.evaluations.stored.ErrorMessage,
		"no longer has an active account",
	)
	assert.Empty(t, inactive.contexts.requests, "nothing runs as anyone else instead")

	removed := newReplayWorld(t)
	removed.users.EXPECT().
		GetTenantMember(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("User not found")).
		Once()
	opened, err = removed.activities.openReplay(t.Context(), removed.payload)
	require.NoError(t, err)
	assert.Nil(t, opened)
	assert.Equal(t, agent.EvaluationStatusSkipped, removed.evaluations.stored.Status)
	assert.Contains(t, removed.evaluations.stored.ErrorMessage, "no longer belongs")

	deleted := newReplayWorld(t)
	deleted.conversations.thread = nil
	opened, err = deleted.activities.openReplay(t.Context(), deleted.payload)
	require.NoError(t, err)
	assert.Nil(t, opened)
	assert.Equal(t, agent.EvaluationStatusSkipped, deleted.evaluations.stored.Status)
}

func caseEvaluation(w *replayWorld, evalCase *agentquality.EvalCase) {
	caseID := evalCase.ID
	w.evaluations.stored.SourceRunID = pulid.Nil
	w.evaluations.stored.EvalCaseID = &caseID
	w.cases.evalCase = evalCase
}

func chatCase(w *replayWorld) *agentquality.EvalCase {
	threadID := w.conversations.thread.ID

	return &agentquality.EvalCase{
		ID:                pulid.MustNew("aec_"),
		OrganizationID:    w.tenant.OrgID,
		BusinessUnitID:    w.tenant.BuID,
		AgentDefinitionID: w.evaluations.stored.AgentDefinitionID,
		Source:            agentquality.CaseSourceDecidedProposal,
		Status:            agentquality.CaseStatusActive,
		Trigger:           agent.RunTriggerChat,
		SourceThreadID:    &threadID,
		Input:             "Rate S-100",
		History: []agentquality.HistoryMessage{
			{Role: conversation.RoleUser, Content: "Earlier"},
			{Role: conversation.RoleAssistant, Content: "Answered"},
		},
		HeldTools: []string{"update_rate"},
		Expected: agentquality.Expected{
			ToolMode: agentquality.ToolMatchAnyOrder,
			Proposals: []agentquality.ExpectedProposal{{
				ToolName: "update_rate",
				Params:   map[string]any{"shipmentId": "shp_1", "rate": float64(1350)},
			}},
			MustMention: []string{"rate"},
		},
	}
}

func TestOpenReplay_ACaseReplaysItsFrozenInputAsItsPerson(t *testing.T) {
	t.Parallel()

	w := newReplayWorld(t)
	caseEvaluation(w, chatCase(w))
	w.member(domaintypes.StatusActive)

	opened, err := w.activities.openReplay(t.Context(), w.payload)
	require.NoError(t, err)
	require.NotNil(t, opened)

	assert.Equal(t, "Rate S-100", opened.request.Input)
	require.Len(t, opened.request.History, 2)
	assert.Equal(t, "Earlier", opened.request.History[0].Content)
	assert.Equal(t, w.owner, opened.request.Actor.UserID)
	require.Len(t, opened.originals, 1)
	assert.Equal(t, float64(1350), opened.originals[0].Params["rate"])
	assert.Equal(t, 1, w.evaluations.stored.OriginalProposals)
}

func TestOpenReplay_ADeletedCaseIsSkipped(t *testing.T) {
	t.Parallel()

	w := newReplayWorld(t)
	caseEvaluation(w, chatCase(w))
	w.cases.evalCase = nil

	opened, err := w.activities.openReplay(t.Context(), w.payload)
	require.NoError(t, err)
	assert.Nil(t, opened)
	assert.Equal(t, agent.EvaluationStatusSkipped, w.evaluations.stored.Status)
}

func TestStoreReplay_ScoresACaseReplay(t *testing.T) {
	t.Parallel()

	w := newReplayWorld(t)
	evalCase := chatCase(w)
	caseEvaluation(w, evalCase)

	originals := expectedOriginals(evalCase)
	_, err := w.activities.storeReplay(t.Context(), w.evaluations.stored, originals,
		&serviceports.RunResult{
			Reply: "I proposed the new rate.",
			Messages: []conversation.Message{
				{
					Role: conversation.RoleAssistant,
					ToolCalls: []conversation.ToolCallRecord{{
						ID:        "call_1",
						Name:      "update_rate",
						Arguments: map[string]any{"shipmentId": "shp_1", "rate": 1350},
					}},
				},
				{
					Role:      conversation.RoleAssistant,
					Kind:      conversation.MessageKindDelegated,
					ToolCalls: []conversation.ToolCallRecord{{ID: "call_9", Name: "outside_tool"}},
				},
			},
			Actions: []serviceports.PendingAction{{
				ToolName:  "update_rate",
				Arguments: map[string]any{"shipmentId": "shp_1", "rate": 1350},
			}},
		})
	require.NoError(t, err)

	stored := w.evaluations.stored
	assert.Equal(t, agent.EvaluationStatusCompleted, stored.Status)
	require.NotNil(t, stored.Checks)
	require.NotNil(t, stored.CaseScore)
	assert.False(t, stored.Checks.HardFailure, "a delegate's tools are not this agent's")
	assert.True(t, stored.Checks.Passed)
	assert.InDelta(t, stored.Checks.Final, *stored.CaseScore, 1e-9)
	require.Len(t, stored.Checks.Calls, 1)
	assert.Equal(t, "update_rate", stored.Checks.Calls[0].ToolName)
	assert.Equal(t, 1, stored.Comparison.Agreed)
}

func TestObserveReplay_ReadsRefusals(t *testing.T) {
	t.Parallel()

	refused := observeReplay(&serviceports.RunResult{OutputRefused: true})
	assert.True(t, refused.refused)

	flagged := observeReplay(&serviceports.RunResult{Messages: []conversation.Message{
		{Role: conversation.RoleAssistant, Refused: true},
	}})
	assert.True(t, flagged.refused)

	answered := observeReplay(&serviceports.RunResult{Messages: []conversation.Message{
		{Role: conversation.RoleTool, Content: `{"rate": 1350}`},
	}})
	assert.False(t, answered.refused)
	assert.Equal(t, []string{`{"rate": 1350}`}, answered.results)
}

func TestObserveReplay_TracesWhichCallsWouldHaveRunUnasked(t *testing.T) {
	t.Parallel()

	observed := observeReplay(&serviceports.RunResult{
		Messages: []conversation.Message{{
			Role: conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{
				{ID: "call_1", Name: "read_inbound_email"},
				{ID: "call_2", Name: "send_customer_email", Arguments: map[string]any{"to": "a"}},
				{ID: "call_3", Name: "update_rate"},
			},
		}},
		Actions: []serviceports.PendingAction{
			{ToolName: "send_customer_email", ToolCallID: "call_2", Tier: agent.TierAutoExecute},
			{ToolName: "update_rate", ToolCallID: "call_3", Tier: agent.TierActWithApproval},
		},
	})

	require.Len(t, observed.trace, 3)
	assert.Equal(t, "read_inbound_email", observed.trace[0].ToolName)
	assert.False(t, observed.trace[0].AutoRun)
	assert.True(t, observed.trace[1].AutoRun)
	assert.Equal(t, map[string]any{"to": "a"}, observed.trace[1].Arguments)
	assert.False(t, observed.trace[2].AutoRun, "a proposal waits for a person")
	assert.Len(t, observed.calls, 3)
}
