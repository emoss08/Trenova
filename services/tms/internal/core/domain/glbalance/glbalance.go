package glbalance

import (
	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/shared/pulid"
)

type PeriodAccountBalance struct {
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

type StatementLine struct {
	GLAccountID     pulid.ID             `json:"glAccountId"`
	AccountCode     string               `json:"accountCode"`
	AccountName     string               `json:"accountName"`
	AccountCategory accounttype.Category `json:"accountCategory"`
	AmountMinor     int64                `json:"amountMinor"`
}

type StatementSection struct {
	Key        string           `json:"key"`
	Label      string           `json:"label"`
	TotalMinor int64            `json:"totalMinor"`
	Lines      []*StatementLine `json:"lines"`
}

type IncomeStatement struct {
	FiscalPeriodID   pulid.ID          `json:"fiscalPeriodId"`
	Revenue          *StatementSection `json:"revenue"`
	CostOfRevenue    *StatementSection `json:"costOfRevenue"`
	OperatingExpense *StatementSection `json:"operatingExpense"`
	GrossProfitMinor int64             `json:"grossProfitMinor"`
	NetIncomeMinor   int64             `json:"netIncomeMinor"`
}

type BalanceSheet struct {
	FiscalPeriodID              pulid.ID          `json:"fiscalPeriodId"`
	Assets                      *StatementSection `json:"assets"`
	Liabilities                 *StatementSection `json:"liabilities"`
	Equity                      *StatementSection `json:"equity"`
	CurrentPeriodNetIncomeMinor int64             `json:"currentPeriodNetIncomeMinor"`
	TotalAssetsMinor            int64             `json:"totalAssetsMinor"`
	TotalLiabilitiesMinor       int64             `json:"totalLiabilitiesMinor"`
	TotalEquityMinor            int64             `json:"totalEquityMinor"`
}
