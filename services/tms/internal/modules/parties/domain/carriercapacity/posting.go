package carriercapacity

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const (
	MaxRadiusMiles   = 1000
	MaxTruckCount    = 999
	MaxNotesLength   = 2000
	MaxPostingWindow = int64(60 * 24 * 60 * 60)
)

var (
	_ validationframework.TenantedEntity = (*Posting)(nil)
	_ domaintypes.PostgresSearchable     = (*Posting)(nil)
	_ pagination.CursorEntity            = (*Posting)(nil)
)

type Posting struct {
	keel.Table `keel:"carrier_capacity_postings"`

	ID                 pulid.ID            `json:"id"                 keel:"id,pk,autoid:ccp_"`
	BusinessUnitID     pulid.ID            `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID            `json:"organizationId"     keel:"organization_id,pk,tenant"`
	CarrierID          pulid.ID            `json:"carrierId"          keel:"carrier_id"`
	OriginLocationID   *pulid.ID           `json:"originLocationId"   keel:"origin_location_id"`
	OriginStateID      *pulid.ID           `json:"originStateId"      keel:"origin_state_id"`
	OriginRadiusMiles  *int                `json:"originRadiusMiles"  keel:"origin_radius_miles"`
	DestinationStateID *pulid.ID           `json:"destinationStateId" keel:"destination_state_id"`
	EquipmentTypeID    *pulid.ID           `json:"equipmentTypeId"    keel:"equipment_type_id"`
	AvailableFrom      int64               `json:"availableFrom"      keel:"available_from"`
	AvailableTo        int64               `json:"availableTo"        keel:"available_to"`
	TruckCount         int                 `json:"truckCount"         keel:"truck_count"`
	RateMethod         RateMethod          `json:"rateMethod"         keel:"rate_method"`
	Rate               decimal.NullDecimal `json:"rate"               keel:"rate"`
	Source             Source              `json:"source"             keel:"source"`
	Notes              string              `json:"notes"              keel:"notes,nullzero"`
	SearchVector       string              `json:"-"                  keel:"search_vector,tsvector,readonly,noselect"`
	Rank               string              `json:"-"                  keel:"rank,computed"`
	Version            int64               `json:"version"            keel:"version,version"`
	CreatedAt          int64               `json:"createdAt"          keel:"created_at,autocreate"`
	UpdatedAt          int64               `json:"updatedAt"          keel:"updated_at,autoupdate"`

	BusinessUnit     *tenant.BusinessUnit         `json:"businessUnit,omitempty"`
	Organization     *tenant.Organization         `json:"organization,omitempty"`
	Carrier          *carrier.Carrier             `json:"carrier,omitempty"          keel:"belongs:carrier_id"`
	OriginLocation   *location.Location           `json:"originLocation,omitempty"   keel:"belongs:origin_location_id"`
	OriginState      *usstate.UsState             `json:"originState,omitempty"      keel:"belongs:origin_state_id"`
	DestinationState *usstate.UsState             `json:"destinationState,omitempty" keel:"belongs:destination_state_id"`
	EquipmentType    *equipmenttype.EquipmentType `json:"equipmentType,omitempty"    keel:"belongs:equipment_type_id"`
}

func (p *Posting) BeforeCreate(_ context.Context) error {
	if p.TruckCount == 0 {
		p.TruckCount = 1
	}
	if p.RateMethod == "" {
		p.RateMethod = RateMethodPerMile
	}
	if p.Source == "" {
		p.Source = SourceManual
	}

	return nil
}

func (p *Posting) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.CarrierID, validation.Required.Error("Carrier is required")),
		validation.Field(&p.AvailableFrom,
			validation.Required.Error("Available from is required"),
		),
		validation.Field(&p.AvailableTo,
			validation.Required.Error("Available to is required"),
		),
		validation.Field(&p.TruckCount,
			validation.Required.Error("Truck count is required"),
			validation.Min(1).Error("Truck count must be at least 1"),
			validation.Max(MaxTruckCount).Error("Truck count must be 999 or fewer"),
		),
		validation.Field(&p.RateMethod,
			validation.Required.Error("Rate method is required"),
			domainvalidation.ValidEnum[RateMethod]("Rate method must be Flat or PerMile"),
		),
		validation.Field(&p.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[Source]("Source must be Manual, Email or EDI"),
		),
		validation.Field(&p.Notes,
			validation.Length(0, MaxNotesLength).Error("Notes must be 2000 characters or fewer"),
		),
	))

	p.validateOrigin(multiErr)
	p.validateWindow(multiErr)
	p.validateRate(multiErr)
}

func (p *Posting) validateOrigin(multiErr *errortypes.MultiError) {
	hasLocation := p.OriginLocationID != nil && p.OriginLocationID.IsNotNil()
	hasState := p.OriginStateID != nil && p.OriginStateID.IsNotNil()
	if !hasLocation && !hasState {
		multiErr.Add(
			"originLocationId",
			errortypes.ErrRequired,
			"An origin location or origin state is required",
		)
	}
	if p.OriginRadiusMiles == nil {
		return
	}
	if !hasLocation {
		multiErr.Add(
			"originRadiusMiles",
			errortypes.ErrInvalid,
			"A radius needs an origin location to measure from",
		)
	}
	if *p.OriginRadiusMiles <= 0 || *p.OriginRadiusMiles > MaxRadiusMiles {
		multiErr.Add(
			"originRadiusMiles",
			errortypes.ErrInvalid,
			"Radius must be between 1 and 1000 miles",
		)
	}
}

func (p *Posting) validateWindow(multiErr *errortypes.MultiError) {
	if p.AvailableFrom <= 0 || p.AvailableTo <= 0 {
		return
	}
	if p.AvailableTo <= p.AvailableFrom {
		multiErr.Add(
			"availableTo",
			errortypes.ErrInvalid,
			"Available to must be after available from",
		)
		return
	}
	if p.AvailableTo-p.AvailableFrom > MaxPostingWindow {
		multiErr.Add(
			"availableTo",
			errortypes.ErrInvalid,
			"A posting cannot stay open longer than 60 days",
		)
	}
}

func (p *Posting) validateRate(multiErr *errortypes.MultiError) {
	if p.Rate.Valid && p.Rate.Decimal.IsNegative() {
		multiErr.Add("rate", errortypes.ErrInvalid, "Rate cannot be negative")
	}
}

func (p *Posting) ActiveAt(at int64) bool {
	return p.AvailableFrom <= at && at <= p.AvailableTo
}

func (p *Posting) QuoteFor(miles float64) decimal.NullDecimal {
	if !p.Rate.Valid {
		return decimal.NullDecimal{}
	}
	if p.RateMethod == RateMethodFlat {
		return p.Rate
	}
	return decimal.NewNullDecimal(p.Rate.Decimal.Mul(decimal.NewFromFloat(miles)).Round(2))
}

func (p *Posting) RatePerMile(miles float64) decimal.NullDecimal {
	if !p.Rate.Valid {
		return decimal.NullDecimal{}
	}
	if p.RateMethod == RateMethodPerMile {
		return p.Rate
	}
	if miles <= 0 {
		return decimal.NullDecimal{}
	}
	return decimal.NewNullDecimal(p.Rate.Decimal.Div(decimal.NewFromFloat(miles)).Round(4))
}

func (p *Posting) GetID() pulid.ID { return p.ID }

func (p *Posting) GetCreatedAt() int64 { return p.CreatedAt }

func (p *Posting) GetTableName() string { return "carrier_capacity_postings" }

func (p *Posting) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *Posting) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *Posting) GetVersion() int64 { return p.Version }

func (p *Posting) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "ccp",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "notes", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
		UseSearchVector: true,
	}
}

func (Posting) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_carrier_capacity_postings_carrier",
			Message: "Carrier does not exist in your organization",
		},
		{
			Name:    "fk_carrier_capacity_postings_origin_location",
			Message: "Origin location does not exist in your organization",
		},
		{
			Name:    "fk_carrier_capacity_postings_equipment_type",
			Message: "Equipment type does not exist in your organization",
		},
	}
}
