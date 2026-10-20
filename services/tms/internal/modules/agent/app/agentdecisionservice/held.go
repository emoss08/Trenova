package agentdecisionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) heldVerdict(
	ctx context.Context,
	tenant pagination.TenantInfo,
	proposal *agent.AgentProposal,
) (agentshadow.Verdict, error) {
	if s.shadow == nil {
		return agentshadow.Verdict{}, nil
	}

	return s.shadow.ForRun(ctx, tenant, proposal.RunID)
}

func (s *Service) withdrawHeld(
	ctx context.Context,
	tenant pagination.TenantInfo,
	group []*agent.AgentDecision,
	open []pulid.ID,
	actor kernelports.AuditActor,
) ([]pulid.ID, error) {
	openSet := make(map[pulid.ID]struct{}, len(open))
	for _, id := range open {
		openSet[id] = struct{}{}
	}

	held := make([]pulid.ID, 0)
	verdicts := make(map[pulid.ID]agentshadow.Verdict)
	proposals := make(map[pulid.ID]*agent.AgentProposal)
	for _, decision := range group {
		if _, isOpen := openSet[decision.ID]; !isOpen || decision.ProposalID == nil {
			continue
		}
		proposal, err := s.proposalRepo.GetByID(ctx, agentports.GetAgentProposalByIDRequest{
			ID:         *decision.ProposalID,
			TenantInfo: &tenant,
		})
		if err != nil {
			return nil, err
		}
		verdict, err := s.heldVerdict(ctx, tenant, proposal)
		if err != nil {
			return nil, err
		}
		if verdict.Shadow() {
			held = append(held, decision.ID)
			verdicts[decision.ID] = verdict
			proposals[decision.ID] = proposal
		}
	}
	if len(held) == 0 {
		return open, nil
	}

	withdrawn, err := s.decisionRepo.MarkUndone(ctx, agentports.SettleAgentDecisionsRequest{
		IDs:        held,
		TenantInfo: tenant,
		At:         timeutils.NowUnix(),
	})
	if err != nil {
		return nil, err
	}

	gone := make(map[pulid.ID]struct{}, len(withdrawn))
	for _, id := range withdrawn {
		gone[id] = struct{}{}
		proposal := proposals[id]
		s.reopen(ctx, tenant, proposal.ID)
		s.logHeld(proposal, verdicts[id], actor)
		s.announce(ctx, proposal, tenant, actor, agentports.ActivityUndone)
	}

	remaining := make([]pulid.ID, 0, len(open))
	for _, id := range open {
		if _, isGone := gone[id]; !isGone {
			remaining = append(remaining, id)
		}
	}

	return remaining, nil
}

func (s *Service) logHeld(
	proposal *agent.AgentProposal,
	verdict agentshadow.Verdict,
	actor kernelports.AuditActor,
) {
	s.l.Info("an approval was taken back before it ran, because a switch holds its proposal",
		zap.String("proposal", proposal.ID.String()),
		zap.String("cause", string(verdict.Cause)))
	if s.audit == nil {
		return
	}

	if err := s.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceAgentProposal,
		ResourceID:     proposal.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         actor.UserID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		APIKeyID:       actor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(proposal),
		OrganizationID: proposal.OrganizationID,
		BusinessUnitID: proposal.BusinessUnitID,
	}, auditservice.WithComment(
		"Approval taken back before it ran: "+shadowRefusal(verdict).Error(),
	)); err != nil {
		s.l.Error("failed to log a held approval", zap.Error(err))
	}
}
