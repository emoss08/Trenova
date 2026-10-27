package agentdecisionservice

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalexecutor"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/decisioncommitjobs/decisioncommitcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	agentports.AgentProposalRepository

	mu       sync.Mutex
	byID     map[pulid.ID]*agent.AgentProposal
	executed []pulid.ID
}

func (p *windowProposals) GetByID(
	_ context.Context,
	req agentports.GetAgentProposalByIDRequest,
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
	req agentports.UpdateAgentProposalStatusRequest,
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
	req agentports.RecordAgentProposalExecutionRequest,
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
	agentports.AgentDecisionRepository

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
	req agentports.ListAgentDecisionsByProposalsRequest,
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
	req agentports.ListAgentDecisionsByCommitWorkflowRequest,
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

func (d *windowDecisionRows) settle(req agentports.SettleAgentDecisionsRequest, undo bool) []pulid.ID {
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
	req agentports.SettleAgentDecisionsRequest,
) ([]pulid.ID, error) {
	return d.settle(req, false), nil
}

func (d *windowDecisionRows) MarkUndone(
	_ context.Context,
	req agentports.SettleAgentDecisionsRequest,
) ([]pulid.ID, error) {
	return d.settle(req, true), nil
}

type sentSignal struct {
	workflowID string
	name       string
}

// windowWorkflows records what the service asked of the workflow engine.
type windowWorkflows struct {
	kernelports.WorkflowStarter

	started   []client.StartWorkflowOptions
	payloads  []*agentports.CommitApprovalRequest
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
	w.payloads = append(w.payloads, args[0].(*agentports.CommitApprovalRequest))

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
	agentports.AgentRunRepository

	threadID pulid.ID
}

func (r windowRuns) GetByID(
	_ context.Context,
	req agentports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return &agent.AgentRun{
		ID:          req.ID,
		SubjectType: agent.SubjectAssistantThread,
		SubjectID:   r.threadID,
	}, nil
}

type windowThreads struct {
	agentports.ConversationRepository
}

func (windowThreads) GetThread(
	_ context.Context,
	req agentports.GetThreadRequest,
) (*conversation.Thread, error) {
	return &conversation.Thread{ID: req.ID, UserID: req.UserID}, nil
}

type windowActivity struct {
	agentports.AgentActivityPublisher

	mu      sync.Mutex
	actions []string
}

func (a *windowActivity) ProposalChanged(
	_ context.Context,
	_ *agent.AgentProposal,
	_ kernelports.AuditActor,
	action string,
) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.actions = append(a.actions, action)
}

// windowControls is the organization's pause, switched mid-window by a test.
type windowControls struct {
	tenantports.AgentControlRepository

	mu     sync.Mutex
	paused bool
}

func (c *windowControls) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return &tenant.AgentControl{ShadowMode: c.paused}, nil
}

func (c *windowControls) pause() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = true
}

type windowHarness struct {
	controls  *windowControls
	service   *Service
	proposals *windowProposals
	decisions *windowDecisionRows
	workflows *windowWorkflows
	activity  *windowActivity
	actor     *permissionports.RequestActor
	tenant    pagination.TenantInfo
	ids       []pulid.ID
}

func newWindowHarness(t *testing.T, count int) *windowHarness {
	t.Helper()

	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
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
			ToolName:       "shipment_place_hold",
			ToolParams:     map[string]any{},
			AutonomyTier:   agent.TierPropose,
			EgressClass:    agent.EgressInternal,
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
	controls := &windowControls{}
	executor := proposalexecutor.New(proposalexecutor.Params{
		Logger: zap.NewNop(),
		Tools: &agenttest.StubActionRegistry{Tools: []agentports.AgentTool{
			&agenttest.StubActionTool{ToolName: "shipment_place_hold"},
		}},
		ProposalRepo: proposals,
		Permissions:  &agenttest.StubPermissions{},
		AuditService: traceAudit{},
	})

	return &windowHarness{
		service: &Service{
			l:            zap.NewNop(),
			decisionRepo: decisions,
			proposalRepo: proposals,
			runRepo:      runs,
			threads:      windowThreads{},
			shadow:       agentshadow.New(agentshadow.Params{Control: controls, Runs: runs}),
			workflows:    workflows,
			executor:     executor,
			audit:        traceAudit{},
			activity:     activity,
		},
		controls:  controls,
		proposals: proposals,
		decisions: decisions,
		workflows: workflows,
		activity:  activity,
		actor:     actor,
		tenant:    tenant,
		ids:       ids,
	}
}

