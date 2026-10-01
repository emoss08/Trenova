package resolvertest

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PreviewPermissions struct {
	services.PermissionEngine

	Granted map[string]bool
}

func (p *PreviewPermissions) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	return &services.PermissionCheckResult{
		Allowed: p.Granted[req.Resource+"|"+string(req.Operation)],
	}, nil
}

type PreviewProposals struct {
	services.AgentProposalService

	Proposal *agent.AgentProposal
}

func (p *PreviewProposals) GetByID(
	context.Context,
	repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return p.Proposal, nil
}

type PreviewDecisions struct {
	services.AgentDecisionService

	Yours bool
	Asked []pulid.ID
}

func (d *PreviewDecisions) AssertOwnProposal(
	_ context.Context,
	proposalID pulid.ID,
	_ pagination.TenantInfo,
	_ *services.RequestActor,
) error {
	d.Asked = append(d.Asked, proposalID)
	if !d.Yours {
		return errortypes.NewNotFoundError("That proposal was not raised in one of your conversations")
	}

	return nil
}

type PreviewPlans struct {
	services.AgentPlanService

	Yours bool
}

func (p *PreviewPlans) AssertOwnPlan(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
	*services.RequestActor,
) error {
	if !p.Yours {
		return errortypes.NewNotFoundError("That plan was not raised in one of your conversations")
	}

	return nil
}

func (p *PreviewPlans) GetByID(
	_ context.Context,
	req repositories.GetAgentPlanByIDRequest,
) (*agent.AgentPlan, error) {
	return &agent.AgentPlan{ID: req.ID}, nil
}

type PreviewService struct {
	services.ProposalPreviewService

	Proposals int
	Plans     int
	Viewer    *services.PreviewViewer
	Mods      map[string]any
	Warnings  []agent.PreviewWarning
}

func (s *PreviewService) ForProposal(
	_ context.Context,
	req *services.ProposalPreviewRequest,
) (*agent.ProposalPreview, error) {
	s.Proposals++
	s.Viewer = req.Viewer
	s.Mods = req.Modifications

	return &agent.ProposalPreview{ProposalID: req.Proposal.ID, Warnings: s.Warnings}, nil
}

func (s *PreviewService) ForPlan(
	_ context.Context,
	req *services.PlanPreviewRequest,
) (*agent.PlanPreview, error) {
	s.Plans++

	return &agent.PlanPreview{PlanID: req.Plan.ID}, nil
}
