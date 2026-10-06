package carriercapacity

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	MaxRadiusMiles   = 1000
	MaxTruckCount    = 999
	MaxNotesLength   = 2000
	MaxPostingWindow = int64(60 * 24 * 60 * 60)
)

var (
	_ bun.BeforeAppendModelHook          = (*Posting)(nil)
	_ validationframework.TenantedEntity = (*Posting)(nil)
	_ domaintypes.PostgresSearchable     = (*Posting)(nil)
	_ pagination.CursorEntity            = (*Posting)(nil)
)

type Posting struct {
	bun.BaseModel             `bun:"table:carrier_capacity_postings,alias:ccp" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                    json:"-"`

	ID                 pulid.ID            `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID     pulid.ID            `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID     pulid.ID            `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	CarrierID          pulid.ID            `json:"carrierId"          bun:"carrier_id,type:VARCHAR(100),notnull"`
	OriginLocationID   *pulid.ID           `json:"originLocationId"   bun:"origin_location_id,type:VARCHAR(100),nullzero"`
	OriginStateID      *pulid.ID           `json:"originStateId"      bun:"origin_state_id,type:VARCHAR(100),nullzero"`
	OriginRadiusMiles  *int                `json:"originRadiusMiles"  bun:"origin_radius_miles,type:INTEGER,nullzero"`
	DestinationStateID *pulid.ID           `json:"destinationStateId" bun:"destination_state_id,type:VARCHAR(100),nullzero"`
	EquipmentTypeID    *pulid.ID           `json:"equipmentTypeId"    bun:"equipment_type_id,type:VARCHAR(100),nullzero"`
	AvailableFrom      int64               `json:"availableFrom"      bun:"available_from,type:BIGINT,notnull"`
	AvailableTo        int64               `json:"availableTo"        bun:"available_to,type:BIGINT,notnull"`
	TruckCount         int                 `json:"truckCount"         bun:"truck_count,type:INTEGER,notnull,default:1"`
	RateMethod         RateMethod          `json:"rateMethod"         bun:"rate_method,type:VARCHAR(20),notnull,default:'PerMile'"`
	Rate               decimal.NullDecimal `json:"rate"               bun:"rate,type:NUMERIC(19,4),nullzero"`
	Source             Source              `json:"source"             bun:"source,type:VARCHAR(20),notnull,default:'Manual'"`
	Notes              string              `json:"notes"              bun:"notes,type:TEXT,nullzero"`
	SearchVector       string              `json:"-"                  bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank               string              `json:"-"                  bun:"rank,type:VARCHAR(100),scanonly"`
	Version            int64               `json:"version"            bun:"version,type:BIGINT"`
	CreatedAt          int64               `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64               `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit     *tenant.BusinessUnit         `json:"businessUnit,omitempty"     bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization     *tenant.Organization         `json:"organization,omitempty"     bun:"rel:belongs-to,join:organization_id=id"`
	Carrier          *carrier.Carrier             `json:"carrier,omitempty"          bun:"rel:belongs-to,join:carrier_id=id"`
	OriginLocation   *location.Location           `json:"originLocation,omitempty"   bun:"rel:belongs-to,join:origin_location_id=id"`
	OriginState      *usstate.UsState             `json:"originState,omitempty"      bun:"rel:belongs-to,join:origin_state_id=id"`
	DestinationState *usstate.UsState             `json:"destinationState,omitempty" bun:"rel:belongs-to,join:destination_state_id=id"`
	EquipmentType    *equipmenttype.EquipmentType `json:"equipmentType,omitempty"    bun:"rel:belongs-to,join:equipment_type_id=id"`
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

func (p *Posting) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("ccp_")
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
