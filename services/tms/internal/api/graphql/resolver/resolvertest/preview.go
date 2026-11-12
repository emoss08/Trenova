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
	// Batch is what ListByIDs finds, in any order; the IDs asked for that it
	// holds no proposal for are missing.
	Batch []*agent.AgentProposal
}

func (p *PreviewProposals) ListByIDs(
	_ context.Context,
	req agentproposalports.ListAgentProposalsByIDsRequest,
) ([]*agent.AgentProposal, error) {
	asked := make(map[pulid.ID]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		asked[id] = struct{}{}
	}
	out := make([]*agent.AgentProposal, 0, len(req.IDs))
	for _, proposal := range p.Batch {
		if _, ok := asked[proposal.ID]; ok {
			out = append(out, proposal)
		}
	}

	return out, nil
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
	// Batched is the proposals a batch preview was asked for, in order;
	// Failures fail the named ones.
	Batched  []pulid.ID
	Failures map[pulid.ID]error
}

func (s *PreviewService) ForProposals(
	_ context.Context,
	req *agentruntimeports.ProposalPreviewsRequest,
) []agentruntimeports.ProposalPreviewOutcome {
	s.Viewer = req.Viewer
	out := make([]agentruntimeports.ProposalPreviewOutcome, 0, len(req.Proposals))
	for _, proposal := range req.Proposals {
		s.Batched = append(s.Batched, proposal.ID)
		outcome := agentruntimeports.ProposalPreviewOutcome{ProposalID: proposal.ID}
		if err := s.Failures[proposal.ID]; err != nil {
			outcome.Err = err
		} else {
			outcome.Preview = &agentpreview.ProposalPreview{ProposalID: proposal.ID, Digest: "d-" + proposal.ID.String()}
		}
		out = append(out, outcome)
	}

	return out
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
