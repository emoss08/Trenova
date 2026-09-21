package agentdecisionqueueservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubQueue struct {
	repositories.AgentDecisionQueueRepository

	page  *repositories.PendingDecisionsPage
	count int
	last  repositories.ListPendingDecisionsRequest
}

func (q *stubQueue) ListPending(
	_ context.Context,
	req repositories.ListPendingDecisionsRequest,
) (*repositories.PendingDecisionsPage, error) {
	q.last = req

	return q.page, nil
}

func (q *stubQueue) CountPending(
	_ context.Context,
	_ repositories.ListPendingDecisionsRequest,
) (int, error) {
	return q.count, nil
}

type stubProposals struct {
	repositories.AgentProposalRepository

	byID map[pulid.ID]*agent.AgentProposal
}

func (r *stubProposals) ListByIDs(
	_ context.Context,
	req repositories.ListAgentProposalsByIDsRequest,
) ([]*agent.AgentProposal, error) {
	out := make([]*agent.AgentProposal, 0, len(req.IDs))
	for _, id := range req.IDs {
		if proposal, ok := r.byID[id]; ok {
			out = append(out, proposal)
		}
	}

	return out, nil
}

type stubPlans struct {
	repositories.AgentPlanRepository

	byID map[pulid.ID]*agent.AgentPlan
}

func (r *stubPlans) ListByIDs(
	_ context.Context,
	req repositories.ListAgentPlansByIDsRequest,
) ([]*agent.AgentPlan, error) {
	out := make([]*agent.AgentPlan, 0, len(req.IDs))
	for _, id := range req.IDs {
		if plan, ok := r.byID[id]; ok {
			out = append(out, plan)
		}
	}

	return out, nil
}

type stubDecider struct {
	services.AgentDecisionService

	decided []pulid.ID
	fail    map[pulid.ID]error
	execErr map[pulid.ID]error
}

func (d *stubDecider) DecideWithOutcome(
	_ context.Context,
	req *services.DecideAgentProposalRequest,
	_ *services.RequestActor,
) (*services.DecisionOutcome, error) {
	d.decided = append(d.decided, req.ProposalID)
	if err, ok := d.fail[req.ProposalID]; ok {
		return nil, err
	}
	proposalID := req.ProposalID

	return &services.DecisionOutcome{
		Decision:       &agent.AgentDecision{ProposalID: &proposalID, Decision: req.Decision},
		ExecutionError: d.execErr[req.ProposalID],
	}, nil
}

type stubShadow struct{ shadow bool }

func (s stubShadow) Organization(context.Context, pagination.TenantInfo) (bool, error) {
	return s.shadow, nil
}

func actor() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func proposal(tool string) *agent.AgentProposal {
	return &agent.AgentProposal{ID: pulid.MustNew("ap_"), ToolName: tool, Status: agent.ProposalStatusPending}
}

func newService(proposals *stubProposals, decider *stubDecider) *Service {
	return &Service{
		l:         zap.NewNop(),
		queue:     &stubQueue{},
		proposals: proposals,
		plans:     &stubPlans{},
		decisions: decider,
		shadow:    stubShadow{},
	}
}

// An approver reads the tool once and approves what they read. A batch that
// mixes tools, or that includes a step of a plan, is refused before any
// proposal is touched.
func TestDecideMany_RefusesAMixedBatchBeforeDecidingAnything(t *testing.T) {
	t.Parallel()

	first := proposal("assign_move")
	second := proposal("email_customer")
	planID := pulid.MustNew("apl_")
	step := proposal("assign_move")
	step.PlanID = &planID
	proposals := &stubProposals{byID: map[pulid.ID]*agent.AgentProposal{
		first.ID: first, second.ID: second, step.ID: step,
	}}
	decider := &stubDecider{}
	svc := newService(proposals, decider)

	_, err := svc.DecideMany(t.Context(), &services.DecideAgentProposalsRequest{
		ProposalIDs: []pulid.ID{first.ID, second.ID},
		Decision:    agent.DecisionAccepted,
	}, actor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "one tool at a time")
	assert.Empty(t, decider.decided, "nothing was decided")

	_, err = svc.DecideMany(t.Context(), &services.DecideAgentProposalsRequest{
		ProposalIDs: []pulid.ID{first.ID, step.ID},
		Decision:    agent.DecisionAccepted,
	}, actor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decide the plan")
	assert.Empty(t, decider.decided)

	_, err = svc.DecideMany(t.Context(), &services.DecideAgentProposalsRequest{
		ProposalIDs: []pulid.ID{first.ID, pulid.MustNew("ap_")},
		Decision:    agent.DecisionAccepted,
	}, actor())
	require.Error(t, err)
	assert.True(t, errortypes.IsMultiError(err))
	assert.Empty(t, decider.decided)
}

// Once a batch starts, every proposal is decided in turn and each outcome is
// reported: one that could not be decided neither stops the rest nor hides.
func TestDecideMany_ReportsEachOutcomeAndCarriesOn(t *testing.T) {
	t.Parallel()

	first := proposal("assign_move")
	second := proposal("assign_move")
	third := proposal("assign_move")
	proposals := &stubProposals{byID: map[pulid.ID]*agent.AgentProposal{
		first.ID: first, second.ID: second, third.ID: third,
	}}
	decider := &stubDecider{
		fail:    map[pulid.ID]error{second.ID: errortypes.NewBusinessError("Someone else decided this one")},
		execErr: map[pulid.ID]error{third.ID: errors.New("move is no longer open")},
	}
	svc := newService(proposals, decider)

	results, err := svc.DecideMany(t.Context(), &services.DecideAgentProposalsRequest{
		ProposalIDs: []pulid.ID{first.ID, second.ID, third.ID},
		Decision:    agent.DecisionAccepted,
	}, actor())
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, []pulid.ID{first.ID, second.ID, third.ID}, decider.decided)

	assert.True(t, results[0].Executed)
	assert.Empty(t, results[0].Error)
	require.NotNil(t, results[0].Decision)

	assert.False(t, results[1].Executed)
	assert.Equal(t, "Someone else decided this one", results[1].Error)
	assert.Nil(t, results[1].Decision)

	assert.False(t, results[2].Executed)
	assert.Equal(t, "move is no longer open", results[2].Error)
	require.NotNil(t, results[2].Decision)
}

