package caseowner

import (
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
)

type Source string

const (
	SourceAuto     = Source("Auto")
	SourceClaimed  = Source("Claimed")
	SourceAssigned = Source("Assigned")
)

type Owner struct {
	keel.Table `keel:"record_case_owners"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:rcow_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RecordKind recordkind.Kind `json:"recordKind" keel:"record_kind"`
	RecordID   pulid.ID        `json:"recordId"   keel:"record_id"`
	OwnerID    pulid.ID        `json:"ownerId"    keel:"owner_id"`
	Source     Source          `json:"source"     keel:"source"`

	AssignedByID pulid.ID `json:"assignedById" keel:"assigned_by_id,nullzero"`
	AssignedAt   int64    `json:"assignedAt"   keel:"assigned_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (o *Owner) GetID() pulid.ID { return o.ID }

func (o *Owner) GetTableName() string { return "record_case_owners" }

type Sight struct {
	All bool
	As  []pulid.ID
	// Viewer is the person reading. A record shared with them is in their
	// sight whoever owns its case: they read it and decide nothing on it.
	// Nil for anything that is not a person.
	Viewer pulid.ID
}

func (s Sight) Works(owner pulid.ID) bool {
	return owner.IsNil() || slices.Contains(s.As, owner)
}

func (s Sight) Sees(owner pulid.ID) bool {
	return s.All || s.Works(owner)
}
