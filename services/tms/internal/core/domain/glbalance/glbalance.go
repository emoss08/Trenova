package glbalance

import "github.com/emoss08/trenova/shared/pulid"

type PeriodAccountBalance struct {
	OrganizationID    pulid.ID `json:"organizationId"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"`
	GLAccountID       pulid.ID `json:"glAccountId"`
	FiscalYearID      pulid.ID `json:"fiscalYearId"`
	FiscalPeriodID    pulid.ID `json:"fiscalPeriodId"`
	AccountCode       string   `json:"accountCode"`
	AccountName       string   `json:"accountName"`
	PeriodDebitMinor  int64    `json:"periodDebitMinor"`
	PeriodCreditMinor int64    `json:"periodCreditMinor"`
	NetChangeMinor    int64    `json:"netChangeMinor"`
}
