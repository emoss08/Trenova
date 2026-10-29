package agentproposalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   agentproposalports.AgentProposalRepository
}

type Service struct {
	l    *zap.Logger
	repo agentproposalports.AgentProposalRepository
}

func New(p Params) agentproposalports.AgentProposalService {
	return &Service{
		l:    p.Logger.Named("service.agentproposal"),
		repo: p.Repo,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *agentproposalports.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *agentproposalports.ListAgentProposalConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentProposal], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req agentproposalports.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return s.repo.GetByID(ctx, req)
}
