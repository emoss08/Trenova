package ifta

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*ReturnLine)(nil)
	_ validationframework.TenantedEntity = (*ReturnLine)(nil)
)

type ReturnLine struct {
	bun.BaseModel `bun:"table:ifta_return_lines,alias:ifrl" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ReturnID       pulid.ID `json:"returnId"       bun:"return_id,type:VARCHAR(100),notnull"`
	JurisdictionID pulid.ID `json:"jurisdictionId" bun:"jurisdiction_id,type:VARCHAR(100),notnull"`

	FuelType     domaintypes.IFTAFuelType `json:"fuelType"     bun:"fuel_type,type:ifta_fuel_type_enum,notnull"`
	IsIftaMember bool                     `json:"isIftaMember" bun:"is_ifta_member,type:BOOLEAN,notnull"`

	TotalMiles   decimal.Decimal `json:"totalMiles"   bun:"total_miles,type:NUMERIC(12,0),notnull"`
	TaxableMiles decimal.Decimal `json:"taxableMiles" bun:"taxable_miles,type:NUMERIC(12,0),notnull"`
	RouteMiles   decimal.Decimal `json:"routeMiles"   bun:"route_miles,type:NUMERIC(12,2),notnull"`
	ManualMiles  decimal.Decimal `json:"manualMiles"  bun:"manual_miles,type:NUMERIC(12,2),notnull"`
	LoadedMiles  decimal.Decimal `json:"loadedMiles"  bun:"loaded_miles,type:NUMERIC(12,2),notnull"`
	EmptyMiles   decimal.Decimal `json:"emptyMiles"   bun:"empty_miles,type:NUMERIC(12,2),notnull"`

	TaxPaidGallons    decimal.Decimal `json:"taxPaidGallons"    bun:"tax_paid_gallons,type:NUMERIC(12,0),notnull"`
	TaxPaidGallonsRaw decimal.Decimal `json:"taxPaidGallonsRaw" bun:"tax_paid_gallons_raw,type:NUMERIC(12,3),notnull"`
	PurchaseCount     int             `json:"purchaseCount"     bun:"purchase_count,type:INTEGER,notnull,default:0"`
	TaxableGallons    decimal.Decimal `json:"taxableGallons"    bun:"taxable_gallons,type:NUMERIC(12,0),notnull"`
	NetTaxableGallons decimal.Decimal `json:"netTaxableGallons" bun:"net_taxable_gallons,type:NUMERIC(12,0),notnull"`

	RatePerGallon          decimal.NullDecimal `json:"ratePerGallon"          bun:"rate_per_gallon,type:NUMERIC(10,4),nullzero"`
	SurchargeRatePerGallon decimal.NullDecimal `json:"surchargeRatePerGallon" bun:"surcharge_rate_per_gallon,type:NUMERIC(10,4),nullzero"`
	RateMissing            bool                `json:"rateMissing"            bun:"rate_missing,type:BOOLEAN,notnull"`

	TaxDueMinor       int64 `json:"taxDueMinor"       bun:"tax_due_minor,type:BIGINT,notnull"`
	SurchargeDueMinor int64 `json:"surchargeDueMinor" bun:"surcharge_due_minor,type:BIGINT,notnull"`
	LineTotalMinor    int64 `json:"lineTotalMinor"    bun:"line_total_minor,type:BIGINT,notnull"`
	SortOrder         int   `json:"sortOrder"         bun:"sort_order,type:INTEGER,notnull,default:0"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Return       *Return       `json:"-"                      bun:"rel:belongs-to,join:return_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Jurisdiction *Jurisdiction `json:"jurisdiction,omitempty" bun:"rel:belongs-to,join:jurisdiction_id=id"`
}

func (l *ReturnLine) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(l,
		validation.Field(&l.ReturnID, validation.Required.Error("Return is required")),
		validation.Field(&l.JurisdictionID,
			validation.Required.Error("Jurisdiction is required"),
		),
		validation.Field(&l.FuelType,
			validation.Required.Error("Fuel type is required"),
			domainvalidation.ValidEnum[domaintypes.IFTAFuelType]("Fuel type is not valid"),
		),
		validation.Field(&l.PurchaseCount,
			validation.Min(0).Error("Purchase count cannot be negative"),
		),
	))

	if l.TotalMiles.IsNegative() || l.TaxableMiles.IsNegative() || l.RouteMiles.IsNegative() ||
		l.ManualMiles.IsNegative() || l.LoadedMiles.IsNegative() || l.EmptyMiles.IsNegative() {
		multiErr.Add("totalMiles", errortypes.ErrInvalid, "Miles cannot be negative")
	}
	if l.TaxPaidGallons.IsNegative() || l.TaxPaidGallonsRaw.IsNegative() ||
		l.TaxableGallons.IsNegative() {
		multiErr.Add("taxPaidGallons", errortypes.ErrInvalid, "Gallons cannot be negative")
	}
	if l.SurchargeDueMinor < 0 {
		multiErr.Add(
			"surchargeDueMinor",
			errortypes.ErrInvalid,
			"Surcharge is charged on gallons consumed and can never be a credit",
		)
	}
	if !l.RateMissing && !l.RatePerGallon.Valid {
		multiErr.Add(
			"ratePerGallon",
			errortypes.ErrRequired,
			"A line must carry a rate unless the rate is flagged missing",
		)
	}
	if l.LineTotalMinor != l.TaxDueMinor+l.SurchargeDueMinor {
		multiErr.Add(
			"lineTotalMinor",
			errortypes.ErrInvalid,
			"Line total must equal tax due plus surcharge",
		)
	}
}

func (l *ReturnLine) TaxDue() decimal.Decimal { return money.DecimalFromMinor(l.TaxDueMinor) }

func (l *ReturnLine) SurchargeDue() decimal.Decimal {
	return money.DecimalFromMinor(l.SurchargeDueMinor)
}

func (l *ReturnLine) LineTotal() decimal.Decimal {
	return money.DecimalFromMinor(l.LineTotalMinor)
}

func (l *ReturnLine) IsCredit() bool { return l.LineTotalMinor < 0 }

func (l *ReturnLine) RateKey() RateKey {
	return RateKey{JurisdictionID: l.JurisdictionID, FuelType: l.FuelType}
}

func (l *ReturnLine) GetID() pulid.ID { return l.ID }

func (l *ReturnLine) GetCreatedAt() int64 { return l.CreatedAt }

func (l *ReturnLine) GetOrganizationID() pulid.ID { return l.OrganizationID }

func (l *ReturnLine) GetBusinessUnitID() pulid.ID { return l.BusinessUnitID }

func (l *ReturnLine) GetTableName() string { return "ifta_return_lines" }

func (l *ReturnLine) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("ifrl_")
		}
		l.CreatedAt = now
		l.UpdatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}

	return nil
}