func (h *windowHarness) approve(t *testing.T) []agentports.DeferredApproval {
	t.Helper()

	reqs := make([]*agentports.DecideAgentProposalRequest, 0, len(h.ids))
	for _, id := range h.ids {
		reqs = append(reqs, &agentports.DecideAgentProposalRequest{
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

func (h *windowHarness) settle(id pulid.ID) *agentports.SettleApprovalRequest {
	return &agentports.SettleApprovalRequest{ProposalID: id, TenantInfo: h.tenant}
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
	assert.GreaterOrEqual(t, window, agentports.ApprovalUndoWindow,
		"the client's ring counts down from the server's clock, never less than the full window")
	assert.Less(t, window, agentports.ApprovalUndoWindow+2*time.Second)

	id := h.ids[0]
	assert.Equal(t, agent.ProposalStatusApproving, h.proposals.status(id))
	assert.Empty(t, h.proposals.executed, "nothing runs while the approval can still be undone")
	require.Len(t, h.workflows.started, 1)
	assert.Equal(t, decisioncommitcontract.WorkflowID(id), h.workflows.started[0].ID)
	assert.Contains(t, h.activity.actions, agentports.ActivityApproving)

	h.closeWindow(t)

	assert.Equal(t, []pulid.ID{id}, h.proposals.executed, "the window closing runs the write")
	assert.Equal(t, agent.ProposalStatusExecuted, h.proposals.status(id))
	assert.Contains(t, h.activity.actions, agentports.ActivityCommitted)

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
	assert.Equal(t, decisioncommitcontract.UndoSignalName, h.workflows.signals[0].name)
	assert.Contains(t, h.activity.actions, agentports.ActivityUndone)

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
	assert.Equal(t, decisioncommitcontract.CommitNowSignalName, h.workflows.signals[0].name)
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

	_, err := h.service.ApproveDeferred(t.Context(), []*agentports.DecideAgentProposalRequest{{
		ProposalID: h.ids[0],
		Decision:   agent.DecisionAccepted,
		TenantInfo: h.tenant,
	}}, h.actor)

	require.Error(t, err)
	assert.Equal(t, agent.ProposalStatusPending, h.proposals.status(h.ids[0]))
	decisions, listErr := h.decisions.ListByProposals(t.Context(),
		agentports.ListAgentDecisionsByProposalsRequest{ProposalIDs: h.ids, TenantInfo: h.tenant})
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

func (h *windowHarness) decisionOf(t *testing.T, id pulid.ID) *agent.AgentDecision {
	t.Helper()

	h.decisions.mu.Lock()
	defer h.decisions.mu.Unlock()
	for i := len(h.decisions.rows) - 1; i >= 0; i-- {
		row := h.decisions.rows[i]
		if row.ProposalID != nil && *row.ProposalID == id {
			clone := *row
			return &clone
		}
	}
	require.FailNow(t, "no decision for the proposal")

	return nil
}

/*
Nothing runs while a switch holds it. An approval made while agents were
live waits out its undo window; if a person pauses every agent, or turns the
agent's shadow switch on, before the window closes, the commit must not run
the write. The approval is taken back by the system and the proposal waits
on the person again, as it would after an undo.
*/
func TestApproval_CommittedAfterAPauseGoesBackToWaitingAndNeverRuns(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	h.approve(t)
	id := h.ids[0]
	h.controls.pause()

	h.closeWindow(t)

	assert.Empty(t, h.proposals.executed, "a held approval never runs")
	assert.Equal(t, agent.ProposalStatusPending, h.proposals.status(id))
	decision := h.decisionOf(t, id)
	require.NotNil(t, decision.UndoneAt, "the approval is taken back")
	assert.Nil(t, decision.CommittedAt)
	assert.Contains(t, h.activity.actions, agentports.ActivityUndone)

	h.closeWindow(t)
	assert.Empty(t, h.proposals.executed, "a retried commit still runs nothing")
}

func TestApproval_CommittedNowAfterAPauseNeverRuns(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	h.approve(t)
	h.workflows.signalErr = errors.New("workflow not found")
	h.controls.pause()

	require.NoError(t, h.service.CommitOwnApprovalNow(t.Context(), h.settle(h.ids[0]), h.actor))

	assert.Empty(t, h.proposals.executed)
	assert.Equal(t, agent.ProposalStatusPending, h.proposals.status(h.ids[0]))
}

// A commit that claimed its decision and stopped before running it is
// finished by the next attempt; a pause between the two still stops it.
func TestApproval_FinishedAfterAPauseByALaterAttemptNeverRuns(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	h.approve(t)
	id := h.ids[0]
	decision := h.decisionOf(t, id)
	_, err := h.decisions.MarkCommitted(t.Context(), agentports.SettleAgentDecisionsRequest{
		IDs:        []pulid.ID{decision.ID},
		TenantInfo: h.tenant,
		At:         timeutils.NowUnix(),
	})
	require.NoError(t, err)
	h.controls.pause()

	h.closeWindow(t)

	assert.Empty(t, h.proposals.executed)
	assert.Equal(t, agent.ProposalStatusPending, h.proposals.status(id))
}

// A decision on a proposal a switch holds is refused by the service, before
// anything is recorded or scheduled, and the refusal names the switch.
func TestDecision_OnAHeldProposalIsRefusedAndRecordsNothing(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	h.controls.pause()

	approved, err := h.service.ApproveDeferred(t.Context(), []*agentports.DecideAgentProposalRequest{{
		ProposalID: h.ids[0],
		Decision:   agent.DecisionAccepted,
		TenantInfo: h.tenant,
	}}, h.actor)
	if err == nil {
		require.Len(t, approved, 1)
		err = approved[0].Err
	}

	require.Error(t, err)
	assert.Contains(t, err.Error(), "while all agents are paused")
	assert.Equal(t, agent.ProposalStatusPending, h.proposals.status(h.ids[0]))
	assert.Empty(t, h.workflows.started, "nothing is scheduled")
	assert.Empty(t, h.decisions.rows, "nothing is recorded")
	assert.Empty(t, h.proposals.executed)

	_, err = h.service.DecideWithOutcome(t.Context(), &agentports.DecideAgentProposalRequest{
		ProposalID: h.ids[0],
		Decision:   agent.DecisionRejected,
		ReasonCode: "not now",
		TenantInfo: h.tenant,
	}, h.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "while all agents are paused")
	assert.Empty(t, h.decisions.rows)
}

func TestApproval_AStaleApprovalIsRefusedAtOnceAndNeverHeldForAWindow(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 2)
	stale := h.ids[0]
	h.service.previews = &stalePreviews{stale: stale}
	followUps := &recordedFollowUps{}
	h.service.followUps = followUps

	approved := h.approve(t)
	require.Len(t, approved, 2)

	require.NoError(t, approved[0].Err)
	require.NotNil(t, approved[0].Decision, "the refused approval is recorded")
	assert.Nil(t, approved[0].Decision.CommitsAt, "it waits on no window")
	require.ErrorIs(t, approved[0].ExecutionError, agent.ErrTargetChanged)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, h.proposals.status(stale))
	require.Len(t, followUps.requests, 1, "the refusal is reported as it is made")
	assert.Equal(t, stale, followUps.requests[0].ProposalID)

	require.NoError(t, approved[1].Err)
	require.NoError(t, approved[1].ExecutionError)
	require.NotNil(t, approved[1].Decision.CommitsAt, "the other approval still waits out its window")
	assert.Equal(t, agent.ProposalStatusApproving, h.proposals.status(h.ids[1]))
	require.Len(t, h.workflows.payloads, 1)

	require.NoError(t, h.service.CommitOwnApprovalNow(t.Context(), h.settle(stale), h.actor),
		"committing a refused approval now leaves it as it is")
	assert.Equal(t, agent.ProposalStatusExecutionFailed, h.proposals.status(stale))
	assert.Len(t, followUps.requests, 1)
	require.Error(t, h.service.UndoOwnApproval(t.Context(), h.settle(stale), h.actor),
		"a refused approval has nothing left to undo")

	h.closeWindow(t)
	assert.Equal(t, agent.ProposalStatusExecuted, h.proposals.status(h.ids[1]))
	assert.Equal(t, agent.ProposalStatusExecutionFailed, h.proposals.status(stale),
		"closing the window does not touch the refused approval")
}

func TestDecideOwn_AStaleApprovalTellsItsDeciderAtOnceThatNothingChanged(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	stale := h.ids[0]
	h.service.previews = &stalePreviews{stale: stale}
	h.service.followUps = &recordedFollowUps{}

	outcome, err := h.service.DecideOwn(t.Context(), &agentports.DecideAgentProposalRequest{
		ProposalID: stale,
		Decision:   agent.DecisionAccepted,
		TenantInfo: h.tenant,
	}, h.actor)
	require.NoError(t, err, "the refusal is the decision's outcome, not an error")

	result := outcome.Result(stale)
	assert.Equal(t, stale, result.ProposalID)
	assert.True(t, result.Refused)
	assert.False(t, result.Executed)
	assert.Contains(t, result.Error,
		"was at version 3 when this was proposed and is at version 4 now, so nothing was changed")
	require.NotNil(t, result.Decision, "the approval is recorded")
	assert.Nil(t, result.Decision.CommitsAt, "it waits on no undo window")
	assert.Equal(t, agent.ProposalStatusExecutionFailed, h.proposals.status(stale))
	assert.Empty(t, h.workflows.payloads)
}

func TestDecideOwn_AnApprovalInItsUndoWindowHasNotRunAndIsNotRefused(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	id := h.ids[0]

	outcome, err := h.service.DecideOwn(t.Context(), &agentports.DecideAgentProposalRequest{
		ProposalID: id,
		Decision:   agent.DecisionAccepted,
		TenantInfo: h.tenant,
	}, h.actor)
	require.NoError(t, err)

	result := outcome.Result(id)
	assert.False(t, result.Refused)
	assert.False(t, result.Executed, "nothing runs until the window closes")
	assert.Empty(t, result.Error)
	require.NotNil(t, result.Decision)
	assert.NotNil(t, result.Decision.CommitsAt)
	assert.Equal(t, agent.ProposalStatusApproving, h.proposals.status(id))
}

type stalePreviews struct {
	agentports.ProposalPreviewService

	stale pulid.ID
}

func (p *stalePreviews) ForProposal(
	_ context.Context,
	req *agentports.ProposalPreviewRequest,
) (*agent.ProposalPreview, error) {
	current := int64(3)
	if req.Proposal.ID == p.stale {
		current = 4
	}

	return &agent.ProposalPreview{
		Schema:        agent.PreviewSchemaVersion,
		ProposalID:    req.Proposal.ID,
		Tool:          req.Proposal.ToolName,
		Coverage:      agent.PreviewCoverageFull,
		TargetVersion: &current,
		Staleness:     &agent.PreviewStaleness{Pinned: true, ProposedVersion: 3, CurrentVersion: current},
	}, nil
}
