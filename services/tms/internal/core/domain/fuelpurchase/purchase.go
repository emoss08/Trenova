package fuelpurchase

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/worker"
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
	"github.com/uptrace/bun"
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
	_ bun.BeforeAppendModelHook          = (*FuelPurchase)(nil)
	_ domaintypes.PostgresSearchable     = (*FuelPurchase)(nil)
	_ pagination.CursorEntity            = (*FuelPurchase)(nil)
	_ validationframework.TenantedEntity = (*FuelPurchase)(nil)
)

type FuelPurchase struct {
	bun.BaseModel             `bun:"table:fuel_purchases,alias:fpur" json:"-"`
	pagination.CursorValueSet `bun:",embed"                          json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	TractorID      pulid.ID  `json:"tractorId"      bun:"tractor_id,type:VARCHAR(100),notnull"`
	WorkerID       *pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),nullzero"`
	JurisdictionID pulid.ID  `json:"jurisdictionId" bun:"jurisdiction_id,type:VARCHAR(100),notnull"`
	FuelCardID     *pulid.ID `json:"fuelCardId"     bun:"fuel_card_id,type:VARCHAR(100),nullzero"`
	CardLastFour   string    `json:"cardLastFour"   bun:"card_last_four,type:VARCHAR(4),nullzero"`

	PurchasedAt int64  `json:"purchasedAt" bun:"purchased_at,type:BIGINT,notnull"`
	Vendor      string `json:"vendor"      bun:"vendor,type:VARCHAR(150),nullzero"`
	VendorCity  string `json:"vendorCity"  bun:"vendor_city,type:VARCHAR(100),nullzero"`

	FuelType     domaintypes.IFTAFuelType `json:"fuelType"     bun:"fuel_type,type:ifta_fuel_type_enum,notnull"`
	Quantity     decimal.Decimal          `json:"quantity"     bun:"quantity,type:NUMERIC(12,3),notnull"`
	QuantityUnit QuantityUnit             `json:"quantityUnit" bun:"quantity_unit,type:fuel_quantity_unit_enum,notnull,default:'Gallon'"`
	Gallons      decimal.Decimal          `json:"gallons"      bun:"gallons,type:NUMERIC(12,3),notnull"`
	UnitPrice    decimal.NullDecimal      `json:"unitPrice"    bun:"unit_price,type:NUMERIC(19,4),nullzero"`

	TotalAmountMinor int64  `json:"totalAmountMinor" bun:"total_amount_minor,type:BIGINT,notnull"`
	CurrencyCode     string `json:"currencyCode"     bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	Odometer         *int64 `json:"odometer"         bun:"odometer,type:BIGINT,nullzero"`

	TransactionReference string         `json:"transactionReference" bun:"transaction_reference,type:VARCHAR(150),nullzero"`
	Source               PurchaseSource `json:"source"               bun:"source,type:fuel_purchase_source_enum,notnull,default:'Manual'"`
	ImportBatchID        *pulid.ID      `json:"importBatchId"        bun:"import_batch_id,type:VARCHAR(100),nullzero"`
	TaxPaid              bool           `json:"taxPaid"              bun:"tax_paid,type:BOOLEAN,notnull"`
	Notes                string         `json:"notes"                bun:"notes,type:TEXT,nullzero"`
	CreatedByID          pulid.ID       `json:"createdById"          bun:"created_by_id,type:VARCHAR(100),nullzero"`

	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`
	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Tractor      *tractor.Tractor   `json:"tractor,omitempty"      bun:"rel:belongs-to,join:tractor_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Worker       *worker.Worker     `json:"worker,omitempty"       bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	FuelCard     *FuelCard          `json:"fuelCard,omitempty"     bun:"rel:belongs-to,join:fuel_card_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Jurisdiction *ifta.Jurisdiction `json:"jurisdiction,omitempty" bun:"rel:belongs-to,join:jurisdiction_id=id"`
	ImportBatch  *ImportBatch       `json:"importBatch,omitempty"  bun:"rel:belongs-to,join:import_batch_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (p *FuelPurchase) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("fpur_")
		}
		if p.QuantityUnit == "" {
			p.QuantityUnit = QuantityUnitGallon
		}
		if p.Source == "" {
			p.Source = PurchaseSourceManual
		}
		if p.CurrencyCode == "" {
			p.CurrencyCode = money.DefaultCurrencyCode
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
