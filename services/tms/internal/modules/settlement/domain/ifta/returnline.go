package ifta

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*ReturnLine)(nil)
)

// One return line per jurisdiction and fuel type
type ReturnLine struct {
	keel.Table `keel:"ifta_return_lines,report=Equipment,label=IFTA Return Line"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:ifrl_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	ReturnID       pulid.ID `json:"returnId"       keel:"return_id"`
	JurisdictionID pulid.ID `json:"jurisdictionId" keel:"jurisdiction_id"`

	FuelType     domaintypes.IFTAFuelType `json:"fuelType"     keel:"fuel_type"`
	IsIftaMember bool                     `json:"isIftaMember" keel:"is_ifta_member,label=IFTA Member"`

	TotalMiles   decimal.Decimal `json:"totalMiles"   keel:"total_miles,format=distance"`
	TaxableMiles decimal.Decimal `json:"taxableMiles" keel:"taxable_miles,format=distance"`
	RouteMiles   decimal.Decimal `json:"routeMiles"   keel:"route_miles"`
	ManualMiles  decimal.Decimal `json:"manualMiles"  keel:"manual_miles"`
	LoadedMiles  decimal.Decimal `json:"loadedMiles"  keel:"loaded_miles"`
	EmptyMiles   decimal.Decimal `json:"emptyMiles"   keel:"empty_miles"`

	TaxPaidGallons    decimal.Decimal `json:"taxPaidGallons"    keel:"tax_paid_gallons,label=Tax-Paid Gallons"`
	TaxPaidGallonsRaw decimal.Decimal `json:"taxPaidGallonsRaw" keel:"tax_paid_gallons_raw"`
	PurchaseCount     int             `json:"purchaseCount"     keel:"purchase_count"`
	TaxableGallons    decimal.Decimal `json:"taxableGallons"    keel:"taxable_gallons"`
	NetTaxableGallons decimal.Decimal `json:"netTaxableGallons" keel:"net_taxable_gallons"`

	RatePerGallon          decimal.NullDecimal `json:"ratePerGallon"          keel:"rate_per_gallon,label=Rate per Gallon"`
	SurchargeRatePerGallon decimal.NullDecimal `json:"surchargeRatePerGallon" keel:"surcharge_rate_per_gallon,label=Surcharge Rate per Gallon"`
	RateMissing            bool                `json:"rateMissing"            keel:"rate_missing"`

	TaxDueMinor       int64 `json:"taxDueMinor"       keel:"tax_due_minor,label=Tax Due (Minor)"`
	SurchargeDueMinor int64 `json:"surchargeDueMinor" keel:"surcharge_due_minor,label=Surcharge Due (Minor)"`
	LineTotalMinor    int64 `json:"lineTotalMinor"    keel:"line_total_minor,label=Line Total (Minor)"`
	SortOrder         int   `json:"sortOrder"         keel:"sort_order,dimension"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Return       *Return       `json:"-"                      keel:"belongs:return_id"`
	Jurisdiction *Jurisdiction `json:"jurisdiction,omitempty" keel:"belongs:jurisdiction_id,noreport"`
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
