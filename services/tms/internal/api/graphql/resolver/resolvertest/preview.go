package resolvertest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PreviewPermissions struct {
	permissionports.PermissionEngine

	Granted map[string]bool
}

func (p *PreviewPermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{
		Allowed: p.Granted[req.Resource+"|"+string(req.Operation)],
	}, nil
}

type PreviewProposals struct {
	agentproposalports.AgentProposalService

	Proposal *agent.AgentProposal
}

func (p *PreviewProposals) GetByID(
	context.Context,
	agentproposalports.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return p.Proposal, nil
}

type PreviewDecisions struct {
	agentdecisionports.AgentDecisionService

	Yours bool
	Asked []pulid.ID
}

func (d *PreviewDecisions) AssertOwnProposal(
	_ context.Context,
	proposalID pulid.ID,
	_ pagination.TenantInfo,
	_ *permissionports.RequestActor,
) error {
	d.Asked = append(d.Asked, proposalID)
	if !d.Yours {
		return errortypes.NewNotFoundError(
			"That proposal was not raised in one of your conversations",
		)
	}

	return nil
}

type PreviewPlans struct {
	agentplanports.AgentPlanService

	Yours bool
}

func (p *PreviewPlans) AssertOwnPlan(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
	*permissionports.RequestActor,
) error {
	if !p.Yours {
		return errortypes.NewNotFoundError("That plan was not raised in one of your conversations")
	}

	return nil
}

func (p *PreviewPlans) GetByID(
	_ context.Context,
	req agentplanports.GetAgentPlanByIDRequest,
) (*agentplan.AgentPlan, error) {
	return &agentplan.AgentPlan{ID: req.ID}, nil
}

type PreviewService struct {
	agentruntimeports.ProposalPreviewService

	Proposals int
	Plans     int
	Viewer    *agentruntimeports.PreviewViewer
	Mods      map[string]any
	Warnings  []agentpreview.PreviewWarning
}

func (s *PreviewService) ForProposal(
	_ context.Context,
	req *agentruntimeports.ProposalPreviewRequest,
) (*agentpreview.ProposalPreview, error) {
	s.Proposals++
	s.Viewer = req.Viewer
	s.Mods = req.Modifications

	return &agentpreview.ProposalPreview{ProposalID: req.Proposal.ID, Warnings: s.Warnings}, nil
}

func (s *PreviewService) ForPlan(
	_ context.Context,
	req *agentruntimeports.PlanPreviewRequest,
) (*agentpreview.PlanPreview, error) {
	s.Plans++

	return &agentpreview.PlanPreview{PlanID: req.Plan.ID}, nil
}
