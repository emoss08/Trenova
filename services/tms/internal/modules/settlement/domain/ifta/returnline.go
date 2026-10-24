package ifta

import (
	"context"

	"github.com/emoss08/keel"
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
	keel.Table    `keel:"ifta_return_lines"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:ifrl_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	ReturnID       pulid.ID `json:"returnId"       bun:"return_id,type:VARCHAR(100),notnull" keel:"return_id"`
	JurisdictionID pulid.ID `json:"jurisdictionId" bun:"jurisdiction_id,type:VARCHAR(100),notnull" keel:"jurisdiction_id"`

	FuelType     domaintypes.IFTAFuelType `json:"fuelType"     bun:"fuel_type,type:ifta_fuel_type_enum,notnull" keel:"fuel_type"`
	IsIftaMember bool                     `json:"isIftaMember" bun:"is_ifta_member,type:BOOLEAN,notnull" keel:"is_ifta_member"`

	TotalMiles   decimal.Decimal `json:"totalMiles"   bun:"total_miles,type:NUMERIC(12,0),notnull" keel:"total_miles"`
	TaxableMiles decimal.Decimal `json:"taxableMiles" bun:"taxable_miles,type:NUMERIC(12,0),notnull" keel:"taxable_miles"`
	RouteMiles   decimal.Decimal `json:"routeMiles"   bun:"route_miles,type:NUMERIC(12,2),notnull" keel:"route_miles"`
	ManualMiles  decimal.Decimal `json:"manualMiles"  bun:"manual_miles,type:NUMERIC(12,2),notnull" keel:"manual_miles"`
	LoadedMiles  decimal.Decimal `json:"loadedMiles"  bun:"loaded_miles,type:NUMERIC(12,2),notnull" keel:"loaded_miles"`
	EmptyMiles   decimal.Decimal `json:"emptyMiles"   bun:"empty_miles,type:NUMERIC(12,2),notnull" keel:"empty_miles"`

	TaxPaidGallons    decimal.Decimal `json:"taxPaidGallons"    bun:"tax_paid_gallons,type:NUMERIC(12,0),notnull" keel:"tax_paid_gallons"`
	TaxPaidGallonsRaw decimal.Decimal `json:"taxPaidGallonsRaw" bun:"tax_paid_gallons_raw,type:NUMERIC(12,3),notnull" keel:"tax_paid_gallons_raw"`
	PurchaseCount     int             `json:"purchaseCount"     bun:"purchase_count,type:INTEGER,notnull,default:0" keel:"purchase_count"`
	TaxableGallons    decimal.Decimal `json:"taxableGallons"    bun:"taxable_gallons,type:NUMERIC(12,0),notnull" keel:"taxable_gallons"`
	NetTaxableGallons decimal.Decimal `json:"netTaxableGallons" bun:"net_taxable_gallons,type:NUMERIC(12,0),notnull" keel:"net_taxable_gallons"`

	RatePerGallon          decimal.NullDecimal `json:"ratePerGallon"          bun:"rate_per_gallon,type:NUMERIC(10,4),nullzero" keel:"rate_per_gallon"`
	SurchargeRatePerGallon decimal.NullDecimal `json:"surchargeRatePerGallon" bun:"surcharge_rate_per_gallon,type:NUMERIC(10,4),nullzero" keel:"surcharge_rate_per_gallon"`
	RateMissing            bool                `json:"rateMissing"            bun:"rate_missing,type:BOOLEAN,notnull" keel:"rate_missing"`

	TaxDueMinor       int64 `json:"taxDueMinor"       bun:"tax_due_minor,type:BIGINT,notnull" keel:"tax_due_minor"`
	SurchargeDueMinor int64 `json:"surchargeDueMinor" bun:"surcharge_due_minor,type:BIGINT,notnull" keel:"surcharge_due_minor"`
	LineTotalMinor    int64 `json:"lineTotalMinor"    bun:"line_total_minor,type:BIGINT,notnull" keel:"line_total_minor"`
	SortOrder         int   `json:"sortOrder"         bun:"sort_order,type:INTEGER,notnull,default:0" keel:"sort_order"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Return       *Return       `json:"-"                      bun:"rel:belongs-to,join:return_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:return_id"`
	Jurisdiction *Jurisdiction `json:"jurisdiction,omitempty" bun:"rel:belongs-to,join:jurisdiction_id=id" keel:"belongs:jurisdiction_id"`
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
