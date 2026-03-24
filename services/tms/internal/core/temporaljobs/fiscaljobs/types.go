package fiscaljobs

import "github.com/emoss08/trenova/shared/pulid"

const (
	DefaultDaysBeforeYearEnd int = 60
)

type AutoClosePeriodsPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

type AutoClosePeriodsResult struct {
	ClosedCount int      `json:"closedCount"`
	Errors      []string `json:"errors,omitempty"`
}

type AutoCreateFiscalYearPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

type AutoCreateFiscalYearResult struct {
	Created    bool     `json:"created"`
	FiscalYear int      `json:"fiscalYear,omitempty"`
	SkipReason string   `json:"skipReason,omitempty"`
	Errors     []string `json:"errors,omitempty"`
}

type OrgTenant struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

type GetAutoCloseTenantsResult struct {
	Tenants []OrgTenant `json:"tenants"`
}
