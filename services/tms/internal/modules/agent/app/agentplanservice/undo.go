package agentplanservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/decisioncommitjobs/decisioncommitcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentapprovalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// defers says whether there is a workflow engine to hold an approval for its
// undo window. Without one a plan approved from a conversation runs at once,
// as it always has.
func (s *Service) defers() bool {
	return s.workflows != nil && s.workflows.Enabled()
}

// approveDeferred approves a plan that runs when the undo window closes. It
// is checked as a decision is, now, and moved to Approving; nothing runs until
// the window closes, when it is checked again against the world as it is then.
func (s *Service) approveDeferred(
	ctx context.Context,
	req *agentplanports.DecideAgentPlanRequest,
	actor *permissionports.RequestActor,
) (*agentplan.AgentPlan, error) {
	prepared, err := s.prepare(ctx, req, actor, agentplan.PlanStatusPending)
	if err != nil {
		return nil, err
	}
	if prepared.shown.stale() {
		return s.claimAndRun(ctx, req, actor, prepared, agentplan.PlanStatusPending)
	}

	now := time.Now()
	commitsAt := decisioncommitcontract.CommitsAt(now)
	plan, err := s.plans.UpdateStatus(ctx, &agentplanports.UpdateAgentPlanStatusRequest{
		ID:              prepared.plan.ID,
		TenantInfo:      req.TenantInfo,
		Status:          agentplan.PlanStatusApproving,
		FromStatus:      agentplan.PlanStatusPending,
		DecidedByUserID: actor.UserID,
		DecidedAt:       now.Unix(),
		CommitsAt:       commitsAt,
		Approval: agentplanports.PlanApproval{
			ReasonCode:    req.ReasonCode,
			Note:          req.Note,
			PreviewDigest: prepared.shown.digest(),
		},
	})
	if err != nil {
		return nil, err
	}

	workflowID := decisioncommitcontract.WorkflowID(plan.ID)
	if _, err = s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                decisioncommitcontract.TaskQueue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_TERMINATE_EXISTING,
		StaticSummary:            "Plan approval undo window",
	}, decisioncommitcontract.ProposalCommitWorkflowName, &agentapprovalports.CommitApprovalRequest{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		DecidedByUserID: actor.UserID,
		WorkflowID:      workflowID,
		PlanID:          plan.ID,
		ReasonCode:      req.ReasonCode,
		Note:            req.Note,
		PreviewDigest:   prepared.shown.digest(),
		CommitsAt:       commitsAt,
	}); err != nil {
		s.l.Error("a plan approval could not be scheduled; taking it back",
			zap.String("plan", plan.ID.String()), zap.Error(err))
		s.reopenPlan(ctx, plan, actor, true)

		return nil, errortypes.NewBusinessError(
			"The approval could not be scheduled, so nothing was approved. Try again",
		)
	}

	s.announce(ctx, plan, actor, agentactivityports.ActivityApproving)

	return plan, nil
}

// CommitApproval runs a plan whose undo window has closed, as the person who
// approved it. A plan undone in the window, or approved again since in a
// later window, is left alone. A plan a record of which moved on in the
// window is refused as approved: nothing runs, the plan fails at that step
// and its conversation is told why. A plan that can no longer be approved as
// it stands for any other reason goes back to waiting on its decider.
func (s *Service) CommitApproval(
	ctx context.Context,
	req *agentapprovalports.CommitApprovalRequest,
) error {
	tenant := req.TenantInfo()
	actor := permissionports.UserActor(tenant)
	ctx = dbscope.WithTenant(ctx, actor.DBTenant())

	plan, err := s.plans.GetByID(ctx, agentplanports.GetAgentPlanByIDRequest{
		ID:         req.PlanID,
		TenantInfo: tenant,
	})
	if err != nil {
		return err
	}
	if plan.Status != agentplan.PlanStatusApproving || plan.CommitsAt == nil ||
		(req.CommitsAt != 0 && *plan.CommitsAt != req.CommitsAt) {
		return nil
	}

	decideReq := &agentplanports.DecideAgentPlanRequest{
		PlanID:        plan.ID,
		Decision:      agentdecision.DecisionAccepted,
		ReasonCode:    stringutils.FirstNonEmpty(req.ReasonCode, plan.ApprovalReasonCode),
		Note:          stringutils.FirstNonEmpty(req.Note, plan.ApprovalNote),
		TenantInfo:    tenant,
		PreviewDigest: stringutils.FirstNonEmpty(req.PreviewDigest, plan.ApprovalPreviewDigest),
	}
	prepared, err := s.prepare(ctx, decideReq, actor, agentplan.PlanStatusApproving)
	if err != nil {
		if agent.IsReviewRequired(err) {
			s.l.Warn("a plan approved in its undo window came to commit without its preview",
				zap.String("plan", plan.ID.String()), zap.Error(err))
			s.reopenPlan(ctx, plan, actor, false)

			return err
		}
		if errortypes.IsBusinessError(err) || errortypes.IsConflictError(err) ||
			errortypes.IsMultiError(err) || errortypes.IsAuthorizationError(err) {
			s.l.Warn("a plan approved in its undo window can no longer go through",
				zap.String("plan", plan.ID.String()), zap.Error(err))
			s.reopenPlan(ctx, plan, actor, false)

			return nil
		}

		return err
	}

	if _, err = s.claimAndRun(
		ctx,
		decideReq,
		actor,
		prepared,
		agentplan.PlanStatusApproving,
	); err != nil {
		if errortypes.IsVersionMismatchError(err) {
			// Undone in the moment between the read and the claim.
			return nil
		}

		return err
	}

	return nil
}

