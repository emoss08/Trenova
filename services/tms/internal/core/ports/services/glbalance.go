package services

import (
	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/shared/pulid"
)

type GLStatementLine struct {
	GLAccountID     pulid.ID             `json:"glAccountId"`
	AccountCode     string               `json:"accountCode"`
	AccountName     string               `json:"accountName"`
	AccountCategory accounttype.Category `json:"accountCategory"`
	AmountMinor     int64                `json:"amountMinor"`
}

type GLStatementSection struct {
	Key        string             `json:"key"`
	Label      string             `json:"label"`
	TotalMinor int64              `json:"totalMinor"`
	Lines      []*GLStatementLine `json:"lines"`
}

type GLIncomeStatement struct {
	FiscalPeriodID   pulid.ID            `json:"fiscalPeriodId"`
	Revenue          *GLStatementSection `json:"revenue"`
	CostOfRevenue    *GLStatementSection `json:"costOfRevenue"`
	OperatingExpense *GLStatementSection `json:"operatingExpense"`
	GrossProfitMinor int64               `json:"grossProfitMinor"`
	NetIncomeMinor   int64               `json:"netIncomeMinor"`
}

type GLBalanceSheet struct {
	FiscalPeriodID              pulid.ID            `json:"fiscalPeriodId"`
	Assets                      *GLStatementSection `json:"assets"`
	Liabilities                 *GLStatementSection `json:"liabilities"`
	Equity                      *GLStatementSection `json:"equity"`
	CurrentPeriodNetIncomeMinor int64               `json:"currentPeriodNetIncomeMinor"`
	TotalAssetsMinor            int64               `json:"totalAssetsMinor"`
	TotalLiabilitiesMinor       int64               `json:"totalLiabilitiesMinor"`
	TotalEquityMinor            int64               `json:"totalEquityMinor"`
}
