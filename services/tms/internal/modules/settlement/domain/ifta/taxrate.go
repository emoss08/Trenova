package ifta

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const RateScale = 4

var (
	_ pagination.CursorEntity = (*TaxRate)(nil)
)

type RateKey struct {
	JurisdictionID pulid.ID
	FuelType       domaintypes.IFTAFuelType
}

type TaxRate struct {
	keel.Table `keel:"ifta_tax_rates"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:iftr_"`
	JurisdictionID pulid.ID `json:"jurisdictionId" keel:"jurisdiction_id"`
	Year           int      `json:"year" keel:"year"`
	Quarter        int      `json:"quarter" keel:"quarter"`

	FuelType               domaintypes.IFTAFuelType `json:"fuelType" keel:"fuel_type"`
	RatePerGallon          decimal.Decimal          `json:"ratePerGallon" keel:"rate_per_gallon"`
	SurchargeRatePerGallon decimal.NullDecimal      `json:"surchargeRatePerGallon" keel:"surcharge_rate_per_gallon"`

	SourceNote string `json:"sourceNote" keel:"source_note,nullzero"`
	SourceURL  string `json:"sourceUrl" keel:"source_url,nullzero"`

	SearchVector string `json:"-"`
	Rank         string `json:"-" keel:"rank,computed"`
	Version      int64  `json:"version" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`

	Jurisdiction *Jurisdiction `json:"jurisdiction,omitempty" keel:"belongs:jurisdiction_id"`
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
