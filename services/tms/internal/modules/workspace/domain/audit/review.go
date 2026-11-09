package audit

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type Review struct {
	keel.Table `keel:"audit_reviews"`

	ID             pulid.ID            `json:"id"                     keel:"id,pk,autoid:arv_"`
	BusinessUnitID pulid.ID            `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID            `json:"organizationId"         keel:"organization_id,pk,tenant"`
	Kind           AlertKind           `json:"kind"                   keel:"kind"`
	DedupeKey      string              `json:"-"                      keel:"dedupe_key"`
	EntryID        pulid.ID            `json:"entryId"                keel:"entry_id"`
	Action         string              `json:"action"                 keel:"action"`
	Resource       permission.Resource `json:"resource"               keel:"resource"`
	ResourceID     string              `json:"resourceId,omitempty"   keel:"resource_id,nullzero"`
	EntityRef      string              `json:"entityRef,omitempty"    keel:"entity_ref,nullzero"`
	PrincipalID    pulid.ID            `json:"principalId"            keel:"principal_id"`
	PrincipalType  string              `json:"principalType"          keel:"principal_type"`
	ActorName      string              `json:"actorName,omitempty"    keel:"actor_name,nullzero"`
	Count          int64               `json:"count"                  keel:"count"`
	Threshold      int                 `json:"threshold"              keel:"threshold"`
	WindowMinutes  int                 `json:"windowMinutes"          keel:"window_minutes"`
	OccurredAt     int64               `json:"occurredAt"             keel:"occurred_at"`
	ReviewedByID   pulid.ID            `json:"reviewedById,omitempty" keel:"reviewed_by_id,nullzero"`
	ReviewedAt     *int64              `json:"reviewedAt,omitempty"   keel:"reviewed_at"`
	Version        int64               `json:"version"                keel:"version,version"`
	CreatedAt      int64               `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt      int64               `json:"updatedAt"              keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (r *Review) Open() bool {
	return r != nil && r.ReviewedAt == nil
}

func (r *Review) Counted() bool {
	def, ok := AlertDefinitionFor(r.Kind)
	return ok && def.Counted
}
