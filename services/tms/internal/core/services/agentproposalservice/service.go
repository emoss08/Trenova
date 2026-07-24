package agentproposalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger  *zap.Logger
	Repo    repositories.AgentProposalRepository
	Control services.AgentControlService
}

type Service struct {
	l       *zap.Logger
	repo    repositories.AgentProposalRepository
	control services.AgentControlService
}

func New(p Params) services.AgentProposalService {
	return &Service{
		l:       p.Logger.Named("service.agentproposal"),
		repo:    p.Repo,
		control: p.Control,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	shadow, err := s.isShadow(ctx, req.Filter.TenantInfo)
	if err != nil {
		return nil, err
	}

	if shadow {
		return &pagination.ListResult[*agent.AgentProposal]{
			Items: []*agent.AgentProposal{},
			Total: 0,
		}, nil
	}

	return s.repo.List(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	shadow, err := s.isShadow(ctx, *req.TenantInfo)
	if err != nil {
		return nil, err
	}

	if shadow {
		return nil, errortypes.NewNotFoundError("Agent proposal not found")
	}

	return s.repo.GetByID(ctx, req)
}

func (s *Service) isShadow(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error) {
	control, err := s.control.Get(ctx, tenantInfo)
	if err != nil {
		return false, err
	}

	return control.ShadowMode, nil
}
