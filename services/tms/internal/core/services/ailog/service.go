package ailog

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.AILogRepository
}

type Service struct {
	l    *zap.Logger
	repo repositories.AILogRepository
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.ailog"),
		repo: p.Repo,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListAILogRequest,
) (*pagination.ListResult[*ailog.AILog], error) {
	return s.repo.List(ctx, req)
}
