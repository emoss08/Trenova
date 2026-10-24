package agentextension

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ExtensionUsageDaily struct {
	keel.Table `keel:"agent_extension_usage_daily"`

	OrganizationID pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	ExtensionType  Type            `json:"extensionType" keel:"extension_type,pk"`
	Day            int             `json:"day" keel:"day,pk"`
	Requests       int             `json:"requests" keel:"requests"`
	Failures       int             `json:"failures" keel:"failures"`
	CostUSD        decimal.Decimal `json:"costUsd" keel:"cost_usd"`
	UpdatedAt      int64           `json:"updatedAt" keel:"updated_at"`
}

type UsageSummary struct {
	RequestsToday     int             `json:"requestsToday"`
	RequestsThisMonth int             `json:"requestsThisMonth"`
	FailuresThisMonth int             `json:"failuresThisMonth"`
	CostThisMonthUSD  decimal.Decimal `json:"costThisMonthUsd"`
}
