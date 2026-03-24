package apikey

import (
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type UsageDaily struct {
	bun.BaseModel `bun:"table:api_key_usage_daily,alias:akud" json:"-"`

	APIKeyID       pulid.ID  `json:"apiKeyId"       bun:"api_key_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	UsageDate      time.Time `json:"usageDate"      bun:"usage_date,pk,type:date,notnull"`
	RequestCount   int64     `json:"requestCount"   bun:"request_count,notnull,default:0"`
}
