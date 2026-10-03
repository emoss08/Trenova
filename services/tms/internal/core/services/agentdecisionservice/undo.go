package agentdecisionservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/decisioncommitjobs"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// undoWindow is the wait an approval from a person's own conversation goes
// through before it commits: when, and which workflow commits it.
type undoWindow struct {
	workflowID string
	commitsAt  int64
}

func newUndoWindow(key pulid.ID, now time.Time) *undoWindow {
	return &undoWindow{
		workflowID: decisioncommitjobs.WorkflowID(key),
		commitsAt:  decisioncommitjobs.CommitsAt(now),
	}
}

// Defers says whether an approval waits out the undo window. Only an
// approval does, and only when there is a workflow engine to hold it; without
// one it goes through at once, as it always has.
func (s *Service) Defers(decision agent.DecisionType) bool {
	if decision != agent.DecisionAccepted && decision != agent.DecisionModified {
		return false
	}

	return s.workflows != nil && s.workflows.Enabled()
}

// ApproveDeferred records approvals that commit when the undo window closes,
// all in one window held by one workflow. Each proposal is checked and
// recorded on its own, and one refused does not stop the rest. Nothing is
// written to any record until the window closes.
//
// If the workflow cannot be started nothing is left waiting on it: every
// approval recorded here is taken back and the call fails.
func (s *Service) ApproveDeferred(
	ctx context.Context,
	reqs []*services.DecideAgentProposalRequest,
	actor *services.RequestActor,
) ([]services.DeferredApproval, error) {
	results := make([]services.DeferredApproval, len(reqs))
	if len(reqs) == 0 {
		return results, nil
	}

	window := newUndoWindow(reqs[0].ProposalID, time.Now())
	recordedProposals := make([]*agent.AgentProposal, 0, len(reqs))
	recordedDecisions := make([]pulid.ID, 0, len(reqs))
	for i, req := range reqs {
		results[i].ProposalID = req.ProposalID
		_, rec, span, err := s.record(ctx, req, actor, window)
		if span != nil {
			span.End()
		}
		if err != nil {
			results[i].Err = err

			continue
		}
		results[i].Decision = rec.decision
		recordedProposals = append(recordedProposals, rec.proposal)
		recordedDecisions = append(recordedDecisions, rec.decision.ID)
	}
	if len(recordedDecisions) == 0 {
		return results, nil
	}

	tenant := reqs[0].TenantInfo
	if err := s.startCommit(ctx, window, tenant, actor); err != nil {
		s.l.Error("an approval could not be scheduled; taking it back",
			zap.String("workflow", window.workflowID), zap.Error(err))
		s.takeBack(ctx, tenant, recordedDecisions, recordedProposals, actor)

		return nil, errortypes.NewBusinessError(
			"The approval could not be scheduled, so nothing was approved. Try again",
		)
	}

	for _, proposal := range recordedProposals {
		s.announce(ctx, proposal, tenant, actor.AuditActor(), services.ActivityApproving)
	}

	return results, nil
}

func (s *Service) startCommit(
	ctx context.Context,
	window *undoWindow,
	tenant pagination.TenantInfo,
	actor *services.RequestActor,
) error {
	_, err := s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:        window.workflowID,
		TaskQueue: decisioncommitjobs.TaskQueue,
		// The same proposal approved again after an undo reuses the id. A
		// wait from the earlier approval that never heard its undo is ended
		// here; it would find nothing of its own to commit anyway.
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_TERMINATE_EXISTING,
		StaticSummary:            "Approval undo window",
	}, decisioncommitjobs.ProposalCommitWorkflowName, &services.CommitApprovalRequest{
		OrganizationID:  tenant.OrgID,
		BusinessUnitID:  tenant.BuID,
		DecidedByUserID: actor.UserID,
		WorkflowID:      window.workflowID,
		CommitsAt:       window.commitsAt,
	})

	return err
}

