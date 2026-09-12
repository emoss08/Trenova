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

// ListYearToDateBalancesRequest asks for one row per account summed across every
// period of a fiscal year — the position a year-end close reads to work out what
// to close and what to carry forward.
type ListYearToDateBalancesRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	FiscalYearID pulid.ID              `json:"fiscalYearId"`
}

// ListCumulativeBalancesThroughPeriodRequest asks for each account's position as
// at the end of one period: every period of that fiscal year up to and including
// it, summed. The year's opening entry sits in period 1, so summing within the
// year yields the full carried-forward balance without reaching into prior years.
type ListCumulativeBalancesThroughPeriodRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	FiscalPeriodID pulid.ID              `json:"fiscalPeriodId"`
}

type GLBalanceRepository interface {
	ListTrialBalanceByPeriod(
		ctx context.Context,
		req ListTrialBalanceByPeriodRequest,
	) ([]*GLPeriodAccountBalance, error)
	ListYearToDateBalances(
		ctx context.Context,
		req ListYearToDateBalancesRequest,
	) ([]*GLPeriodAccountBalance, error)
	ListCumulativeBalancesThroughPeriod(
		ctx context.Context,
		req ListCumulativeBalancesThroughPeriodRequest,
	) ([]*GLPeriodAccountBalance, error)
}
