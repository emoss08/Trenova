package agentproposalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   agentports.AgentProposalRepository
	Shadow *agentshadow.Resolver
}

type Service struct {
	l      *zap.Logger
	repo   agentports.AgentProposalRepository
	shadow *agentshadow.Resolver
}

func New(p Params) agentports.AgentProposalService {
	return &Service{
		l:      p.Logger.Named("service.agentproposal"),
		repo:   p.Repo,
		shadow: p.Shadow,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *agentports.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	shadow, err := s.shadow.Organization(ctx, req.Filter.TenantInfo)
	if err != nil {
		return nil, err
	}

	if shadow {
		return &pagination.ListResult[*agent.AgentProposal]{
			Items: []*agent.AgentProposal{},
			Total: 0,
		}, nil
	}

	req.ExcludeShadowDefinitions = true

	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *agentports.ListAgentProposalConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentProposal], error) {
	shadow, err := s.shadow.Organization(ctx, req.Filter.TenantInfo)
	if err != nil {
		return nil, err
	}

	if shadow {
		return &pagination.CursorListResult[*agent.AgentProposal]{
			Items: []*agent.AgentProposal{},
		}, nil
	}

	req.ExcludeShadowDefinitions = true

	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req agentports.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	proposal, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	verdict, err := s.shadow.ForRun(ctx, *req.TenantInfo, proposal.RunID)
	if err != nil {
		return nil, err
	}

	if verdict.Shadow() {
		return nil, errortypes.NewNotFoundError("Agent proposal not found")
	}

	return proposal, nil
}
