package agentjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs/agentcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentevaluationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
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
	agentevaluationports.AgentEvaluationRepository
	stored *agentevaluation.Evaluation
}

func (s *evalStore) GetByID(
	context.Context,
	agentevaluationports.GetAgentEvaluationByIDRequest,
) (*agentevaluation.Evaluation, error) {
	copied := *s.stored

	return &copied, nil
}

func (s *evalStore) Update(
	_ context.Context,
	entity *agentevaluation.Evaluation,
) (*agentevaluation.Evaluation, error) {
	entity.Version++
	copied := *entity
	s.stored = &copied

	return entity, nil
}

type definitionStore struct {
	agentdefinitionports.AgentDefinitionRepository
	definition *agentdefinition.Definition
}

func (s *definitionStore) GetByID(
	context.Context,
	agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return s.definition, nil
}

type runStore struct {
	agentruntimeports.AgentRunRepository
	run *agent.AgentRun
}

func (s *runStore) GetByID(
	context.Context,
	agentruntimeports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return s.run, nil
}

type proposalStore struct {
	agentproposalports.AgentProposalRepository
	proposals []*agent.AgentProposal
}

func (s *proposalStore) ListByRun(
	context.Context,
	agentproposalports.ListAgentProposalsByRunRequest,
) ([]*agent.AgentProposal, error) {
	return s.proposals, nil
}

type decisionStore struct {
	agentdecisionports.AgentDecisionRepository
	decisions []*agentdecision.AgentDecision
}

func (s *decisionStore) ListByProposals(
	context.Context,
	agentdecisionports.ListAgentDecisionsByProposalsRequest,
) ([]*agentdecision.AgentDecision, error) {
	return s.decisions, nil
}

type conversationStore struct {
	conversationports.ConversationRepository
	thread   *conversation.Thread
	messages []conversation.Message
}

func (s *conversationStore) GetThreadOwned(
	context.Context,
	conversationports.GetThreadOwnedRequest,
) (*conversation.Thread, error) {
	if s.thread == nil {
		return nil, errortypes.NewNotFoundError("Thread not found")
	}

	return s.thread, nil
}

func (s *conversationStore) ListMessages(
	context.Context,
	conversationports.ListMessagesRequest,
) ([]conversation.Message, error) {
	return s.messages, nil
}

type caseStore struct {
	agentqualityports.AgentEvalCaseRepository
	evalCase *agentquality.EvalCase
}

func (s *caseStore) GetByID(
	context.Context,
	agentqualityports.GetAgentEvalCaseByIDRequest,
) (*agentquality.EvalCase, error) {
	if s.evalCase == nil {
		return nil, errortypes.NewNotFoundError("AgentEvalCase not found")
	}

	return s.evalCase, nil
}

type recordingContexts struct {
	requests []*agentruntimeports.RuntimeContextRequest
}

func (c *recordingContexts) Build(
	_ context.Context,
	req *agentruntimeports.RuntimeContextRequest,
) (agentturn.RuntimeContext, error) {
	c.requests = append(c.requests, req)

	return agentturn.RuntimeContext{Timezone: "America/Chicago"}, nil
}

