package ratezone

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var ErrMatchKeyNotApplied = errors.New(
	"a rate zone member is written before its match key was applied",
)

// RateZoneMember is one place that belongs to a zone.
//
// A member names a place with the same vocabulary a rate rule uses, so a zone
// is exactly a union of the primitive scopes. Zones deliberately cannot contain
// other zones: nesting would turn membership from an indexed lookup into a
// graph walk, and the rating path cannot afford that.
type RateZoneMember struct {
	keel.Table `keel:"rate_zone_members"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:rzm_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	RateZoneID     pulid.ID `json:"rateZoneId" keel:"rate_zone_id"`

	ScopeType  rategeo.ScopeType `json:"scopeType" keel:"scope_type"`
	ScopeValue string            `json:"scopeValue" keel:"scope_value,nullzero"`
	City       string            `json:"city" keel:"city,nullzero"`
	MatchKey   string            `json:"matchKey" keel:"match_key"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	RateZone *RateZone `json:"-" keel:"belongs:rate_zone_id"`

	keyApplied bool
}

func (rzm *RateZoneMember) BeforeUpdate(_ context.Context) error {
	if rzm.MatchKey == "" {
		return ErrMatchKeyNotApplied
	}

	return nil
}

// Scope reads the member back as the geography value object the key helpers
// operate on.
func (rzm *RateZoneMember) Scope() rategeo.Scope {
	return rategeo.Scope{
		Type:  rzm.ScopeType,
		Value: rzm.ScopeValue,
		City:  rzm.City,
	}
}

func (rzm *RateZoneMember) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(rzm,
		validation.Field(&rzm.ScopeType,
			validation.Required.Error("Scope type is required"),
			domainvalidation.ValidEnum[rategeo.ScopeType]("Scope type is invalid"),
		),
		validation.Field(&rzm.ScopeValue,
			validation.Length(0, 120).
				Error("Scope value cannot be longer than 120 characters"),
		),
		validation.Field(&rzm.City,
			validation.Length(0, 100).Error("City cannot be longer than 100 characters"),
		),
	))

	rzm.validateScope(multiErr)
}

func (rzm *RateZoneMember) validateScope(multiErr *errortypes.MultiError) {
	switch rzm.ScopeType {
	case rategeo.ScopeTypeZone:
		multiErr.Add(
			"scopeType",
			errortypes.ErrInvalid,
			"A zone cannot contain another zone",
		)
		return
	case rategeo.ScopeTypeAny:
		multiErr.Add(
			"scopeType",
			errortypes.ErrInvalid,
			"A zone member must name a place",
		)
		return
	case rategeo.ScopeTypeRadius:
		multiErr.Add(
			"scopeType",
			errortypes.ErrInvalid,
			"A radius belongs on a rate rule rather than in a zone",
		)
		return
	case rategeo.ScopeTypeCityState:
		if rzm.City == "" {
			multiErr.Add("city", errortypes.ErrRequired, "City is required")
		}
	case rategeo.ScopeTypeCountry,
		rategeo.ScopeTypeState,
		rategeo.ScopeTypeZip3,
		rategeo.ScopeTypeZip5,
		rategeo.ScopeTypeLocation:
	}

	if rzm.ScopeValue == "" {
		multiErr.Add("scopeValue", errortypes.ErrRequired, "Scope value is required")
		return
	}

	// A member whose scope cannot be reduced to a key would be stored but never
	// matched, which reads to the user as a zone that silently ignores part of
	// its own definition. The commonest cause is a postal code too short to
	// yield a three digit prefix.
	if rzm.keyApplied && rzm.MatchKey == "" {
		multiErr.Add(
			"scopeValue",
			errortypes.ErrInvalid,
			"Scope value is not valid for a {0} member", rzm.ScopeType.String(),
		)
	}
}

func (rzm *RateZoneMember) BeforeCreate(ctx context.Context) error {
	if rzm.MatchKey == "" {
		return ErrMatchKeyNotApplied
	}

	return nil
}

func (rzm *RateZoneMember) GetID() pulid.ID {
	return rzm.ID
}

func (rzm *RateZoneMember) GetOrganizationID() pulid.ID {
	return rzm.OrganizationID
}

func (rzm *RateZoneMember) GetBusinessUnitID() pulid.ID {
	return rzm.BusinessUnitID
}

func (rzm *RateZoneMember) GetTableName() string {
	return "rate_zone_members"
}
