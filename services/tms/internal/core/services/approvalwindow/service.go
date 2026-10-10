// Package approvalwindow is the one place an approval in its undo window is
// committed, undone or hurried along, whether it approved a proposal, a batch
// of them or a plan. It hands each to the service that decided it.
package approvalwindow

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Decisions agentports.AgentDecisionService
	Plans     agentports.AgentPlanService
}

type Service struct {
	decisions agentports.ApprovalCommitter
	plans     agentports.ApprovalCommitter
}

func New(p Params) agentports.ApprovalCommitter {
	return &Service{decisions: p.Decisions, plans: p.Plans}
}

func (s *Service) CommitApproval(ctx context.Context, req *agentports.CommitApprovalRequest) error {
	if req.PlanID.IsNotNil() {
		return s.plans.CommitApproval(ctx, req)
	}

	return s.decisions.CommitApproval(ctx, req)
}

func (s *Service) UndoOwnApproval(
	ctx context.Context,
	req *agentports.SettleApprovalRequest,
	actor *permissionports.RequestActor,
) error {
	if req.PlanID.IsNotNil() {
		return s.plans.UndoOwnApproval(ctx, req, actor)
	}

	return s.decisions.UndoOwnApproval(ctx, req, actor)
}

func (s *Service) CommitOwnApprovalNow(
	ctx context.Context,
	req *agentports.SettleApprovalRequest,
	actor *permissionports.RequestActor,
) error {
	if req.PlanID.IsNotNil() {
		return s.plans.CommitOwnApprovalNow(ctx, req, actor)
	}

	return s.decisions.CommitOwnApprovalNow(ctx, req, actor)
}
