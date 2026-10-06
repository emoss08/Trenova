package agentdecisionservice

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/internal/core/services/proposalexecutor"
	"github.com/emoss08/trenova/internal/core/temporaljobs/decisioncommitjobs"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// windowProposals keeps proposals by id and honours the FromStatus guard the
// way the database does, which is what decides a commit racing an undo.
type windowProposals struct {
	repositories.AgentProposalRepository

	mu       sync.Mutex
	byID     map[pulid.ID]*agent.AgentProposal
	executed []pulid.ID
}

func (p *windowProposals) GetByID(
	_ context.Context,
	req repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	proposal, ok := p.byID[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Proposal not found")
	}
	clone := *proposal

	return &clone, nil
}

func (p *windowProposals) UpdateStatus(
	_ context.Context,
	req repositories.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	proposal := p.byID[req.ID]
	if proposal == nil || (req.FromStatus != "" && proposal.Status != req.FromStatus) {
		return nil, dberror.CreateVersionMismatchError("AgentProposal", req.ID.String())
	}
	proposal.Status = req.Status
	clone := *proposal

	return &clone, nil
}

func (p *windowProposals) RecordExecution(
	_ context.Context,
	req repositories.RecordAgentProposalExecutionRequest,
) (*agent.AgentProposal, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	proposal := p.byID[req.ID]
	proposal.Status = req.Status
	p.executed = append(p.executed, req.ID)
	clone := *proposal

	return &clone, nil
}

func (p *windowProposals) status(id pulid.ID) agent.ProposalStatus {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.byID[id].Status
}

// windowDecisionRows keeps decisions in memory and settles them with the same
// guard the repository's statement has.
type windowDecisionRows struct {
	repositories.AgentDecisionRepository

	mu   sync.Mutex
	rows []*agent.AgentDecision
}

func (d *windowDecisionRows) Create(
	_ context.Context,
	decision *agent.AgentDecision,
) (*agent.AgentDecision, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	decision.ID = pulid.MustNew("adec_")
	decision.CreatedAt = timeutils.NowUnix()
	d.rows = append(d.rows, decision)

	return decision, nil
}

func (d *windowDecisionRows) ListByProposals(
	_ context.Context,
	req repositories.ListAgentDecisionsByProposalsRequest,
) ([]*agent.AgentDecision, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	out := make([]*agent.AgentDecision, 0)
	for i := len(d.rows) - 1; i >= 0; i-- {
		row := d.rows[i]
		if row.UndoneAt != nil || row.ProposalID == nil {
			continue
		}
		for _, id := range req.ProposalIDs {
			if *row.ProposalID == id {
				clone := *row
				out = append(out, &clone)
			}
		}
	}

	return out, nil
}

func (d *windowDecisionRows) ListByCommitWorkflow(
	_ context.Context,
	req repositories.ListAgentDecisionsByCommitWorkflowRequest,
) ([]*agent.AgentDecision, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	out := make([]*agent.AgentDecision, 0)
	for _, row := range d.rows {
		if row.CommitWorkflowID == req.WorkflowID {
			clone := *row
			out = append(out, &clone)
		}
	}

	return out, nil
}

func (d *windowDecisionRows) settle(req repositories.SettleAgentDecisionsRequest, undo bool) []pulid.ID {
	d.mu.Lock()
	defer d.mu.Unlock()

	settled := make([]pulid.ID, 0)
	for _, row := range d.rows {
		for _, id := range req.IDs {
			if row.ID != id || row.CommitsAt == nil || row.CommittedAt != nil || row.UndoneAt != nil {
				continue
			}
			at := req.At
			if undo {
				row.UndoneAt = &at
				by := req.UndoneByUserID
				row.UndoneByUserID = &by
			} else {
				row.CommittedAt = &at
			}
			settled = append(settled, row.ID)
		}
	}

	return settled
}

func (d *windowDecisionRows) MarkCommitted(
	_ context.Context,
	req repositories.SettleAgentDecisionsRequest,
) ([]pulid.ID, error) {
	return d.settle(req, false), nil
}

