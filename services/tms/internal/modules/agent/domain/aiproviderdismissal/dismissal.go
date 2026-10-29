package aiproviderdismissal

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

const FailureDismissalIDPrefix = "aipfd_"

// FailureDismissal is one person putting away the notice that a provider is
// failing. It holds only up to the failure they saw: a newer failure brings
// the notice back.
type FailureDismissal struct {
	keel.Table `keel:"ai_provider_failure_dismissals"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:aipfd_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	UserID         pulid.ID `json:"userId"         keel:"user_id"`
	ProviderID     pulid.ID `json:"providerId"     keel:"provider_id"`
	// FailureAt is the last failure the person saw when they dismissed it.
	FailureAt int64 `json:"failureAt" keel:"failure_at"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
}

// Covers reports whether the dismissal still hides a failure last seen at.
func (d *FailureDismissal) Covers(lastFailureAt int64) bool {
	return d != nil && d.FailureAt >= lastFailureAt
}
