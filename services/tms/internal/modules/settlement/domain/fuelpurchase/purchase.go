package fuelpurchase

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const (
	PurchaseFutureToleranceSeconds = int64(5 * 60)

	maxVendorLength               = 150
	maxVendorCityLength           = 100
	maxTransactionReferenceLength = 150
	quantityScale                 = 3
	unitPriceScale                = 4
)

var (
	_ domaintypes.PostgresSearchable     = (*FuelPurchase)(nil)
	_ pagination.CursorEntity            = (*FuelPurchase)(nil)
	_ validationframework.TenantedEntity = (*FuelPurchase)(nil)
)

// Fuel bought at the pump or on a fuel card
type FuelPurchase struct {
	keel.Table `                                               keel:"fuel_purchases,report=Equipment"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:fpur_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	TractorID      pulid.ID  `json:"tractorId" keel:"tractor_id"`
	WorkerID       *pulid.ID `json:"workerId" keel:"worker_id"`
	JurisdictionID pulid.ID  `json:"jurisdictionId" keel:"jurisdiction_id"`
	FuelCardID     *pulid.ID `json:"fuelCardId" keel:"fuel_card_id"`
	CardLastFour   string    `json:"cardLastFour" keel:"card_last_four,nullzero"`

	PurchasedAt int64  `json:"purchasedAt" keel:"purchased_at"`
	Vendor      string `json:"vendor" keel:"vendor,nullzero"`
	VendorCity  string `json:"vendorCity" keel:"vendor_city,nullzero"`

	FuelType     domaintypes.IFTAFuelType `json:"fuelType" keel:"fuel_type"`
	Quantity     decimal.Decimal          `json:"quantity" keel:"quantity"`
	QuantityUnit QuantityUnit             `json:"quantityUnit" keel:"quantity_unit"`
	Gallons      decimal.Decimal          `json:"gallons" keel:"gallons"`
	UnitPrice    decimal.NullDecimal      `json:"unitPrice" keel:"unit_price"`

	TotalAmountMinor int64  `json:"totalAmountMinor" keel:"total_amount_minor,label=Total Amount (Minor)"`
	CurrencyCode     string `json:"currencyCode" keel:"currency_code,label=Currency"`
	Odometer         *int64 `json:"odometer" keel:"odometer,format=distance"`

	TransactionReference string         `json:"transactionReference" keel:"transaction_reference,nullzero"`
	Source               PurchaseSource `json:"source" keel:"source"`
	ImportBatchID        *pulid.ID      `json:"importBatchId" keel:"import_batch_id"`
	TaxPaid              bool           `json:"taxPaid" keel:"tax_paid"`
	Notes                string         `json:"notes" keel:"notes,nullzero"`
	CreatedByID          pulid.ID       `json:"createdById" keel:"created_by_id,nullzero"`

	SearchVector string `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-" keel:"rank,computed"`
	Version      int64  `json:"version" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`

	Tractor      *tractor.Tractor   `json:"tractor,omitempty" keel:"belongs:tractor_id"`
	Worker       *worker.Worker     `json:"worker,omitempty" keel:"belongs:worker_id,label=Driver"`
	FuelCard     *FuelCard          `json:"fuelCard,omitempty" keel:"belongs:fuel_card_id"`
	Jurisdiction *ifta.Jurisdiction `json:"jurisdiction,omitempty" keel:"belongs:jurisdiction_id,noreport"`
	ImportBatch  *ImportBatch       `json:"importBatch,omitempty" keel:"belongs:import_batch_id,noreport"`
}

func (p *FuelPurchase) Normalize() {
	p.Vendor = strings.TrimSpace(p.Vendor)
	p.VendorCity = strings.TrimSpace(p.VendorCity)
	p.CardLastFour = strings.TrimSpace(p.CardLastFour)
	p.TransactionReference = strings.ToUpper(strings.TrimSpace(p.TransactionReference))
	p.Notes = strings.TrimSpace(p.Notes)

	p.CurrencyCode = strings.ToUpper(strings.TrimSpace(p.CurrencyCode))
	if p.CurrencyCode == "" {
		p.CurrencyCode = money.DefaultCurrencyCode
	}
	if p.QuantityUnit == "" {
		p.QuantityUnit = QuantityUnitGallon
	}
	if p.Source == "" {
		p.Source = PurchaseSourceManual
	}

	p.Quantity = p.Quantity.Round(quantityScale)
	p.Gallons = p.QuantityUnit.ToGallons(p.Quantity)
	if p.UnitPrice.Valid {
		p.UnitPrice.Decimal = p.UnitPrice.Decimal.Round(unitPriceScale)
	}
}