func (d *windowDecisionRows) MarkUndone(
	_ context.Context,
	req repositories.SettleAgentDecisionsRequest,
) ([]pulid.ID, error) {
	return d.settle(req, true), nil
}

type sentSignal struct {
	workflowID string
	name       string
}

// windowWorkflows records what the service asked of the workflow engine.
type windowWorkflows struct {
	services.WorkflowStarter

	started   []client.StartWorkflowOptions
	payloads  []*services.CommitApprovalRequest
	signals   []sentSignal
	signalErr error
	startErr  error
}

func (w *windowWorkflows) Enabled() bool { return true }

func (w *windowWorkflows) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	args ...any,
) (client.WorkflowRun, error) {
	if w.startErr != nil {
		return nil, w.startErr
	}
	w.started = append(w.started, options)
	w.payloads = append(w.payloads, args[0].(*services.CommitApprovalRequest))

	return nil, nil
}

func (w *windowWorkflows) SignalWorkflow(
	_ context.Context,
	workflowID, _, signalName string,
	_ any,
) error {
	w.signals = append(w.signals, sentSignal{workflowID: workflowID, name: signalName})

	return w.signalErr
}

// windowRuns says every proposal came from the actor's own conversation.
type windowRuns struct {
	repositories.AgentRunRepository

	threadID pulid.ID
}

func (r windowRuns) GetByID(
	_ context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return &agent.AgentRun{
		ID:          req.ID,
		SubjectType: agent.SubjectAssistantThread,
		SubjectID:   r.threadID,
	}, nil
}

type windowThreads struct {
	repositories.ConversationRepository
}

func (windowThreads) GetThread(
	_ context.Context,
	req repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	return &conversation.Thread{ID: req.ID, UserID: req.UserID}, nil
}

type windowActivity struct {
	services.AgentActivityPublisher

	mu      sync.Mutex
	actions []string
}

func (a *windowActivity) ProposalChanged(
	_ context.Context,
	_ *agent.AgentProposal,
	_ services.AuditActor,
	action string,
) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.actions = append(a.actions, action)
}

type windowHarness struct {
	service   *Service
	proposals *windowProposals
	decisions *windowDecisionRows
	workflows *windowWorkflows
	activity  *windowActivity
	actor     *services.RequestActor
	tenant    pagination.TenantInfo
	ids       []pulid.ID
}

func newWindowHarness(t *testing.T, count int) *windowHarness {
	t.Helper()

	actor := &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	actor.PrincipalID = actor.UserID
	tenant := pagination.TenantInfo{OrgID: actor.OrganizationID, BuID: actor.BusinessUnitID}

	proposals := &windowProposals{byID: map[pulid.ID]*agent.AgentProposal{}}
	ids := make([]pulid.ID, 0, count)
	for range count {
		id := pulid.MustNew("ap_")
		proposals.byID[id] = &agent.AgentProposal{
			ID:             id,
			OrganizationID: actor.OrganizationID,
			BusinessUnitID: actor.BusinessUnitID,
			RunID:          pulid.MustNew("ar_"),
			ToolName:       "place_shipment_hold",
			ToolParams:     map[string]any{},
			AutonomyTier:   agent.TierPropose,
			Status:         agent.ProposalStatusPending,
			ExpiresAt:      timeutils.NowUnix() + 3600,
			TargetResource: string(permission.ResourceShipmentMove),
			TargetID:       pulid.MustNew("smv_"),
			TargetVersion:  3,
		}
		ids = append(ids, id)
	}

	decisions := &windowDecisionRows{}
	workflows := &windowWorkflows{}
	activity := &windowActivity{}
	runs := windowRuns{threadID: pulid.MustNew("athr_")}
	executor := proposalexecutor.New(proposalexecutor.Params{
		Logger: zap.NewNop(),
		Tools: &agentruntimetest.StubActionRegistry{Tools: []services.AgentTool{
			&agentruntimetest.StubActionTool{ToolName: "place_shipment_hold"},
		}},
		ProposalRepo: proposals,
		Permissions:  &agentruntimetest.StubPermissions{},
		AuditService: traceAudit{},
	})

	return &windowHarness{
		service: &Service{
			l:            zap.NewNop(),
			decisionRepo: decisions,
			proposalRepo: proposals,
			runRepo:      runs,
			threads:      windowThreads{},
			shadow:       agentshadow.New(agentshadow.Params{Control: traceControls{}, Runs: runs}),
			workflows:    workflows,
			executor:     executor,
			audit:        traceAudit{},
			activity:     activity,
		},
		proposals: proposals,
		decisions: decisions,
		workflows: workflows,
		activity:  activity,
		actor:     actor,
		tenant:    tenant,
		ids:       ids,
	}
}

