package usstateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.UsStateRepository
}

type Service struct {
	l    *zap.Logger
	repo repositories.UsStateRepository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.usstate"),
		repo: p.Repo,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetUsStateByIDRequest,
) (*usstate.UsState, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*usstate.UsState], error) {
	return s.repo.SelectOptions(ctx, req)
}
