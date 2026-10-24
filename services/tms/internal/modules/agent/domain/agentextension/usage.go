package agentextension

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type ExtensionUsageDaily struct {
	bun.BaseModel `bun:"table:agent_extension_usage_daily,alias:agextu" json:"-"`
	keel.Table    `keel:"agent_extension_usage_daily"`

	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	ExtensionType  Type            `json:"extensionType"  bun:"extension_type,type:VARCHAR(50),pk,notnull" keel:"extension_type,pk"`
	Day            int             `json:"day"            bun:"day,type:INTEGER,pk,notnull" keel:"day,pk"`
	Requests       int             `json:"requests"       bun:"requests,type:INTEGER,notnull" keel:"requests"`
	Failures       int             `json:"failures"       bun:"failures,type:INTEGER,notnull" keel:"failures"`
	CostUSD        decimal.Decimal `json:"costUsd"        bun:"cost_usd,type:NUMERIC(19,6),notnull" keel:"cost_usd"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull" keel:"updated_at"`
}

type UsageSummary struct {
	RequestsToday     int             `json:"requestsToday"`
	RequestsThisMonth int             `json:"requestsThisMonth"`
	FailuresThisMonth int             `json:"failuresThisMonth"`
	CostThisMonthUSD  decimal.Decimal `json:"costThisMonthUsd"`
}