func (h *windowHarness) approve(t *testing.T) []services.DeferredApproval {
	t.Helper()

	reqs := make([]*services.DecideAgentProposalRequest, 0, len(h.ids))
	for _, id := range h.ids {
		reqs = append(reqs, &services.DecideAgentProposalRequest{
			ProposalID: id,
			Decision:   agent.DecisionAccepted,
			TenantInfo: h.tenant,
		})
	}
	approved, err := h.service.ApproveDeferred(t.Context(), reqs, h.actor)
	require.NoError(t, err)

	return approved
}

func (h *windowHarness) closeWindow(t *testing.T) {
	t.Helper()
	require.NotEmpty(t, h.workflows.payloads, "an approval starts the workflow that holds it")

	payload := h.workflows.payloads[len(h.workflows.payloads)-1]
	require.NoError(t, h.service.CommitApproval(t.Context(), payload))
}

func (h *windowHarness) settle(id pulid.ID) *services.SettleApprovalRequest {
	return &services.SettleApprovalRequest{ProposalID: id, TenantInfo: h.tenant}
}

// An approval from the person's own conversation records the decision and
// nothing else: the proposal waits in Approving, a workflow holds it for the
// window, and the write runs only when that workflow commits it.
func TestApproval_WaitsOutItsWindowAndThenCommits(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	before := time.Now()
	approved := h.approve(t)

	require.Len(t, approved, 1)
	require.NoError(t, approved[0].Err)
	decision := approved[0].Decision
	require.NotNil(t, decision.CommitsAt)
	window := time.Unix(*decision.CommitsAt, 0).Sub(before)
	assert.GreaterOrEqual(t, window, services.ApprovalUndoWindow,
		"the client's ring counts down from the server's clock, never less than the full window")
	assert.Less(t, window, services.ApprovalUndoWindow+2*time.Second)

	id := h.ids[0]
	assert.Equal(t, agent.ProposalStatusApproving, h.proposals.status(id))
	assert.Empty(t, h.proposals.executed, "nothing runs while the approval can still be undone")
	require.Len(t, h.workflows.started, 1)
	assert.Equal(t, decisioncommitjobs.WorkflowID(id), h.workflows.started[0].ID)
	assert.Contains(t, h.activity.actions, services.ActivityApproving)

	h.closeWindow(t)

	assert.Equal(t, []pulid.ID{id}, h.proposals.executed, "the window closing runs the write")
	assert.Equal(t, agent.ProposalStatusExecuted, h.proposals.status(id))
	assert.Contains(t, h.activity.actions, services.ActivityCommitted)

	h.closeWindow(t)
	assert.Len(t, h.proposals.executed, 1, "a retried commit does not run the write twice")
}

// Undo before the window closes puts the proposal back in front of its
// decider, tells the workflow, and leaves the commit nothing to do.
func TestApproval_UndoneInItsWindowNeverRuns(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	h.approve(t)
	id := h.ids[0]

	require.NoError(t, h.service.UndoOwnApproval(t.Context(), h.settle(id), h.actor))

	assert.Equal(t, agent.ProposalStatusPending, h.proposals.status(id))
	require.Len(t, h.workflows.signals, 1)
	assert.Equal(t, decisioncommitjobs.UndoSignalName, h.workflows.signals[0].name)
	assert.Contains(t, h.activity.actions, services.ActivityUndone)

	// A workflow that never heard the undo commits nothing when it wakes.
	h.closeWindow(t)
	assert.Empty(t, h.proposals.executed)
	assert.Equal(t, agent.ProposalStatusPending, h.proposals.status(id))

	// Undone is no decision: the proposal can be approved again.
	h.approve(t)
	assert.Equal(t, agent.ProposalStatusApproving, h.proposals.status(id))
	h.closeWindow(t)
	assert.Equal(t, []pulid.ID{id}, h.proposals.executed)
}

