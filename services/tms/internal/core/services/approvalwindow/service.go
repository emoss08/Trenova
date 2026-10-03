// Package approvalwindow is the one place an approval in its undo window is
// committed, undone or hurried along, whether it approved a proposal, a batch
// of them or a plan. It hands each to the service that decided it.
package approvalwindow

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Decisions services.AgentDecisionService
	Plans     services.AgentPlanService
}

type Service struct {
	decisions services.ApprovalCommitter
	plans     services.ApprovalCommitter
}

func New(p Params) services.ApprovalCommitter {
	return &Service{decisions: p.Decisions, plans: p.Plans}
}

func (s *Service) CommitApproval(ctx context.Context, req *services.CommitApprovalRequest) error {
	if req.PlanID.IsNotNil() {
		return s.plans.CommitApproval(ctx, req)
	}

	return s.decisions.CommitApproval(ctx, req)
}

func (s *Service) UndoOwnApproval(
	ctx context.Context,
	req *services.SettleApprovalRequest,
	actor *services.RequestActor,
) error {
	if req.PlanID.IsNotNil() {
		return s.plans.UndoOwnApproval(ctx, req, actor)
	}

	return s.decisions.UndoOwnApproval(ctx, req, actor)
}

func (s *Service) CommitOwnApprovalNow(
	ctx context.Context,
	req *services.SettleApprovalRequest,
	actor *services.RequestActor,
) error {
	if req.PlanID.IsNotNil() {
		return s.plans.CommitOwnApprovalNow(ctx, req, actor)
	}

	return s.decisions.CommitOwnApprovalNow(ctx, req, actor)
}
