package agentproposalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.AgentProposalRepository
	Shadow *agentshadow.Resolver
}

type Service struct {
	l      *zap.Logger
	repo   repositories.AgentProposalRepository
	shadow *agentshadow.Resolver
}

func New(p Params) services.AgentProposalService {
	return &Service{
		l:      p.Logger.Named("service.agentproposal"),
		repo:   p.Repo,
		shadow: p.Shadow,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListAgentProposalRequest,
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
	req *repositories.ListAgentProposalConnectionRequest,
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
	req repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	proposal, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	shadow, err := s.shadow.ForRun(ctx, *req.TenantInfo, proposal.RunID)
	if err != nil {
		return nil, err
	}

	if shadow {
		return nil, errortypes.NewNotFoundError("Agent proposal not found")
	}

	return proposal, nil
}
