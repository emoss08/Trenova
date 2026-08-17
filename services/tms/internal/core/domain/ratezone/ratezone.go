// Package ratezone groups places into named areas a contract can be priced
// against.
//
// Carriers and brokers quote by market area constantly — a Southeast zone, a
// KMA, a metro. One zone stands in for hundreds of postal prefixes, so a
// tariff that would otherwise need three hundred rows needs one.
package ratezone

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*RateZone)(nil)
	_ validationframework.TenantedEntity = (*RateZone)(nil)
	_ domaintypes.PostgresSearchable     = (*RateZone)(nil)
)

const (
	maxCodeLength        = 50
	maxNameLength        = 100
	maxDescriptionLength = 500
)

type RateZone struct {
	bun.BaseModel             `bun:"table:rate_zones,alias:rzn" json:"-"`
	pagination.CursorValueSet `bun:",embed"                     json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Code           string             `json:"code"           bun:"code,type:VARCHAR(50),notnull"`
	Name           string             `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Description    string             `json:"description"    bun:"description,type:TEXT,nullzero"`
	Kind           ZoneKind           `json:"kind"           bun:"kind,type:rate_zone_kind_enum,notnull,default:'Custom'"`
	Status         domaintypes.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector   string             `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string             `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit *tenant.BusinessUnit `json:"-"                 bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"                 bun:"rel:belongs-to,join:organization_id=id"`
	Members      []*RateZoneMember    `json:"members,omitempty" bun:"rel:has-many,join:id=rate_zone_id"`
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

		member.ApplyMatchKey()
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

func (rz *RateZone) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rz.ID.IsNil() {
			rz.ID = pulid.MustNew("rzn_")
		}
		rz.CreatedAt = now
		rz.UpdatedAt = now
	case *bun.UpdateQuery:
		rz.UpdatedAt = now
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
