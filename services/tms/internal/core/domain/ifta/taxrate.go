package ifta

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const RateScale = 4

var (
	_ bun.BeforeAppendModelHook = (*TaxRate)(nil)
	_ pagination.CursorEntity   = (*TaxRate)(nil)
)

type RateKey struct {
	JurisdictionID pulid.ID
	FuelType       domaintypes.IFTAFuelType
}

type TaxRate struct {
	bun.BaseModel             `bun:"table:ifta_tax_rates,alias:iftr" json:"-"`
	pagination.CursorValueSet `bun:",embed"                          json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	JurisdictionID pulid.ID `json:"jurisdictionId" bun:"jurisdiction_id,type:VARCHAR(100),notnull"`
	Year           int      `json:"year"           bun:"year,type:SMALLINT,notnull"`
	Quarter        int      `json:"quarter"        bun:"quarter,type:SMALLINT,notnull"`

	FuelType               domaintypes.IFTAFuelType `json:"fuelType"               bun:"fuel_type,type:ifta_fuel_type_enum,notnull"`
	RatePerGallon          decimal.Decimal          `json:"ratePerGallon"          bun:"rate_per_gallon,type:NUMERIC(10,4),notnull"`
	SurchargeRatePerGallon decimal.NullDecimal      `json:"surchargeRatePerGallon" bun:"surcharge_rate_per_gallon,type:NUMERIC(10,4),nullzero"`

	SourceNote string `json:"sourceNote" bun:"source_note,type:TEXT,nullzero"`
	SourceURL  string `json:"sourceUrl"  bun:"source_url,type:TEXT,nullzero"`

	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`
	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Jurisdiction *Jurisdiction `json:"jurisdiction,omitempty" bun:"rel:belongs-to,join:jurisdiction_id=id"`
}

func (r *TaxRate) Normalize() {
	r.SourceNote = strings.TrimSpace(r.SourceNote)
	r.SourceURL = strings.TrimSpace(r.SourceURL)
	r.RatePerGallon = r.RatePerGallon.Round(RateScale)
	if r.SurchargeRatePerGallon.Valid {
		r.SurchargeRatePerGallon.Decimal = r.SurchargeRatePerGallon.Decimal.Round(RateScale)
	}
}

func (r *TaxRate) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		r,
		validation.Field(
			&r.JurisdictionID,
			validation.Required.Error("Jurisdiction is required"),
		),
		validation.Field(
			&r.FuelType,
			validation.Required.Error("Fuel type is required"),
			domainvalidation.ValidEnum[domaintypes.IFTAFuelType]("Fuel type is not valid"),
		),
	))

	if err := r.Period().Validate(); err != nil {
		multiErr.Add("quarter", errortypes.ErrInvalid, err.Error())
	}

	if r.FuelType.IsValid() && !r.FuelType.CountsForIFTA() {
		multiErr.Add(
			"fuelType",
			errortypes.ErrInvalid,
			"Only an IFTA fuel type carries a tax rate",
		)
	}

	if r.RatePerGallon.IsNegative() {
		multiErr.Add("ratePerGallon", errortypes.ErrInvalid, "Rate cannot be negative")
	}
	if r.SurchargeRatePerGallon.Valid && r.SurchargeRatePerGallon.Decimal.IsNegative() {
		multiErr.Add(
			"surchargeRatePerGallon",
			errortypes.ErrInvalid,
			"Surcharge rate cannot be negative",
		)
	}
}

func (r *TaxRate) Period() Period { return Period{Year: r.Year, Quarter: r.Quarter} }

func (r *TaxRate) Key() RateKey {
	return RateKey{JurisdictionID: r.JurisdictionID, FuelType: r.FuelType}
}

func (r *TaxRate) HasSurcharge() bool {
	return r.SurchargeRatePerGallon.Valid && r.SurchargeRatePerGallon.Decimal.IsPositive()
}

func (r *TaxRate) SurchargeRate() decimal.Decimal {
	if !r.SurchargeRatePerGallon.Valid {
		return decimal.Zero
	}
	return r.SurchargeRatePerGallon.Decimal
}

func (r *TaxRate) GetID() pulid.ID { return r.ID }

func (r *TaxRate) GetCreatedAt() int64 { return r.CreatedAt }

func (r *TaxRate) GetTableName() string { return "ifta_tax_rates" }

func (r *TaxRate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("iftr_")
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

func (r *TaxRate) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "iftr",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "source_note",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
		},
	}
}
