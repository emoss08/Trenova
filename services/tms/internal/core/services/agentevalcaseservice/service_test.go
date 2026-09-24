package agentevalcaseservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeCases struct {
	byID       map[pulid.ID]*agentquality.EvalCase
	candidates []repositories.EvalCaseCaptureCandidate
	purges     []repositories.PurgeExpiredEvalCasesRequest
	creates    int
}

func newFakeCases() *fakeCases {
	return &fakeCases{byID: make(map[pulid.ID]*agentquality.EvalCase)}
}

func (f *fakeCases) Create(
	_ context.Context,
	entity *agentquality.EvalCase,
) (*agentquality.EvalCase, error) {
	f.creates++
	entity.ID = pulid.MustNew("aec_")
	stored := *entity
	f.byID[entity.ID] = &stored

	return entity, nil
}

func (f *fakeCases) Update(
	_ context.Context,
	entity *agentquality.EvalCase,
) (*agentquality.EvalCase, error) {
	entity.Version++
	stored := *entity
	f.byID[entity.ID] = &stored

	return entity, nil
}

func (f *fakeCases) GetByID(
	_ context.Context,
	req repositories.GetAgentEvalCaseByIDRequest,
) (*agentquality.EvalCase, error) {
	found, ok := f.byID[req.ID]
	if !ok || found.OrganizationID != req.TenantInfo.OrgID {
		return nil, errortypes.NewNotFoundError("AgentEvalCase not found")
	}
	copied := *found

	return &copied, nil
}

func (f *fakeCases) GetByContent(
	_ context.Context,
	req repositories.GetAgentEvalCaseByContentRequest,
) (*agentquality.EvalCase, error) {
	for _, found := range f.byID {
		if found.OrganizationID == req.TenantInfo.OrgID &&
			found.AgentDefinitionID == req.AgentDefinitionID &&
			found.ContentHash == req.ContentHash {
			copied := *found
			return &copied, nil
		}
	}

	return nil, errortypes.NewNotFoundError("AgentEvalCase not found")
}

func (f *fakeCases) GetByProposal(
	_ context.Context,
	req repositories.GetAgentEvalCaseByProposalRequest,
) (*agentquality.EvalCase, error) {
	for _, found := range f.byID {
		if found.OrganizationID == req.TenantInfo.OrgID && found.SourceProposalID != nil &&
			*found.SourceProposalID == req.ProposalID {
			copied := *found
			return &copied, nil
		}
	}

	return nil, errortypes.NewNotFoundError("AgentEvalCase not found")
}

func (f *fakeCases) ListConnection(
	context.Context,
	*repositories.ListAgentEvalCaseConnectionRequest,
) (*pagination.CursorListResult[*agentquality.EvalCase], error) {
	return &pagination.CursorListResult[*agentquality.EvalCase]{}, nil
}

func (f *fakeCases) ListCaptureCandidates(
	context.Context,
	repositories.ListEvalCaseCaptureCandidatesRequest,
) ([]repositories.EvalCaseCaptureCandidate, error) {
	return f.candidates, nil
}

func (f *fakeCases) PurgeExpired(
	_ context.Context,
	req repositories.PurgeExpiredEvalCasesRequest,
) (int, error) {
	f.purges = append(f.purges, req)

	return 0, nil
}

func (f *fakeCases) PurgeOrphaned(
	context.Context,
	repositories.PurgeOrphanedEvalCasesRequest,
) (int, error) {
	return 0, nil
}

type fakeProposals struct{ proposal *agent.AgentProposal }

func (f fakeProposals) GetByID(
	_ context.Context,
	req repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	if f.proposal == nil || f.proposal.ID != req.ID {
		return nil, errortypes.NewNotFoundError("proposal not found")
	}

	return f.proposal, nil
}

type fakeDecisions struct{ decisions []*agent.AgentDecision }

func (f fakeDecisions) ListByProposals(
	context.Context,
	repositories.ListAgentDecisionsByProposalsRequest,
) ([]*agent.AgentDecision, error) {
	return f.decisions, nil
}