func (p *FuelPurchase) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		p,
		validation.Field(&p.TractorID, validation.Required.Error("Tractor is required")),
		validation.Field(
			&p.JurisdictionID,
			validation.Required.Error("Jurisdiction is required"),
		),
		validation.Field(
			&p.CardLastFour,
			validation.Match(domaintypes.CardLastFourRegex).
				Error("Card last four must be exactly four digits"),
		),
		validation.Field(
			&p.Vendor,
			validation.Length(0, maxVendorLength).Error("Vendor cannot exceed 150 characters"),
		),
		validation.Field(
			&p.VendorCity,
			validation.Length(0, maxVendorCityLength).
				Error("Vendor city cannot exceed 100 characters"),
		),
		validation.Field(
			&p.FuelType,
			validation.Required.Error("Fuel type is required"),
			domainvalidation.ValidEnum[domaintypes.IFTAFuelType]("Fuel type is not valid"),
		),
		validation.Field(
			&p.QuantityUnit,
			validation.Required.Error("Quantity unit is required"),
			domainvalidation.ValidEnum[QuantityUnit]("Quantity unit must be Gallon or Litre"),
		),
		validation.Field(
			&p.CurrencyCode,
			validation.Required.Error("Currency is required"),
			validation.Match(domaintypes.CurrencyCodeRegex).
				Error("Currency must be a three-letter ISO code"),
		),
		validation.Field(
			&p.TransactionReference,
			validation.Length(0, maxTransactionReferenceLength).
				Error("Transaction reference cannot exceed 150 characters"),
		),
		validation.Field(
			&p.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[PurchaseSource]("Source is not valid"),
		),
	))

	p.validateTiming(multiErr)
	p.validateQuantities(multiErr)
	p.validateProvenance(multiErr)
}

func (p *FuelPurchase) validateTiming(multiErr *errortypes.MultiError) {
	if p.PurchasedAt <= 0 {
		multiErr.Add("purchasedAt", errortypes.ErrRequired, "Purchase date is required")
		return
	}
	if p.PurchasedAt > timeutils.NowUnix()+PurchaseFutureToleranceSeconds {
		multiErr.Add(
			"purchasedAt",
			errortypes.ErrInvalid,
			"Purchase date cannot be in the future",
		)
	}
}

func (p *FuelPurchase) validateQuantities(multiErr *errortypes.MultiError) {
	if !p.Quantity.IsPositive() {
		multiErr.Add("quantity", errortypes.ErrInvalid, "Quantity must be greater than zero")
	}
	if !p.Gallons.IsPositive() {
		multiErr.Add("gallons", errortypes.ErrInvalid, "Gallons must be greater than zero")
	}
	if p.UnitPrice.Valid && p.UnitPrice.Decimal.IsNegative() {
		multiErr.Add("unitPrice", errortypes.ErrInvalid, "Unit price cannot be negative")
	}
	if p.TotalAmountMinor < 0 {
		multiErr.Add("totalAmount", errortypes.ErrInvalid, "Total amount cannot be negative")
	}
	if p.Odometer != nil && *p.Odometer < 0 {
		multiErr.Add("odometer", errortypes.ErrInvalid, "Odometer cannot be negative")
	}
}

func (p *FuelPurchase) validateProvenance(multiErr *errortypes.MultiError) {
	hasBatch := p.ImportBatchID != nil && !p.ImportBatchID.IsNil()

	switch {
	case p.Source == PurchaseSourceCardImport && !hasBatch:
		multiErr.Add(
			"importBatchId",
			errortypes.ErrRequired,
			"An imported purchase must reference the import it came from",
		)
	case p.Source == PurchaseSourceManual && hasBatch:
		multiErr.Add(
			"importBatchId",
			errortypes.ErrInvalid,
			"A manual purchase cannot reference an import",
		)
	}
}

func (p *FuelPurchase) TotalAmount() decimal.Decimal {
	return money.DecimalFromMinor(p.TotalAmountMinor)
}

func (p *FuelPurchase) CountsForIFTA() bool { return p.FuelType.CountsForIFTA() }

func (p *FuelPurchase) TaxPaidGallons() decimal.Decimal {
	if !p.TaxPaid || !p.CountsForIFTA() {
		return decimal.Zero
	}
	return p.Gallons
}

func (p *FuelPurchase) IsImported() bool { return p.Source == PurchaseSourceCardImport }

func (p *FuelPurchase) GetID() pulid.ID { return p.ID }

func (p *FuelPurchase) GetCreatedAt() int64 { return p.CreatedAt }

func (p *FuelPurchase) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *FuelPurchase) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *FuelPurchase) GetTableName() string { return "fuel_purchases" }

func (p *FuelPurchase) GetResourceType() string { return "fuel_purchase" }

func (p *FuelPurchase) GetResourceID() string { return p.ID.String() }

func (p *FuelPurchase) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "fpur",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "vendor", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "transaction_reference",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "vendor_city",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (p *FuelPurchase) BeforeCreate(ctx context.Context) error {
	if p.QuantityUnit == "" {
		p.QuantityUnit = QuantityUnitGallon
	}
	if p.CurrencyCode == "" {
		p.CurrencyCode = "USD"
	}
	if p.Source == "" {
		p.Source = PurchaseSourceManual
	}
	if p.CurrencyCode == "" {
		p.CurrencyCode = money.DefaultCurrencyCode
	}

	return nil
}
