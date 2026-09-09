package fuelpurchase_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldErrors(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}
	return fields
}

func validPurchase() *fuelpurchase.FuelPurchase {
	return &fuelpurchase.FuelPurchase{
		TractorID:        pulid.MustNew("trac_"),
		JurisdictionID:   pulid.MustNew("ifj_"),
		PurchasedAt:      timeutils.NowUnix() - 3600,
		Vendor:           "Pilot #123",
		VendorCity:       "Amarillo",
		FuelType:         domaintypes.IFTAFuelTypeDiesel,
		Quantity:         decimal.RequireFromString("120.5"),
		QuantityUnit:     fuelpurchase.QuantityUnitGallon,
		Gallons:          decimal.RequireFromString("120.500"),
		TotalAmountMinor: 45_000,
		CurrencyCode:     "USD",
		Source:           fuelpurchase.PurchaseSourceManual,
		TaxPaid:          true,
	}
}

func TestFuelPurchase_ValidPurchasePasses(t *testing.T) {
	t.Parallel()

	purchase := validPurchase()
	purchase.Normalize()

	multiErr := errortypes.NewMultiError()
	purchase.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestFuelPurchase_Normalize(t *testing.T) {
	t.Parallel()

	t.Run("upper-cases and trims the reference and currency", func(t *testing.T) {
		t.Parallel()

		purchase := validPurchase()
		purchase.TransactionReference = "  ab-123xyz "
		purchase.CurrencyCode = " cad "
		purchase.Vendor = "  Love's  "
		purchase.Normalize()

		assert.Equal(t, "AB-123XYZ", purchase.TransactionReference)
		assert.Equal(t, "CAD", purchase.CurrencyCode)
		assert.Equal(t, "Love's", purchase.Vendor)
	})

	t.Run("defaults currency, unit and source", func(t *testing.T) {
		t.Parallel()

		purchase := validPurchase()
		purchase.CurrencyCode = ""
		purchase.QuantityUnit = ""
		purchase.Source = ""
		purchase.Normalize()

		assert.Equal(t, "USD", purchase.CurrencyCode)
		assert.Equal(t, fuelpurchase.QuantityUnitGallon, purchase.QuantityUnit)
		assert.Equal(t, fuelpurchase.PurchaseSourceManual, purchase.Source)
	})

	t.Run("derives gallons from litres", func(t *testing.T) {
		t.Parallel()

		purchase := validPurchase()
		purchase.QuantityUnit = fuelpurchase.QuantityUnitLitre
		purchase.Quantity = decimal.RequireFromString("100")
		purchase.Gallons = decimal.Zero
		purchase.Normalize()

		assert.Equal(t, "26.417", purchase.Gallons.StringFixed(3))
		assert.Equal(t, "100.000", purchase.Quantity.StringFixed(3))
	})

	t.Run("rounds the unit price to four places", func(t *testing.T) {
		t.Parallel()

		purchase := validPurchase()
		purchase.UnitPrice = decimal.NewNullDecimal(decimal.RequireFromString("3.99999"))
		purchase.Normalize()

		assert.Equal(t, "4.0000", purchase.UnitPrice.Decimal.StringFixed(4))
	})
}

func TestFuelPurchase_ValidateRejections(t *testing.T) {
	t.Parallel()

	batchID := pulid.MustNew("fpib_")

	tests := []struct {
		name   string
		mutate func(p *fuelpurchase.FuelPurchase)
		field  string
	}{
		{
			name:   "zero quantity",
			mutate: func(p *fuelpurchase.FuelPurchase) { p.Quantity = decimal.Zero },
			field:  "quantity",
		},
		{
			name:   "zero gallons",
			mutate: func(p *fuelpurchase.FuelPurchase) { p.Gallons = decimal.Zero },
			field:  "gallons",
		},
		{
			name:   "negative amount",
			mutate: func(p *fuelpurchase.FuelPurchase) { p.TotalAmountMinor = -1 },
			field:  "totalAmount",
		},
		{
			name: "negative unit price",
			mutate: func(p *fuelpurchase.FuelPurchase) {
				p.UnitPrice = decimal.NewNullDecimal(decimal.RequireFromString("-0.01"))
			},
			field: "unitPrice",
		},
		{
			name: "future purchase date",
			mutate: func(p *fuelpurchase.FuelPurchase) {
				p.PurchasedAt = timeutils.NowUnix() + 86_400
			},
			field: "purchasedAt",
		},
		{
			name:   "missing purchase date",
			mutate: func(p *fuelpurchase.FuelPurchase) { p.PurchasedAt = 0 },
			field:  "purchasedAt",
		},
		{
			name:   "three-digit last four",
			mutate: func(p *fuelpurchase.FuelPurchase) { p.CardLastFour = "123" },
			field:  "cardLastFour",
		},
		{
			name:   "alphabetic last four",
			mutate: func(p *fuelpurchase.FuelPurchase) { p.CardLastFour = "12AB" },
			field:  "cardLastFour",
		},
		{
			name: "card import without a batch",
			mutate: func(p *fuelpurchase.FuelPurchase) {
				p.Source = fuelpurchase.PurchaseSourceCardImport
			},
			field: "importBatchId",
		},
		{
			name:   "manual purchase with a batch",
			mutate: func(p *fuelpurchase.FuelPurchase) { p.ImportBatchID = &batchID },
			field:  "importBatchId",
		},
		{
			name:   "missing tractor",
			mutate: func(p *fuelpurchase.FuelPurchase) { p.TractorID = pulid.Nil },
			field:  "tractorId",
		},
		{
			name:   "missing jurisdiction",
			mutate: func(p *fuelpurchase.FuelPurchase) { p.JurisdictionID = pulid.Nil },
			field:  "jurisdictionId",
		},
		{
			name:   "unknown fuel type",
			mutate: func(p *fuelpurchase.FuelPurchase) { p.FuelType = "Kerosene" },
			field:  "fuelType",
		},
		{
			name:   "lower-case currency",
			mutate: func(p *fuelpurchase.FuelPurchase) { p.CurrencyCode = "usd" },
			field:  "currencyCode",
		},
		{
			name:   "negative odometer",
			mutate: func(p *fuelpurchase.FuelPurchase) { odo := int64(-5); p.Odometer = &odo },
			field:  "odometer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			purchase := validPurchase()
			tt.mutate(purchase)

			multiErr := errortypes.NewMultiError()
			purchase.Validate(multiErr)

			require.True(t, multiErr.HasErrors(), "expected a validation error")
			assert.Contains(t, fieldErrors(multiErr), tt.field)
		})
	}
}

