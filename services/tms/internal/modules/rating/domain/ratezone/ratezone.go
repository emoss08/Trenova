// Package ratezone groups places into named areas a contract can be priced
// against.
//
// Carriers and brokers quote by market area constantly — a Southeast zone, a
// KMA, a metro. One zone stands in for hundreds of postal prefixes, so a
// tariff that would otherwise need three hundred rows needs one.
package ratezone

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*RateZone)(nil)
	_ domaintypes.PostgresSearchable     = (*RateZone)(nil)
)

const (
	maxCodeLength        = 50
	maxNameLength        = 100
	maxDescriptionLength = 500
)

type RateZone struct {
	keel.Table `keel:"rate_zones"`

	ID             pulid.ID           `json:"id"             keel:"id,pk,autoid:rzn_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	Code           string             `json:"code"           keel:"code"`
	Name           string             `json:"name"           keel:"name"`
	Description    string             `json:"description"    keel:"description,nullzero"`
	Kind           ZoneKind           `json:"kind"           keel:"kind"`
	Status         domaintypes.Status `json:"status"         keel:"status"`
	Version        int64              `json:"version"        keel:"version,version"`
	CreatedAt      int64              `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt"      keel:"updated_at,autoupdate"`
	SearchVector   string             `json:"-"              keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string             `json:"-"              keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
	Members      []*RateZoneMember    `json:"members,omitempty" keel:"many:rate_zone_id"`
}

// applyDefaults fills in the values the database would have defaulted, so a
// payload that omits them validates the same way it will be stored.
func (rz *RateZone) applyDefaults() {
	if rz.Status == "" {
		rz.Status = domaintypes.StatusActive
	}
	if rz.Kind == "" {
		rz.Kind = ZoneKindCustom
	}
}

func (rz *RateZone) Validate(multiErr *errortypes.MultiError) {
	rz.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(rz,
		validation.Field(&rz.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, maxCodeLength).
				Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(&rz.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxNameLength).
				Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&rz.Description,
			validation.Length(0, maxDescriptionLength).
				Error("Description cannot be longer than 500 characters"),
		),
		validation.Field(&rz.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[ZoneKind]("Kind is invalid"),
		),
		validation.Field(&rz.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status is invalid"),
		),
	))

	// An empty zone matches nothing, so every rule priced against it would be
	// stored and never fire. That is worth refusing at the point the zone is
	// saved rather than leaving as a rate that mysteriously never applies.
	if len(rz.Members) == 0 {
		multiErr.Add("members", errortypes.ErrRequired, "A zone must have at least one member")
	}

	rz.validateMembers(multiErr)
}

// StampMembers re-seats on every member the values it inherits from its zone:
// tenancy, its parent, and the key it is matched by.
//
// A caller sends members as the places they describe, not as rows — nothing in
// a payload should be trusted to say which zone or which organization a member
// belongs to, and the key is derived, never sent.
func (rz *RateZone) StampMembers(ctx context.Context) error {
	members := make([]*RateZoneMember, 0, len(rz.Members))
	scopes := make([]rategeo.Scope, 0, len(rz.Members))
	for _, member := range rz.Members {
		if member == nil {
			continue
		}

		member.RateZoneID = rz.ID
		member.OrganizationID = rz.OrganizationID
		member.BusinessUnitID = rz.BusinessUnitID
		members = append(members, member)
		scopes = append(scopes, member.Scope())
	}

	keys, err := rategeo.ScopeKeys(ctx, scopes)
	if err != nil {
		return fmt.Errorf("key rate zone members: %w", err)
	}
	for i, member := range members {
		member.MatchKey = keys[i]
		member.keyApplied = true
	}

	return nil
}

// validateMembers checks each member on its own, then rejects duplicates across
// the set. A zone that lists the same place twice contributes it twice to every
// candidate lookup, which shows up later as the same rate rule appearing more
// than once in a rating trace.
func (rz *RateZone) validateMembers(multiErr *errortypes.MultiError) {
	seen := make(map[string]struct{}, len(rz.Members))

	for i, member := range rz.Members {
		if member == nil {
			continue
		}

		memberErr := multiErr.WithIndex("members", i)
		member.Validate(memberErr)

		if member.MatchKey == "" {
			continue
		}

		if _, dup := seen[member.MatchKey]; dup {
			memberErr.Add(
				"scopeValue",
				errortypes.ErrDuplicate,
				"This place is already part of the zone",
			)
		}
		seen[member.MatchKey] = struct{}{}
	}
}

func (rz *RateZone) BeforeCreate(ctx context.Context) error {
	if rz.Kind == "" {
		rz.Kind = ZoneKindCustom
	}
	if rz.Status == "" {
		rz.Status = domaintypes.StatusActive
	}

	return nil
}

func (rz *RateZone) GetID() pulid.ID {
	return rz.ID
}

func (rz *RateZone) GetCreatedAt() int64 {
	return rz.CreatedAt
}

func (rz *RateZone) GetOrganizationID() pulid.ID {
	return rz.OrganizationID
}

func (rz *RateZone) GetBusinessUnitID() pulid.ID {
	return rz.BusinessUnitID
}

func (rz *RateZone) GetTableName() string {
	return "rate_zones"
}

func (rz *RateZone) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "rzn",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "kind", Type: domaintypes.FieldTypeEnum},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (RateZone) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_rate_zones_code",
			Message: "A rate zone with this code already exists in your organization",
		},
	}
}
