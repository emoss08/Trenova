package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/glbalance"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListTrialBalanceByPeriodRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	FiscalPeriodID pulid.ID              `json:"fiscalPeriodId"`
}

type GLBalanceRepository interface {
	ListTrialBalanceByPeriod(
		ctx context.Context,
		req ListTrialBalanceByPeriodRequest,
	) ([]*glbalance.PeriodAccountBalance, error)
}