type fakeRuns struct{ run *agent.AgentRun }

func (f fakeRuns) GetByID(
	context.Context,
	repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return f.run, nil
}

type fakeDefinitions struct{ definition *agentdefinition.Definition }

func (f fakeDefinitions) GetByID(
	context.Context,
	repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return f.definition, nil
}

type fakeConversations struct {
	thread   *conversation.Thread
	messages []conversation.Message
}

func (f fakeConversations) GetThreadOwned(
	context.Context,
	repositories.GetThreadOwnedRequest,
) (*conversation.Thread, error) {
	return f.thread, nil
}

func (f fakeConversations) ListMessages(
	context.Context,
	repositories.ListMessagesRequest,
) ([]conversation.Message, error) {
	return f.messages, nil
}

type fakeRetention struct{ items []*tenant.DataRetention }

func (f fakeRetention) List(context.Context) (
	*pagination.ListResult[*tenant.DataRetention],
	error,
) {
	return &pagination.ListResult[*tenant.DataRetention]{Items: f.items}, nil
}

type fakeFeedback struct {
	rows  map[pulid.ID]*aifeedback.Feedback
	links []repositories.LinkAIFeedbackEvalCaseRequest
}

func (f *fakeFeedback) ListByIDs(
	_ context.Context,
	req repositories.ListAIFeedbackByIDsRequest,
) ([]*aifeedback.Feedback, error) {
	out := make([]*aifeedback.Feedback, 0, len(req.IDs))
	for _, id := range req.IDs {
		row, ok := f.rows[id]
		if ok && row.OrganizationID == req.TenantInfo.OrgID &&
			row.BusinessUnitID == req.TenantInfo.BuID {
			out = append(out, row)
		}
	}

	return out, nil
}

func (f *fakeFeedback) LinkEvalCase(
	_ context.Context,
	req repositories.LinkAIFeedbackEvalCaseRequest,
) error {
	f.links = append(f.links, req)
	if row, ok := f.rows[req.FeedbackID]; ok {
		caseID := req.EvalCaseID
		row.EvalCaseID = &caseID
	}

	return nil
}

type world struct {
	tenant     pagination.TenantInfo
	definition *agentdefinition.Definition
	thread     *conversation.Thread
	run        *agent.AgentRun
	proposal   *agent.AgentProposal
	messages   []conversation.Message
	cases      *fakeCases
	feedback   *fakeFeedback
	service    *Service
}

