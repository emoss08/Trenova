package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GLPeriodAccountBalance struct {
	OrganizationID    pulid.ID             `json:"organizationId"`
	BusinessUnitID    pulid.ID             `json:"businessUnitId"`
	GLAccountID       pulid.ID             `json:"glAccountId"`
	FiscalYearID      pulid.ID             `json:"fiscalYearId"`
	FiscalPeriodID    pulid.ID             `json:"fiscalPeriodId"`
	AccountCode       string               `json:"accountCode"`
	AccountName       string               `json:"accountName"`
	AccountCategory   accounttype.Category `json:"accountCategory"`
	PeriodDebitMinor  int64                `json:"periodDebitMinor"`
	PeriodCreditMinor int64                `json:"periodCreditMinor"`
	NetChangeMinor    int64                `json:"netChangeMinor"`
}

type ListTrialBalanceByPeriodRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	FiscalPeriodID pulid.ID              `json:"fiscalPeriodId"`
}

type GLBalanceRepository interface {
	ListTrialBalanceByPeriod(
		ctx context.Context,
		req ListTrialBalanceByPeriodRequest,
	) ([]*GLPeriodAccountBalance, error)
}
