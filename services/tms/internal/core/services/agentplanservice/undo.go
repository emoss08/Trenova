package agentplanservice

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
	"github.com/emoss08/trenova/shared/timeutils"
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
	req *services.DecideAgentPlanRequest,
	actor *services.RequestActor,
) (*agent.AgentPlan, error) {
	prepared, err := s.prepare(ctx, req, actor, agent.PlanStatusPending)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	commitsAt := decisioncommitjobs.CommitsAt(now)
	plan, err := s.plans.UpdateStatus(ctx, repositories.UpdateAgentPlanStatusRequest{
		ID:              prepared.plan.ID,
		TenantInfo:      req.TenantInfo,
		Status:          agent.PlanStatusApproving,
		FromStatus:      agent.PlanStatusPending,
		DecidedByUserID: actor.UserID,
		DecidedAt:       now.Unix(),
		CommitsAt:       commitsAt,
	})
	if err != nil {
		return nil, err
	}

	workflowID := decisioncommitjobs.WorkflowID(plan.ID)
	if _, err = s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                decisioncommitjobs.TaskQueue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_TERMINATE_EXISTING,
		StaticSummary:            "Plan approval undo window",
	}, decisioncommitjobs.ProposalCommitWorkflowName, &services.CommitApprovalRequest{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		DecidedByUserID: actor.UserID,
		WorkflowID:      workflowID,
		PlanID:          plan.ID,
		ReasonCode:      req.ReasonCode,
		Note:            req.Note,
		PreviewDigest:   req.PreviewDigest,
		CommitsAt:       commitsAt,
	}); err != nil {
		s.l.Error("a plan approval could not be scheduled; taking it back",
			zap.String("plan", plan.ID.String()), zap.Error(err))
		s.reopenPlan(ctx, plan, actor, true)

		return nil, errortypes.NewBusinessError(
			"The approval could not be scheduled, so nothing was approved. Try again",
		)
	}

	s.announce(ctx, plan, actor, services.ActivityApproving)

	return plan, nil
}

// CommitApproval runs a plan whose undo window has closed, as the person who
// approved it. A plan undone in the window, or approved again since in a
// later window, is left alone. A plan that can no longer be approved as it
// stands, because a record it changes moved on in the window, goes back to
// waiting on its decider instead of running.
func (s *Service) CommitApproval(ctx context.Context, req *services.CommitApprovalRequest) error {
	tenant := req.TenantInfo()
	actor := services.UserActor(tenant)
	ctx = dbscope.WithTenant(ctx, actor.DBTenant())

	plan, err := s.plans.GetByID(ctx, repositories.GetAgentPlanByIDRequest{
		ID:         req.PlanID,
		TenantInfo: tenant,
	})
	if err != nil {
		return err
	}
	if plan.Status != agent.PlanStatusApproving || plan.CommitsAt == nil ||
		(req.CommitsAt != 0 && *plan.CommitsAt != req.CommitsAt) {
		return nil
	}

	decideReq := &services.DecideAgentPlanRequest{
		PlanID:        plan.ID,
		Decision:      agent.DecisionAccepted,
		ReasonCode:    req.ReasonCode,
		Note:          req.Note,
		TenantInfo:    tenant,
		PreviewDigest: req.PreviewDigest,
	}
	prepared, err := s.prepare(ctx, decideReq, actor, agent.PlanStatusApproving)
	if err != nil {
		if errortypes.IsBusinessError(err) || errortypes.IsConflictError(err) ||
			errortypes.IsMultiError(err) || errortypes.IsAuthorizationError(err) {
			s.l.Warn("a plan approved in its undo window can no longer go through",
				zap.String("plan", plan.ID.String()), zap.Error(err))
			s.reopenPlan(ctx, plan, actor, false)

			return nil
		}

		return err
	}

	if _, err = s.claimAndRun(ctx, decideReq, actor, prepared, agent.PlanStatusApproving); err != nil {
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
	req *services.SettleApprovalRequest,
	actor *services.RequestActor,
) error {
	plan, err := s.ownApproving(ctx, req, actor)
	if err != nil {
		return err
	}
	if plan.Status != agent.PlanStatusApproving {
		return errAlreadyCommitted()
	}

	if !s.reopenPlan(ctx, plan, actor, true) {
		return errAlreadyCommitted()
	}

	if err = s.workflows.SignalWorkflow(
		ctx, decisioncommitjobs.WorkflowID(plan.ID), "", decisioncommitjobs.UndoSignalName, nil,
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
	req *services.SettleApprovalRequest,
	actor *services.RequestActor,
) error {
	plan, err := s.ownApproving(ctx, req, actor)
	if err != nil {
		return err
	}
	switch plan.Status {
	case agent.PlanStatusApproving:
	case agent.PlanStatusPending:
		return errortypes.NewConflictError(
			"This approval was undone; approve the plan again to run it",
		)
	default:
		return nil
	}

	workflowID := decisioncommitjobs.WorkflowID(plan.ID)
	if err = s.workflows.SignalWorkflow(
		ctx, workflowID, "", decisioncommitjobs.CommitNowSignalName, nil,
	); err == nil {
		return nil
	}

	s.l.Warn("commit-now could not reach its plan's workflow; committing here",
		zap.String("plan", plan.ID.String()), zap.Error(err))

	return s.CommitApproval(ctx, &services.CommitApprovalRequest{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		DecidedByUserID: actor.UserID,
		WorkflowID:      workflowID,
		PlanID:          plan.ID,
	})
}

// ownApproving reads a plan of the actor's own, with the checks deciding it
// takes, so a person can undo exactly what they could approve.
func (s *Service) ownApproving(
	ctx context.Context,
	req *services.SettleApprovalRequest,
	actor *services.RequestActor,
) (*agent.AgentPlan, error) {
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

	return s.plans.GetByID(ctx, repositories.GetAgentPlanByIDRequest{
		ID:         req.PlanID,
		TenantInfo: req.TenantInfo,
	})
}

// reopenPlan moves a plan from Approving back to Pending: undone by the
// actor, or refused at the close of its window. It reports whether it moved;
// a plan that already started to run did not.
func (s *Service) reopenPlan(
	ctx context.Context,
	plan *agent.AgentPlan,
	actor *services.RequestActor,
	undo bool,
) bool {
	tenant := planTenant(plan)
	update := repositories.UpdateAgentPlanStatusRequest{
		ID:         plan.ID,
		TenantInfo: tenant,
		Status:     agent.PlanStatusPending,
		FromStatus: agent.PlanStatusApproving,
		Reopen:     true,
	}
	if undo {
		update.UndoneByUserID = actor.UserID
		update.UndoneAt = timeutils.NowUnix()
	}

	reopened, err := s.plans.UpdateStatus(ctx, update)
	if err != nil {
		if !errortypes.IsVersionMismatchError(err) {
			s.l.Error("a plan could not be returned to waiting",
				zap.String("plan", plan.ID.String()), zap.Error(err))
		}

		return false
	}

	if undo {
		s.logUndo(reopened, actor)
	}
	s.announce(ctx, reopened, actor, services.ActivityUndone)

	return true
}

func (s *Service) logUndo(plan *agent.AgentPlan, actor *services.RequestActor) {
	if s.audit == nil {
		return
	}

	auditActor := actor.AuditActor()
	if err := s.audit.LogAction(&services.LogActionParams{
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

func planTenant(plan *agent.AgentPlan) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: plan.OrganizationID, BuID: plan.BusinessUnitID}
}