func TestApproval_UndoAfterTheCommitIsAConflict(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	h.approve(t)
	h.closeWindow(t)

	err := h.service.UndoOwnApproval(t.Context(), h.settle(h.ids[0]), h.actor)

	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err), "undo after the commit is a conflict, got %v", err)
	assert.Len(t, h.proposals.executed, 1)
}

// "Do it now" asks the workflow to stop waiting; the commit is still the
// workflow's, so it happens once.
func TestApproval_CommitNowSignalsTheWorkflow(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	h.approve(t)

	require.NoError(t, h.service.CommitOwnApprovalNow(t.Context(), h.settle(h.ids[0]), h.actor))

	require.Len(t, h.workflows.signals, 1)
	assert.Equal(t, decisioncommitjobs.CommitNowSignalName, h.workflows.signals[0].name)
	assert.Empty(t, h.proposals.executed, "the workflow commits on the signal, not the request")
}

// A workflow that cannot be reached has ended or never started, so the
// request commits the approval itself rather than leave it waiting.
func TestApproval_CommitNowCommitsHereWhenTheWorkflowIsGone(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	h.approve(t)
	h.workflows.signalErr = errors.New("workflow not found")

	require.NoError(t, h.service.CommitOwnApprovalNow(t.Context(), h.settle(h.ids[0]), h.actor))

	assert.Equal(t, []pulid.ID{h.ids[0]}, h.proposals.executed)
}

// A batch is one window and one workflow: undoing any of it undoes all of it.
func TestApproval_ABatchIsOneWindow(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 3)
	approved := h.approve(t)

	require.Len(t, approved, 3)
	require.Len(t, h.workflows.started, 1, "one workflow holds the whole batch")
	for _, id := range h.ids {
		assert.Equal(t, agent.ProposalStatusApproving, h.proposals.status(id))
	}

	require.NoError(t, h.service.UndoOwnApproval(t.Context(), h.settle(h.ids[1]), h.actor))
	for _, id := range h.ids {
		assert.Equal(t, agent.ProposalStatusPending, h.proposals.status(id))
	}
	h.closeWindow(t)
	assert.Empty(t, h.proposals.executed)
}

// Without a workflow to hold it an approval must not wait in Approving for a
// commit that will never come.
func TestApproval_ThatCannotBeScheduledIsTakenBack(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	h.workflows.startErr = errors.New("temporal unavailable")

	_, err := h.service.ApproveDeferred(t.Context(), []*services.DecideAgentProposalRequest{{
		ProposalID: h.ids[0],
		Decision:   agent.DecisionAccepted,
		TenantInfo: h.tenant,
	}}, h.actor)

	require.Error(t, err)
	assert.Equal(t, agent.ProposalStatusPending, h.proposals.status(h.ids[0]))
	decisions, listErr := h.decisions.ListByProposals(t.Context(),
		repositories.ListAgentDecisionsByProposalsRequest{ProposalIDs: h.ids, TenantInfo: h.tenant})
	require.NoError(t, listErr)
	assert.Empty(t, decisions, "the decision is taken back with it")
}

// A proposal in its undo window is neither waiting on anyone nor decided
// for good: it cannot be decided again, and the expiry sweep leaves it alone.
func TestApproval_ApprovingIsNotDecidableAndDoesNotExpire(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		Status:    agent.ProposalStatusApproving,
		ExpiresAt: timeutils.NowUnix() - 60,
	}

	require.Error(t, decidable(proposal))
	assert.False(t, agent.ProposalStatusApproving.Expirable())
	assert.True(t, agent.ProposalStatusPending.Expirable())
	assert.False(t, agent.PlanStatusApproving.Expirable())
	assert.False(t, agent.PlanStatusApproving.Decidable())
}
