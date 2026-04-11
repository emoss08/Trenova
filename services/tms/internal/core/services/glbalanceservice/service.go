package glbalanceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/glbalance"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.GLBalanceRepository
}

type Service struct {
	l    *zap.Logger
	repo repositories.GLBalanceRepository
}

func New(p Params) *Service {
	return &Service{l: p.Logger.Named("service.gl-balance"), repo: p.Repo}
}

func (s *Service) ListTrialBalanceByPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fiscalPeriodID pulid.ID,
) ([]*glbalance.PeriodAccountBalance, error) {
	return s.repo.ListTrialBalanceByPeriod(ctx, repositories.ListTrialBalanceByPeriodRequest{
		TenantInfo:     tenantInfo,
		FiscalPeriodID: fiscalPeriodID,
	})
}