// An internal fault is not wording for the approver; the row says what to
// do instead.
func TestDecideMany_KeepsInternalFaultsOutOfTheRow(t *testing.T) {
	t.Parallel()

	only := proposal("assign_move")
	decider := &stubDecider{fail: map[pulid.ID]error{only.ID: errors.New("pq: connection reset")}}
	svc := newService(&stubProposals{byID: map[pulid.ID]*agent.AgentProposal{only.ID: only}}, decider)

	results, err := svc.DecideMany(t.Context(), &services.DecideAgentProposalsRequest{
		ProposalIDs: []pulid.ID{only.ID},
		Decision:    agent.DecisionRejected,
		ReasonCode:  "not_now",
	}, actor())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.NotContains(t, results[0].Error, "pq:")
	assert.Contains(t, results[0].Error, "on its own")
}

func TestValidateBatch(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ap_")
	cases := map[string]struct {
		req  services.DecideAgentProposalsRequest
		want string
	}{
		"empty":     {services.DecideAgentProposalsRequest{Decision: agent.DecisionAccepted}, "at least one"},
		"duplicate": {services.DecideAgentProposalsRequest{ProposalIDs: []pulid.ID{id, id}, Decision: agent.DecisionAccepted}, "listed twice"},
		"modified":  {services.DecideAgentProposalsRequest{ProposalIDs: []pulid.ID{id}, Decision: agent.DecisionModified}, "one proposal"},
		"reject without reason": {
			services.DecideAgentProposalsRequest{ProposalIDs: []pulid.ID{id}, Decision: agent.DecisionRejected},
			"Say why",
		},
		"unknown decision": {services.DecideAgentProposalsRequest{ProposalIDs: []pulid.ID{id}, Decision: "Maybe"}, "invalid"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := validateBatch(&tc.req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}

	tooMany := make([]pulid.ID, 0, MaxBatch+1)
	for range MaxBatch + 1 {
		tooMany = append(tooMany, pulid.MustNew("ap_"))
	}
	err := validateBatch(&services.DecideAgentProposalsRequest{ProposalIDs: tooMany, Decision: agent.DecisionAccepted})
	require.Error(t, err)

	require.NoError(t, validateBatch(&services.DecideAgentProposalsRequest{
		ProposalIDs: []pulid.ID{id},
		Decision:    agent.DecisionAccepted,
	}))
}

// The queue lists entries; the records come from their own repositories in
// queue order, and an entry decided between the two reads is dropped rather
// than failing the page. The cursor names where the page ended.
func TestListPending_LoadsRecordsInQueueOrderAndDropsTheGone(t *testing.T) {
	t.Parallel()

	first := proposal("assign_move")
	plan := &agent.AgentPlan{ID: pulid.MustNew("apl_"), Title: "Cover the move", Status: agent.PlanStatusPending}
	gone := pulid.MustNew("ap_")
	queue := &stubQueue{
		page: &repositories.PendingDecisionsPage{
			Entries: []repositories.PendingDecisionEntry{
				{Kind: repositories.PendingDecisionPlan, ID: plan.ID, CreatedAt: 30},
				{Kind: repositories.PendingDecisionProposal, ID: gone, CreatedAt: 20},
				{Kind: repositories.PendingDecisionProposal, ID: first.ID, CreatedAt: 10},
			},
			HasNextPage: true,
		},
		count: 7,
	}
	svc := &Service{
		l:         zap.NewNop(),
		queue:     queue,
		proposals: &stubProposals{byID: map[pulid.ID]*agent.AgentProposal{first.ID: first}},
		plans:     &stubPlans{byID: map[pulid.ID]*agent.AgentPlan{plan.ID: plan}},
		shadow:    stubShadow{},
	}

	page, err := svc.ListPending(t.Context(), services.ListPendingDecisionsRequest{
		First:             2,
		ToolName:          " assign_move ",
		IncludeTotalCount: true,
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	assert.Equal(t, plan, page.Items[0].Plan)
	assert.Equal(t, first, page.Items[1].Proposal)
	assert.True(t, page.HasNextPage)
	require.NotNil(t, page.TotalCount)
	assert.Equal(t, 7, *page.TotalCount)
	assert.Equal(t, "assign_move", queue.last.ToolName)
	assert.True(t, queue.last.ExcludeShadowDefinitions)

	decoded, err := decodeCursor(page.Items[1].Cursor)
	require.NoError(t, err)
	assert.Equal(t, first.ID, decoded.ID)
	assert.EqualValues(t, 10, decoded.CreatedAt)

	_, err = svc.ListPending(t.Context(), services.ListPendingDecisionsRequest{After: "not-a-cursor"})
	require.Error(t, err)
}

// With the organization in shadow mode nothing can be approved, so the
// queue is empty rather than a list of rows without buttons.
func TestListPending_IsEmptyUnderOrganizationShadow(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(), queue: &stubQueue{page: &repositories.PendingDecisionsPage{}}, shadow: stubShadow{shadow: true}}
	page, err := svc.ListPending(t.Context(), services.ListPendingDecisionsRequest{})
	require.NoError(t, err)
	assert.Empty(t, page.Items)

	summary, err := svc.Summary(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Zero(t, summary.Total)
}
