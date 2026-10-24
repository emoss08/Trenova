package aiprovider

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const FailureDismissalIDPrefix = "aipfd_"

// FailureDismissal is one person putting away the notice that a provider is
// failing. It holds only up to the failure they saw: a newer failure brings
// the notice back.
type FailureDismissal struct {
	bun.BaseModel `bun:"table:ai_provider_failure_dismissals,alias:aipfd" json:"-"`
	keel.Table    `keel:"ai_provider_failure_dismissals"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull" keel:"user_id"`
	ProviderID     pulid.ID `json:"providerId"     bun:"provider_id,type:VARCHAR(100),notnull" keel:"provider_id"`
	// FailureAt is the last failure the person saw when they dismissed it.
	FailureAt int64 `json:"failureAt" bun:"failure_at,type:BIGINT,notnull" keel:"failure_at"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
}

// Covers reports whether the dismissal still hides a failure last seen at.
func (d *FailureDismissal) Covers(lastFailureAt int64) bool {
	return d != nil && d.FailureAt >= lastFailureAt
}

func (d *FailureDismissal) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if d.ID.IsNil() {
			d.ID = pulid.MustNew(FailureDismissalIDPrefix)
		}
		d.CreatedAt = timeutils.NowUnix()
	}
	return nil
}