func newWorld(t *testing.T, decision *agent.AgentDecision) *world {
	t.Helper()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	definition := &agentdefinition.Definition{
		ID:             pulid.MustNew("agd_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           "Billing desk",
		Instructions:   "Clear the billing queue.",
		ToolNames:      []string{"get_customer", "update_rate"},
		Version:        4,
	}
	thread := &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		UserID:            pulid.MustNew("usr_"),
		AgentDefinitionID: definition.ID,
	}

	record := map[string]any{"name": "Acme Foods", "creditLimit": 50000}
	payload, err := sonic.MarshalString(record)
	require.NoError(t, err)
	proposing := pulid.MustNew("amsg_")
	liked := pulid.MustNew("amsg_")
	messages := []conversation.Message{
		{
			ID:        pulid.MustNew("amsg_"),
			Role:      conversation.RoleUser,
			Content:   "Earlier question",
			CreatedAt: 100,
		},
		{
			ID:        pulid.MustNew("amsg_"),
			Role:      conversation.RoleAssistant,
			Content:   "Earlier answer",
			CreatedAt: 101,
		},
		{
			ID:          pulid.MustNew("amsg_"),
			Role:        conversation.RoleUser,
			Content:     "Rate S-100 at 1,200",
			PageContext: &agent.PageContext{Path: "/shipments/S-100"},
			Mentions: []agent.EntityRef{
				{Type: "shipment", ID: pulid.MustNew("shp_").String(), Label: "S-100"},
			},
			CreatedAt: 200,
		},
		{
			ID:   pulid.MustNew("amsg_"),
			Role: conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{
				{
					ID:        "call_1",
					Name:      "get_customer",
					Arguments: map[string]any{"name": "Acme Foods"},
				},
				{ID: "call_2", Name: "find_tools"},
			},
			CreatedAt: 201,
		},
		{
			ID:         pulid.MustNew("amsg_"),
			Role:       conversation.RoleTool,
			ToolName:   "get_customer",
			ToolCallID: "call_1",
			Content:    agentruntime.FenceToolResult("get_customer", payload),
			CreatedAt:  202,
		},
		{
			ID:         pulid.MustNew("amsg_"),
			Role:       conversation.RoleTool,
			ToolName:   "find_tools",
			ToolCallID: "call_2",
			Content:    agentruntime.FenceToolResult("find_tools", `{"tools":[]}`),
			CreatedAt:  202,
		},
		{
			ID:        proposing,
			Role:      conversation.RoleAssistant,
			Content:   "I proposed a rate.",
			CreatedAt: 203,
		},
		{ID: liked, Role: conversation.RoleAssistant, Content: "Done.", CreatedAt: 204},
		{
			ID:        pulid.MustNew("amsg_"),
			Role:      conversation.RoleUser,
			Content:   "Next question",
			CreatedAt: 300,
		},
	}

	run := &agent.AgentRun{
		ID:                pulid.MustNew("ar_"),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		AgentDefinitionID: definition.ID,
		Trigger:           agent.RunTriggerChat,
		SubjectType:       agent.SubjectAssistantThread,
		SubjectID:         thread.ID,
		CreatedAt:         250,
	}
	proposal := &agent.AgentProposal{
		ID:              pulid.MustNew("aprop_"),
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  tenantInfo.BuID,
		RunID:           run.ID,
		ToolName:        "update_rate",
		ToolParams:      map[string]any{"shipmentId": "shp_1", "rate": float64(1200)},
		Status:          agent.ProposalStatusExecuted,
		SourceMessageID: proposing,
	}

	decisions := []*agent.AgentDecision{}
	if decision != nil {
		decision.ProposalID = &proposal.ID
		decisions = append(decisions, decision)
	}

	cases := newFakeCases()
	feedback := &fakeFeedback{rows: map[pulid.ID]*aifeedback.Feedback{}}
	service := &Service{
		l:             zap.NewNop(),
		cases:         cases,
		proposals:     fakeProposals{proposal: proposal},
		decisions:     fakeDecisions{decisions: decisions},
		runs:          fakeRuns{run: run},
		definitions:   fakeDefinitions{definition: definition},
		conversations: fakeConversations{thread: thread, messages: messages},
		feedback:      feedback,
		retention:     fakeRetention{},
		redactor:      testRedactor(t),
		now:           func() int64 { return 1_700_000_000 },
	}

	return &world{
		tenant:     tenantInfo,
		definition: definition,
		thread:     thread,
		run:        run,
		proposal:   proposal,
		messages:   messages,
		cases:      cases,
		feedback:   feedback,
		service:    service,
	}
}

func (w *world) rate(rating aifeedback.Rating, target pulid.ID) *aifeedback.Feedback {
	threadID := w.thread.ID
	turnID := pulid.MustNew("atrn_")
	row := &aifeedback.Feedback{
		ID:             pulid.MustNew("aifb_"),
		OrganizationID: w.tenant.OrgID,
		BusinessUnitID: w.tenant.BuID,
		UserID:         w.thread.UserID,
		TargetType:     aifeedback.TargetAssistantMessage,
		TargetID:       target,
		ThreadID:       &threadID,
		TurnID:         &turnID,
		Rating:         rating,
	}
	w.feedback.rows[row.ID] = row

	return row
}

func (w *world) fromProposal(t *testing.T) *serviceports.EvalCaseCapture {
	t.Helper()

	request := &serviceports.CreateEvalCaseFromProposalRequest{
		ProposalID: w.proposal.ID,
		TenantInfo: w.tenant,
	}
	captured, err := w.service.CreateFromProposal(t.Context(), request, nil)
	require.NoError(t, err)

	return captured
}