func TestFuelPurchase_ImportedPurchaseWithBatchPasses(t *testing.T) {
	t.Parallel()

	batchID := pulid.MustNew("fpib_")
	purchase := validPurchase()
	purchase.Source = fuelpurchase.PurchaseSourceCardImport
	purchase.ImportBatchID = &batchID

	multiErr := errortypes.NewMultiError()
	purchase.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.True(t, purchase.IsImported())
}

func TestFuelPurchase_TaxPaidGallons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fuelType domaintypes.IFTAFuelType
		taxPaid  bool
		want     string
	}{
		{"taxed diesel earns credit", domaintypes.IFTAFuelTypeDiesel, true, "120.500"},
		{"untaxed bulk diesel earns nothing", domaintypes.IFTAFuelTypeDiesel, false, "0.000"},
		{"DEF is not motor fuel", domaintypes.IFTAFuelTypeDEF, true, "0.000"},
		{"reefer fuel is not motor fuel", domaintypes.IFTAFuelTypeReefer, true, "0.000"},
		{"taxed CNG earns credit", domaintypes.IFTAFuelTypeCNG, true, "120.500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			purchase := validPurchase()
			purchase.FuelType = tt.fuelType
			purchase.TaxPaid = tt.taxPaid

			assert.Equal(t, tt.want, purchase.TaxPaidGallons().StringFixed(3))
			assert.Equal(t, tt.fuelType.CountsForIFTA(), purchase.CountsForIFTA())
		})
	}
}

func TestFuelPurchase_TotalAmount(t *testing.T) {
	t.Parallel()

	purchase := validPurchase()
	purchase.TotalAmountMinor = 12_345

	assert.Equal(t, "123.45", purchase.TotalAmount().StringFixed(2))
	assert.Equal(t, "fuel_purchases", purchase.GetTableName())
	assert.Equal(t, "fuel_purchase", purchase.GetResourceType())
}