// takeBack returns approvals that never got a workflow to waiting on their
// decider, as if they had been undone at once.
func (s *Service) takeBack(
	ctx context.Context,
	tenant pagination.TenantInfo,
	decisions []pulid.ID,
	proposals []*agent.AgentProposal,
	actor *services.RequestActor,
) {
	if _, err := s.decisionRepo.MarkUndone(ctx, repositories.SettleAgentDecisionsRequest{
		IDs:            decisions,
		TenantInfo:     tenant,
		At:             timeutils.NowUnix(),
		UndoneByUserID: actor.UserID,
	}); err != nil {
		s.l.Error("an unscheduled approval could not be taken back", zap.Error(err))
	}
	for _, proposal := range proposals {
		s.reopen(ctx, tenant, proposal.ID)
	}
}

// reopen moves a proposal from Approving back to Pending.
func (s *Service) reopen(ctx context.Context, tenant pagination.TenantInfo, proposalID pulid.ID) {
	if _, err := s.proposalRepo.UpdateStatus(ctx, repositories.UpdateAgentProposalStatusRequest{
		ID:         proposalID,
		Status:     agent.ProposalStatusPending,
		FromStatus: agent.ProposalStatusApproving,
		TenantInfo: tenant,
	}); err != nil {
		s.l.Error("an undone approval's proposal could not be reopened",
			zap.String("proposal", proposalID.String()), zap.Error(err))
	}
}

// CommitApproval carries out the approvals one window held, as the person
// who made them, once the window has closed.
//
// It claims every decision still in the window in one statement, so an undo
// racing it takes all of them or none. A decision already claimed whose
// proposal is still Approving is one an earlier attempt claimed and did not
// finish; it is finished now, and the guarded status change keeps it from
// running twice.
func (s *Service) CommitApproval(ctx context.Context, req *services.CommitApprovalRequest) error {
	tenant := req.TenantInfo()
	actor := services.UserActor(tenant)
	ctx = dbscope.WithTenant(ctx, actor.DBTenant())

	group, err := s.windowDecisions(ctx, tenant, req.WorkflowID, req.CommitsAt)
	if err != nil {
		return err
	}

	open := make([]pulid.ID, 0, len(group))
	for _, decision := range group {
		if decision.UndoneAt == nil && decision.CommittedAt == nil {
			open = append(open, decision.ID)
		}
	}
	claimed, err := s.decisionRepo.MarkCommitted(ctx, repositories.SettleAgentDecisionsRequest{
		IDs:        open,
		TenantInfo: tenant,
		At:         timeutils.NowUnix(),
	})
	if err != nil {
		return err
	}
	ours := make(map[pulid.ID]struct{}, len(claimed))
	for _, id := range claimed {
		ours[id] = struct{}{}
	}

	for _, decision := range group {
		_, justClaimed := ours[decision.ID]
		if !justClaimed && (decision.CommittedAt == nil || decision.UndoneAt != nil) {
			continue
		}
		s.commitOne(ctx, tenant, decision, actor)
	}

	return nil
}

// windowDecisions reads the decisions one window holds. A workflow id is
// reused when a proposal is approved again after an undo, so the window's
// close time tells this approval's decisions from the earlier one's.
func (s *Service) windowDecisions(
	ctx context.Context,
	tenant pagination.TenantInfo,
	workflowID string,
	commitsAt int64,
) ([]*agent.AgentDecision, error) {
	all, err := s.decisionRepo.ListByCommitWorkflow(ctx,
		repositories.ListAgentDecisionsByCommitWorkflowRequest{
			WorkflowID: workflowID,
			TenantInfo: tenant,
		})
	if err != nil {
		return nil, err
	}

	group := make([]*agent.AgentDecision, 0, len(all))
	for _, decision := range all {
		if decision.CommitsAt != nil && (commitsAt == 0 || *decision.CommitsAt == commitsAt) {
			group = append(group, decision)
		}
	}

	return group, nil
}

