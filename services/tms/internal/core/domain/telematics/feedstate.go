package telematics

import (
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type FeedState struct {
	bun.BaseModel `bun:"table:telematics_feed_states,alias:tfst" json:"-"`

	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	Provider       string   `json:"provider"       bun:"provider,pk,type:VARCHAR(32),notnull,default:'Samsara'"`
	FeedType       FeedType `json:"feedType"       bun:"feed_type,pk,type:VARCHAR(32),notnull"`
	Cursor         string   `json:"cursor"         bun:"cursor,type:TEXT,nullzero"`
	LastPolledAt   int64    `json:"lastPolledAt"   bun:"last_polled_at,type:BIGINT,nullzero"`
	LastSuccessAt  int64    `json:"lastSuccessAt"  bun:"last_success_at,type:BIGINT,nullzero"`
	FailureCount   int      `json:"failureCount"   bun:"failure_count,type:INT,notnull,default:0"`
	LastError      string   `json:"lastError"      bun:"last_error,type:TEXT,nullzero"`
}