// UndoOwnApproval takes back the actor's approval of a plan while it is in
// its undo window. Once the plan has started to run it is a conflict.
func (s *Service) UndoOwnApproval(
	ctx context.Context,
	req *agentapprovalports.SettleApprovalRequest,
	actor *permissionports.RequestActor,
) error {
	plan, err := s.ownApproving(ctx, req, actor)
	if err != nil {
		return err
	}
	if plan.Status != agentplan.PlanStatusApproving {
		return errAlreadyCommitted()
	}

	if !s.reopenPlan(ctx, plan, actor, true) {
		return errAlreadyCommitted()
	}

	if err = s.workflows.SignalWorkflow(
		ctx,
		decisioncommitcontract.WorkflowID(plan.ID),
		"",
		decisioncommitcontract.UndoSignalName,
		nil,
	); err != nil {
		s.l.Warn("an undo could not reach its plan's commit workflow",
			zap.String("plan", plan.ID.String()), zap.Error(err))
	}

	return nil
}

// CommitOwnApprovalNow ends the actor's undo window on a plan early. A plan
// already running or run is left as it is; one undone is a conflict.
func (s *Service) CommitOwnApprovalNow(
	ctx context.Context,
	req *agentapprovalports.SettleApprovalRequest,
	actor *permissionports.RequestActor,
) error {
	plan, err := s.ownApproving(ctx, req, actor)
	if err != nil {
		return err
	}
	switch plan.Status {
	case agentplan.PlanStatusApproving:
	case agentplan.PlanStatusPending:
		return errortypes.NewConflictError(
			"This approval was undone; approve the plan again to run it",
		)
	default:
		return nil
	}

	workflowID := decisioncommitcontract.WorkflowID(plan.ID)
	if err = s.workflows.SignalWorkflow(
		ctx, workflowID, "", decisioncommitcontract.CommitNowSignalName, nil,
	); err == nil {
		return nil
	}
	commitsAt, _ := typeutils.Deref(plan.CommitsAt)

	s.l.Warn("commit-now could not reach its plan's workflow; committing here",
		zap.String("plan", plan.ID.String()), zap.Error(err))

	return s.CommitApproval(ctx, &agentapprovalports.CommitApprovalRequest{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		DecidedByUserID: actor.UserID,
		WorkflowID:      workflowID,
		PlanID:          plan.ID,
		ReasonCode:      plan.ApprovalReasonCode,
		Note:            plan.ApprovalNote,
		PreviewDigest:   plan.ApprovalPreviewDigest,
		CommitsAt:       commitsAt,
	})
}

// ownApproving reads a plan of the actor's own, with the checks deciding it
// takes, so a person can undo exactly what they could approve.
func (s *Service) ownApproving(
	ctx context.Context,
	req *agentapprovalports.SettleApprovalRequest,
	actor *permissionports.RequestActor,
) (*agentplan.AgentPlan, error) {
	if req.PlanID.IsNil() {
		return nil, errortypes.NewValidationError(
			"planId", errortypes.ErrRequired, "Name the plan whose approval this is",
		)
	}
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a human user can decide on agent plans",
		)
	}
	if s.workflows == nil {
		return nil, errAlreadyCommitted()
	}
	if err := s.AssertOwnPlan(ctx, req.PlanID, req.TenantInfo, actor); err != nil {
		return nil, err
	}

	return s.plans.GetByID(ctx, agentplanports.GetAgentPlanByIDRequest{
		ID:         req.PlanID,
		TenantInfo: req.TenantInfo,
	})
}

// reopenPlan moves a plan from Approving back to Pending: undone by the
// actor, or refused at the close of its window. It reports whether it moved;
// a plan that already started to run did not.
func (s *Service) reopenPlan(
	ctx context.Context,
	plan *agentplan.AgentPlan,
	actor *permissionports.RequestActor,
	undo bool,
) bool {
	tenant := planTenant(plan)
	update := agentplanports.UpdateAgentPlanStatusRequest{
		ID:         plan.ID,
		TenantInfo: tenant,
		Status:     agentplan.PlanStatusPending,
		FromStatus: agentplan.PlanStatusApproving,
		Reopen:     true,
	}
	if undo {
		update.UndoneByUserID = actor.UserID
		update.UndoneAt = timeutils.NowUnix()
	}

	reopened, err := s.plans.UpdateStatus(ctx, &update)
	if err != nil {
		if !errortypes.IsVersionMismatchError(err) {
			s.l.Error("a plan could not be returned to waiting",
				zap.String("plan", plan.ID.String()), zap.Error(err))
		}

		return false
	}

	if undo {
		s.logUndo(reopened, actor)
		s.announce(ctx, reopened, actor, agentactivityports.ActivityUndone)

		return true
	}
	s.announce(ctx, reopened, actor, agentactivityports.ActivityReopened)

	return true
}

func (s *Service) logUndo(plan *agentplan.AgentPlan, actor *permissionports.RequestActor) {
	if s.audit == nil {
		return
	}

	auditActor := actor.AuditActor()
	if err := s.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceAgentProposal,
		ResourceID:     plan.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(plan),
		OrganizationID: plan.OrganizationID,
		BusinessUnitID: plan.BusinessUnitID,
	}, auditservice.WithComment("Plan approval undone before it ran")); err != nil {
		s.l.Error("failed to log an undone plan approval", zap.Error(err))
	}
}

func errAlreadyCommitted() error {
	return errortypes.NewConflictError(
		"This approval has already gone through, so it can no longer be undone here",
	)
}

func planTenant(plan *agentplan.AgentPlan) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: plan.OrganizationID, BuID: plan.BusinessUnitID}
}