// commitOne moves one proposal out of Approving into what was decided and
// does what the decision decided. The write runs against the version the
// proposal pinned, so a record that moved on in the window is refused, as it
// would have been had it moved before the click.
func (s *Service) commitOne(
	ctx context.Context,
	tenant pagination.TenantInfo,
	decision *agent.AgentDecision,
	actor *services.RequestActor,
) {
	if decision.ProposalID == nil {
		return
	}

	if _, err := s.proposalRepo.UpdateStatus(ctx, repositories.UpdateAgentProposalStatusRequest{
		ID:         *decision.ProposalID,
		Status:     proposalStatusFor(decision.Decision),
		FromStatus: agent.ProposalStatusApproving,
		TenantInfo: tenant,
	}); err != nil {
		if errortypes.IsVersionMismatchError(err) {
			// Already carried out, by an attempt that got this far.
			return
		}
		s.l.Error("a committed approval's proposal could not be moved on",
			zap.String("proposal", decision.ProposalID.String()), zap.Error(err))

		return
	}

	proposal, err := s.proposalRepo.GetByID(ctx, repositories.GetAgentProposalByIDRequest{
		ID:         *decision.ProposalID,
		TenantInfo: &tenant,
	})
	if err != nil {
		s.l.Error("a committed approval's proposal could not be read",
			zap.String("proposal", decision.ProposalID.String()), zap.Error(err))

		return
	}

	s.carryOut(ctx, &recorded{
		proposal: proposal,
		decision: decision,
		req: &services.DecideAgentProposalRequest{
			ProposalID:    proposal.ID,
			Decision:      decision.Decision,
			Modifications: decision.Modifications,
			ReasonCode:    decision.ReasonCode,
			Note:          decision.Note,
			TenantInfo:    tenant,
			PreviewDigest: decision.PreviewDigest,
		},
	}, actor)
}

// UndoOwnApproval takes back an approval of the actor's own while it is in
// its undo window: the proposal, and every other one approved with it in one
// batch, waits on them again. Once the window has closed it is a conflict.
func (s *Service) UndoOwnApproval(
	ctx context.Context,
	req *services.SettleApprovalRequest,
	actor *services.RequestActor,
) error {
	window, err := s.ownWindow(ctx, req, actor)
	if err != nil {
		return err
	}

	open := make([]pulid.ID, 0, len(window))
	for _, decision := range window {
		if decision.UndoneAt == nil && decision.CommittedAt == nil {
			open = append(open, decision.ID)
		}
	}
	undone, err := s.decisionRepo.MarkUndone(ctx, repositories.SettleAgentDecisionsRequest{
		IDs:            open,
		TenantInfo:     req.TenantInfo,
		At:             timeutils.NowUnix(),
		UndoneByUserID: actor.UserID,
	})
	if err != nil {
		return err
	}
	if len(undone) == 0 {
		return errAlreadyCommitted()
	}

	if err = s.workflows.SignalWorkflow(
		ctx, window[0].CommitWorkflowID, "", decisioncommitjobs.UndoSignalName, nil,
	); err != nil {
		// The commit claims only what is still in the window, so a wait that
		// never hears this finds nothing to do when it closes.
		s.l.Warn("an undo could not reach its commit workflow",
			zap.String("workflow", window[0].CommitWorkflowID), zap.Error(err))
	}

	byID := make(map[pulid.ID]*agent.AgentDecision, len(window))
	for _, decision := range window {
		byID[decision.ID] = decision
	}
	auditActor := actor.AuditActor()
	for _, id := range undone {
		decision := byID[id]
		if decision == nil || decision.ProposalID == nil {
			continue
		}
		s.reopen(ctx, req.TenantInfo, *decision.ProposalID)
		s.logUndo(decision, auditActor)
		if proposal, getErr := s.proposalRepo.GetByID(ctx, repositories.GetAgentProposalByIDRequest{
			ID:         *decision.ProposalID,
			TenantInfo: &req.TenantInfo,
		}); getErr == nil {
			s.announce(ctx, proposal, req.TenantInfo, auditActor, services.ActivityUndone)
		}
	}

	return nil
}