func TestCreateFromProposal_FreezesTheTurnWithTheCorrectedParameters(t *testing.T) {
	t.Parallel()

	w := newWorld(t, &agent.AgentDecision{
		Decision:      agent.DecisionModified,
		Modifications: map[string]any{"rate": float64(1350)},
	})
	captured := w.fromProposal(t)
	evalCase := captured.Case

	assert.False(t, captured.Duplicate)
	assert.Equal(t, agentquality.CaseSourceDecidedProposal, evalCase.Source)
	assert.Equal(t, agentquality.CaseStatusCandidate, evalCase.Status)
	assert.Equal(t, "Rate S-100 at 1,200", evalCase.Input)
	require.Len(t, evalCase.History, 2)
	assert.Equal(t, "Earlier question", evalCase.History[0].Content)
	assert.Equal(t, "/shipments/S-100", evalCase.PageContext.Path)
	require.Len(t, evalCase.Mentions, 1)
	assert.Equal(t, w.thread.ID, *evalCase.SourceThreadID)
	assert.Equal(t, w.proposal.SourceMessageID, *evalCase.SourceMessageID)
	assert.Equal(t, w.run.ID, *evalCase.SourceRunID)

	require.Len(t, evalCase.ToolFixtures, 1, "runtime tools are not fixtures")
	fixture := evalCase.ToolFixtures[0]
	assert.Equal(t, "get_customer", fixture.Tool)
	assert.Equal(t, map[string]any{"name": "Acme Foods"}, fixture.Args)
	result, ok := fixture.Result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "[restricted:creditLimit]", result["creditLimit"])
	require.NotNil(t, evalCase.Redaction)
	assert.NotEmpty(t, evalCase.Redaction.Fields)

	require.Len(t, evalCase.Expected.Proposals, 1)
	expected := evalCase.Expected.Proposals[0]
	assert.False(t, expected.Rejected)
	assert.Equal(
		t,
		float64(1350),
		expected.Params["rate"],
		"the case expects what the person approved",
	)
	assert.ElementsMatch(t, w.definition.EffectiveToolNames(), evalCase.HeldTools)
	require.NotNil(t, evalCase.CapturedFingerprint)
	assert.Equal(t, int64(4), evalCase.CapturedFingerprint.DefinitionVersion)
	assert.NotEmpty(t, evalCase.ContentHash)
}

func TestCreateFromProposal_RejectionIsExpectedToStayDropped(t *testing.T) {
	t.Parallel()

	w := newWorld(t, &agent.AgentDecision{Decision: agent.DecisionRejected})
	evalCase := w.fromProposal(t).Case

	require.Len(t, evalCase.Expected.Proposals, 1)
	assert.True(t, evalCase.Expected.Proposals[0].Rejected)
	assert.Equal(t, float64(1200), evalCase.Expected.Proposals[0].Params["rate"])
	assert.Contains(t, evalCase.Title, "Rejected")
}

