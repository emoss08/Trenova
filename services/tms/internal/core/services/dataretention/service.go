package dataretention

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.DataRetentionRepository
	AuditService services.AuditService
}

type Service struct {
	l    *zap.Logger
	repo repositories.DataRetentionRepository
	as   services.AuditService
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.dataretention"),
		repo: p.Repo,
		as:   p.AuditService,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	return s.repo.Get(ctx, req)
}

func (s *Service) Update(
	ctx context.Context,
	dr *tenant.DataRetention,
	userID pulid.ID,
) (*tenant.DataRetention, error) {
	return s.repo.Update(ctx, dr)
}