type replayWorld struct {
	tenant        pagination.TenantInfo
	owner         pulid.ID
	evaluations   *evalStore
	conversations *conversationStore
	cases         *caseStore
	contexts      *recordingContexts
	users         *tenantportsmock.MockUserRepository
	activities    *Activities
	payload       *agentcontract.AgentEvaluationPayload
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
	evaluation := &agentevaluation.Evaluation{
		ID:                pulid.MustNew(agentevaluation.EvaluationIDPrefix),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		AgentDefinitionID: definition.ID,
		SourceRunID:       run.ID,
		Status:            agentevaluation.EvaluationStatusPending,
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
	users := tenantportsmock.NewMockUserRepository(t)

	activities := NewActivities(ActivitiesParams{
		Logger:       zap.NewNop(),
		Definitions:  &definitionStore{definition: definition},
		RunRepo:      &runStore{run: run},
		ProposalRepo: &proposalStore{proposals: []*agent.AgentProposal{proposal}},
		Evaluations:  evaluations,
		EvalCases:    cases,
		Users:        users,
		Decisions: &decisionStore{decisions: []*agentdecision.AgentDecision{{
			ProposalID:    &proposal.ID,
			Decision:      agentdecision.DecisionModified,
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
		payload: &agentcontract.AgentEvaluationPayload{
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
			func(req tenantports.GetTenantMemberRequest) bool {
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
	assert.Equal(t, permissionports.PrincipalTypeUser, actor.PrincipalType)
	assert.Equal(t, w.owner, actor.UserID)
	assert.Equal(t, w.owner, actor.PrincipalID)
	assert.Equal(t, w.tenant.OrgID, actor.OrganizationID)
	require.Len(t, w.contexts.requests, 1)
	assert.Equal(t, w.owner, w.contexts.requests[0].Actor.UserID,
		"the tools offered are the person's, resolved now")

	assert.Equal(t, aiusageports.AIUsagePurposeEvaluation, opened.request.UsagePurpose)
	assert.Equal(t, aiusageports.AIUsagePurposeEvaluation, opened.request.AttributedPurpose())
	assert.True(t, opened.request.Definition.SimulationMode)
	assert.Equal(t, agentevaluation.EvaluationStatusRunning, w.evaluations.stored.Status)
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
		&agentruntimeports.RunResult{Actions: []agentruntimeports.PendingAction{{
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
	assert.Equal(t, agentevaluation.EvaluationStatusSkipped, inactive.evaluations.stored.Status)
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
	assert.Equal(t, agentevaluation.EvaluationStatusSkipped, removed.evaluations.stored.Status)
	assert.Contains(t, removed.evaluations.stored.ErrorMessage, "no longer belongs")

	deleted := newReplayWorld(t)
	deleted.conversations.thread = nil
	opened, err = deleted.activities.openReplay(t.Context(), deleted.payload)
	require.NoError(t, err)
	assert.Nil(t, opened)
	assert.Equal(t, agentevaluation.EvaluationStatusSkipped, deleted.evaluations.stored.Status)
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
	assert.Equal(t, agentevaluation.EvaluationStatusSkipped, w.evaluations.stored.Status)
}

func TestStoreReplay_ScoresACaseReplay(t *testing.T) {
	t.Parallel()

	w := newReplayWorld(t)
	evalCase := chatCase(w)
	caseEvaluation(w, evalCase)

	originals := expectedOriginals(evalCase)
	_, err := w.activities.storeReplay(t.Context(), w.evaluations.stored, originals,
		&agentruntimeports.RunResult{
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
			Actions: []agentruntimeports.PendingAction{{
				ToolName:  "update_rate",
				Arguments: map[string]any{"shipmentId": "shp_1", "rate": 1350},
			}},
		})
	require.NoError(t, err)

	stored := w.evaluations.stored
	assert.Equal(t, agentevaluation.EvaluationStatusCompleted, stored.Status)
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

	altered := observeReplay(&agentruntimeports.RunResult{OutputAltered: true})
	assert.False(t, altered.refused, "a reply the guard took code out of was still given")

	flagged := observeReplay(&agentruntimeports.RunResult{Messages: []conversation.Message{
		{Role: conversation.RoleAssistant, Refused: true},
	}})
	assert.True(t, flagged.refused)

	answered := observeReplay(&agentruntimeports.RunResult{Messages: []conversation.Message{
		{Role: conversation.RoleTool, Content: `{"rate": 1350}`},
	}})
	assert.False(t, answered.refused)
	assert.Equal(t, []string{`{"rate": 1350}`}, answered.results)
}

func TestObserveReplay_TracesWhichCallsWouldHaveRunUnasked(t *testing.T) {
	t.Parallel()

	observed := observeReplay(&agentruntimeports.RunResult{
		Messages: []conversation.Message{{
			Role: conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{
				{ID: "call_1", Name: "read_inbound_email"},
				{ID: "call_2", Name: "send_customer_email", Arguments: map[string]any{"to": "a"}},
				{ID: "call_3", Name: "update_rate"},
			},
		}},
		Actions: []agentruntimeports.PendingAction{
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

func TestOpenReplay_ABackgroundCaseReplaysAsTheAgentAttributedToTheSystemUser(t *testing.T) {
	t.Parallel()

	w := newReplayWorld(t)
	evalCase := chatCase(w)
	evalCase.Trigger = agent.RunTriggerEvent
	evalCase.SourceThreadID = nil
	evalCase.History = nil
	caseEvaluation(w, evalCase)
	system := pulid.MustNew("usr_")
	w.users.On("GetSystemUser", mock.Anything, []string{"id"}).
		Return(&tenant.User{ID: system}, nil).
		Once()

	opened, err := w.activities.openReplay(t.Context(), w.payload)
	require.NoError(t, err)
	require.NotNil(t, opened)

	actor := opened.request.Actor
	assert.Equal(t, permissionports.PrincipalTypeAgent, actor.PrincipalType)
	require.NotNil(t, opened.request.Definition)
	assert.Equal(t, opened.request.Definition.ID, actor.PrincipalID)
	assert.Equal(t, system, actor.UserID)
	require.Len(t, w.contexts.requests, 1)
	assert.Equal(t, permissionports.PrincipalTypeAgent, w.contexts.requests[0].Actor.PrincipalType)
}
