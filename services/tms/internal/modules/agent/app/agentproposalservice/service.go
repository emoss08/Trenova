package agentproposalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   agentports.AgentProposalRepository
}

type Service struct {
	l    *zap.Logger
	repo agentports.AgentProposalRepository
}

func New(p Params) agentports.AgentProposalService {
	return &Service{
		l:    p.Logger.Named("service.agentproposal"),
		repo: p.Repo,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *agentports.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *agentports.ListAgentProposalConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentProposal], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req agentports.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return s.repo.GetByID(ctx, req)
}