func TestCreateFromProposal_UndecidedIsRefused(t *testing.T) {
	t.Parallel()

	w := newWorld(t, nil)
	request := &serviceports.CreateEvalCaseFromProposalRequest{
		ProposalID: w.proposal.ID,
		TenantInfo: w.tenant,
	}
	_, err := w.service.CreateFromProposal(t.Context(), request, nil)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestCreateFromProposal_DedupesByProposal(t *testing.T) {
	t.Parallel()

	w := newWorld(t, &agent.AgentDecision{Decision: agent.DecisionAccepted})
	first := w.fromProposal(t)
	second := w.fromProposal(t)

	assert.False(t, first.Duplicate)
	assert.True(t, second.Duplicate)
	assert.Equal(t, first.Case.ID, second.Case.ID)
	assert.Equal(t, 1, w.cases.creates)
}

func TestCreateFromMessage_DedupesByContentHash(t *testing.T) {
	t.Parallel()

	w := newWorld(t, &agent.AgentDecision{Decision: agent.DecisionAccepted})
	liked := w.messages[7].ID
	request := &serviceports.CreateEvalCaseFromMessageRequest{
		ThreadID:   w.thread.ID,
		MessageID:  liked,
		TenantInfo: w.tenant,
	}

	first, err := w.service.CreateFromMessage(t.Context(), request, nil)
	require.NoError(t, err)
	second, err := w.service.CreateFromMessage(t.Context(), request, nil)
	require.NoError(t, err)

	assert.False(t, first.Duplicate)
	assert.True(t, second.Duplicate, "the same frozen input is the same case")
	assert.Equal(t, first.Case.ID, second.Case.ID)
	assert.Equal(t, 1, w.cases.creates)

	evalCase := first.Case
	assert.Equal(t, agentquality.CaseSourceThumbsUp, evalCase.Source)
	assert.Equal(t, liked, *evalCase.SourceMessageID)
	assert.Nil(t, evalCase.SourceFeedbackID)
	require.Len(t, evalCase.Expected.Tools, 1)
	assert.Equal(t, "get_customer", evalCase.Expected.Tools[0].Name)
}

func TestCreateFromFeedback_CapturesTheLikedReplyAndLinksTheRating(t *testing.T) {
	t.Parallel()

	w := newWorld(t, nil)
	liked := w.messages[7].ID
	rating := w.rate(aifeedback.RatingPositive, liked)
	request := &serviceports.CreateEvalCaseFromFeedbackRequest{
		FeedbackID: rating.ID,
		TenantInfo: w.tenant,
	}

	first, err := w.service.CreateFromFeedback(t.Context(), request, nil)
	require.NoError(t, err)
	evalCase := first.Case
	assert.Equal(t, agentquality.CaseSourceThumbsUp, evalCase.Source)
	assert.Equal(t, liked, *evalCase.SourceMessageID)
	assert.Equal(t, rating.ID, *evalCase.SourceFeedbackID)
	assert.Equal(t, *rating.TurnID, *evalCase.SourceTurnID)
	require.Len(t, w.feedback.links, 1)
	assert.Equal(t, evalCase.ID, w.feedback.links[0].EvalCaseID)

	second, err := w.service.CreateFromFeedback(t.Context(), request, nil)
	require.NoError(t, err)
	assert.True(t, second.Duplicate)
	assert.Equal(t, evalCase.ID, second.Case.ID)
	assert.Len(t, w.feedback.links, 1, "a rating already linked is not linked again")
	assert.Equal(t, 1, w.cases.creates)
}

func TestCreateFromFeedback_RefusesAThumbsDown(t *testing.T) {
	t.Parallel()

	w := newWorld(t, nil)
	rating := w.rate(aifeedback.RatingNegative, w.messages[7].ID)
	_, err := w.service.CreateFromFeedback(
		t.Context(),
		&serviceports.CreateEvalCaseFromFeedbackRequest{
			FeedbackID: rating.ID,
			TenantInfo: w.tenant,
		},
		nil,
	)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Zero(t, w.cases.creates)
	assert.Empty(t, w.feedback.links)
}

func TestCreateFromFeedback_RefusesARatingFromAnotherTenant(t *testing.T) {
	t.Parallel()

	w := newWorld(t, nil)
	rating := w.rate(aifeedback.RatingPositive, w.messages[7].ID)
	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	_, err := w.service.CreateFromFeedback(
		t.Context(),
		&serviceports.CreateEvalCaseFromFeedbackRequest{
			FeedbackID: rating.ID,
			TenantInfo: other,
		},
		nil,
	)

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Zero(t, w.cases.creates)
}

func TestCreateFromMessage_RefusesAQuestion(t *testing.T) {
	t.Parallel()

	w := newWorld(t, nil)
	request := &serviceports.CreateEvalCaseFromMessageRequest{
		ThreadID:   w.thread.ID,
		MessageID:  w.messages[2].ID,
		TenantInfo: w.tenant,
	}
	_, err := w.service.CreateFromMessage(t.Context(), request, nil)

	require.Error(t, err)
}

func TestCreateCurated_ValidatesBeforeSaving(t *testing.T) {
	t.Parallel()

	w := newWorld(t, nil)
	_, err := w.service.CreateCurated(t.Context(), &serviceports.CreateCuratedEvalCaseRequest{
		AgentDefinitionID: w.definition.ID,
		Input:             "  ",
		Expected: agentquality.Expected{
			Tools: []agentquality.ExpectedTool{{
				Name:  "update_rate",
				Rules: map[string]agentquality.Tolerance{"rate": {Kind: "fuzzy"}},
			}},
		},
		TenantInfo: w.tenant,
	}, nil)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.Contains(t, fields, "input")
	assert.Contains(t, fields, "expected.tools[0].rules.rate.kind")
	assert.Zero(t, w.cases.creates)
}

func TestSetStatus_FollowsTheWorkflow(t *testing.T) {
	t.Parallel()

	w := newWorld(t, &agent.AgentDecision{Decision: agent.DecisionAccepted})
	evalCase := w.fromProposal(t).Case

	active, err := w.service.SetStatus(t.Context(), &serviceports.SetEvalCaseStatusRequest{
		ID:         evalCase.ID,
		Status:     agentquality.CaseStatusActive,
		TenantInfo: w.tenant,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, agentquality.CaseStatusActive, active.Status)

	_, err = w.service.SetStatus(t.Context(), &serviceports.SetEvalCaseStatusRequest{
		ID:         evalCase.ID,
		Status:     agentquality.CaseStatusCandidate,
		TenantInfo: w.tenant,
	}, nil)
	require.Error(t, err, "an active case does not go back to being a candidate")

	_, err = w.service.SetStatus(t.Context(), &serviceports.SetEvalCaseStatusRequest{
		ID:         evalCase.ID,
		Status:     agentquality.CaseStatusQuarantined,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: w.tenant.BuID},
	}, nil)
	require.Error(t, err, "another organization cannot reach the case")
}

func TestCaptureCandidates_CountsWhatHappened(t *testing.T) {
	t.Parallel()

	w := newWorld(t, &agent.AgentDecision{Decision: agent.DecisionAccepted})
	candidate := repositories.EvalCaseCaptureCandidate{
		OrganizationID:    w.tenant.OrgID,
		BusinessUnitID:    w.tenant.BuID,
		ProposalID:        w.proposal.ID,
		AgentDefinitionID: w.definition.ID,
	}
	missing := candidate
	missing.ProposalID = pulid.MustNew("aprop_")
	w.cases.candidates = []repositories.EvalCaseCaptureCandidate{candidate, missing}

	result, err := w.service.CaptureCandidates(
		t.Context(),
		serviceports.CaptureEvalCaseCandidatesRequest{},
	)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Scanned)
	assert.Equal(t, 1, result.Captured)
	assert.Equal(t, 1, result.Failed)

	again, err := w.service.CaptureCandidates(
		t.Context(),
		serviceports.CaptureEvalCaseCandidatesRequest{},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, again.Duplicates)
}

func TestPurge_AppliesEachOrganizationsRetention(t *testing.T) {
	t.Parallel()

	w := newWorld(t, nil)
	keepForever := &tenant.DataRetention{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: w.tenant.BuID,
	}
	yearly := &tenant.DataRetention{
		OrganizationID:               w.tenant.OrgID,
		BusinessUnitID:               w.tenant.BuID,
		AgentEvalCaseRetentionPeriod: 365,
	}
	w.service.retention = fakeRetention{items: []*tenant.DataRetention{keepForever, yearly}}

	_, err := w.service.Purge(t.Context(), serviceports.PurgeEvalCasesRequest{Now: 1_700_000_000})
	require.NoError(t, err)

	require.Len(t, w.cases.purges, 2)
	scoped := w.cases.purges[0]
	assert.Equal(t, w.tenant.OrgID, scoped.TenantInfo.OrgID)
	assert.Equal(t, int64(1_700_000_000-365*86400), scoped.CreatedBefore)
	assert.False(t, scoped.AllTenants)
	assert.True(t, w.cases.purges[1].AllTenants, "explicit expiry applies everywhere")
	assert.Zero(t, w.cases.purges[1].CreatedBefore)
}
