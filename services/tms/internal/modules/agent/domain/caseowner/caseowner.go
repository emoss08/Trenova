package caseowner

import (
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type Source string

const (
	// Routed to the person the record names
	SourceAuto = Source("Auto")
	// Claimed by the owner
	SourceClaimed = Source("Claimed")
	// Given to the owner by someone else
	SourceAssigned = Source("Assigned")
)

// Case owners: who owns the case on a record, how they came to own it and since when. A record
// with open Watchtower items and no owner here is unowned.
type Owner struct {
	keel.Table `keel:"record_case_owners,report=Desk,label=Case Owner"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:rcow_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RecordKind recordkind.Kind `json:"recordKind" keel:"record_kind,noreport"`
	RecordID   pulid.ID        `json:"recordId"   keel:"record_id,noreport"`
	OwnerID    pulid.ID        `json:"ownerId"    keel:"owner_id"`
	// How the owner came to own the case: routed to them, claimed or assigned
	Source Source `json:"source"     keel:"source,label=How Owned"`

	AssignedByID pulid.ID `json:"assignedById" keel:"assigned_by_id,nullzero"`
	// When the owner was given the case
	AssignedAt int64 `json:"assignedAt"   keel:"assigned_at,label=Owned Since"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate,noreport"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate,noreport"`

	// The person who owns the case
	Owner *tenant.Person `json:"-" keel:"belongs:owner_id"`
	// The person who assigned it, when someone did
	AssignedBy *tenant.Person `json:"-" keel:"belongs:assigned_by_id,label=Assigned By"`
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