// CommitOwnApprovalNow ends the actor's undo window early. An approval that
// already went through is left as it is; one that was undone is a conflict.
func (s *Service) CommitOwnApprovalNow(
	ctx context.Context,
	req *services.SettleApprovalRequest,
	actor *services.RequestActor,
) error {
	window, err := s.ownWindow(ctx, req, actor)
	if err != nil {
		return err
	}

	open := false
	for _, decision := range window {
		if decision.UndoneAt == nil && decision.CommittedAt == nil {
			open = true

			break
		}
	}
	if !open {
		return nil
	}

	workflowID := window[0].CommitWorkflowID
	if err = s.workflows.SignalWorkflow(
		ctx, workflowID, "", decisioncommitjobs.CommitNowSignalName, nil,
	); err == nil {
		return nil
	}

	// A wait that cannot be reached has ended or never started; the commit is
	// made here instead, and claims only what is still in the window.
	s.l.Warn("commit-now could not reach its workflow; committing here",
		zap.String("workflow", workflowID), zap.Error(err))

	return s.CommitApproval(ctx, &services.CommitApprovalRequest{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		DecidedByUserID: window[0].DecidedByUserID,
		WorkflowID:      workflowID,
		CommitsAt:       *window[0].CommitsAt,
	})
}

// ownWindow finds the undo window a proposal of the actor's own is in: the
// latest decision on it and every decision made with it. The checks are the
// ones deciding takes, so a person can undo exactly what they could approve.
func (s *Service) ownWindow(
	ctx context.Context,
	req *services.SettleApprovalRequest,
	actor *services.RequestActor,
) ([]*agent.AgentDecision, error) {
	if req.ProposalID.IsNil() {
		return nil, errortypes.NewValidationError(
			"proposalId", errortypes.ErrRequired, "Name the proposal whose approval this is",
		)
	}
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a human user can decide on agent proposals",
		)
	}
	if err := s.AssertOwnProposal(ctx, req.ProposalID, req.TenantInfo, actor); err != nil {
		return nil, err
	}

	decisions, err := s.decisionRepo.ListByProposals(ctx, repositories.ListAgentDecisionsByProposalsRequest{
		ProposalIDs: []pulid.ID{req.ProposalID},
		TenantInfo:  req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if len(decisions) == 0 {
		return nil, errortypes.NewConflictError(
			"This change is not approved, so there is nothing to undo",
		)
	}

	latest := decisions[0]
	if latest.CommitWorkflowID == "" || latest.CommitsAt == nil {
		return nil, errAlreadyCommitted()
	}

	window, err := s.windowDecisions(ctx, req.TenantInfo, latest.CommitWorkflowID, *latest.CommitsAt)
	if err != nil {
		return nil, err
	}
	if len(window) == 0 {
		return nil, errAlreadyCommitted()
	}

	return window, nil
}

func errAlreadyCommitted() error {
	return errortypes.NewConflictError(
		"This approval has already gone through, so it can no longer be undone here",
	)
}

func (s *Service) logUndo(decision *agent.AgentDecision, actor services.AuditActor) {
	if s.audit == nil || decision.ProposalID == nil {
		return
	}

	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentProposal,
		ResourceID:     decision.ProposalID.String(),
		Operation:      permission.OpUpdate,
		UserID:         actor.UserID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		APIKeyID:       actor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(decision),
		OrganizationID: decision.OrganizationID,
		BusinessUnitID: decision.BusinessUnitID,
	}, auditservice.WithComment("Approval undone before it went through")); err != nil {
		s.l.Error("failed to log an undone approval", zap.Error(err))
	}
}
