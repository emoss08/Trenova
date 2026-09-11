package invoiceservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func minorUnitErrors(entity *invoice.Invoice) []string {
	multiErr := errortypes.NewMultiError()
	validateMinorUnitTotals(entity, multiErr)

	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}

	return fields
}

func TestValidateMinorUnitTotalsAcceptsConsistentAmounts(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		TotalAmount:      decimal.RequireFromString("350.50"),
		TotalAmountMinor: 35050,
		Lines: []*invoice.InvoiceLine{
			{Amount: decimal.RequireFromString("100.00"), AmountMinor: 10000},
			{Amount: decimal.RequireFromString("250.50"), AmountMinor: 25050},
		},
	}

	assert.Empty(t, minorUnitErrors(entity))
}

func TestValidateMinorUnitTotalsCatchesABadConversion(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		TotalAmount:      decimal.RequireFromString("350.50"),
		TotalAmountMinor: 3505, // dropped a zero
		Lines: []*invoice.InvoiceLine{
			{Amount: decimal.RequireFromString("350.50"), AmountMinor: 3505},
		},
	}

	require.Len(t, minorUnitErrors(entity), 1)
	assert.Equal(t, "totalAmountMinor", minorUnitErrors(entity)[0])
}

func TestValidateMinorUnitTotalsCatchesPerLineDrift(t *testing.T) {
	t.Parallel()

	// The header total rounds cleanly, so a check against the rounded decimal
	// alone would pass. The lines do not add up to it, which is the whole reason
	// the second check exists: this is what posts to the general ledger.
	entity := &invoice.Invoice{
		TotalAmount:      decimal.RequireFromString("100.00"),
		TotalAmountMinor: 10000,
		Lines: []*invoice.InvoiceLine{
			{Amount: decimal.RequireFromString("50.00"), AmountMinor: 5000},
			{Amount: decimal.RequireFromString("50.00"), AmountMinor: 4995},
		},
	}

	require.Len(t, minorUnitErrors(entity), 1)
	assert.Equal(t, "totalAmountMinor", minorUnitErrors(entity)[0])
}

func TestSyncMinorAmountsSatisfiesTheAssertion(t *testing.T) {
	t.Parallel()

	// Whatever the builder produces must pass its own validator, including at the
	// half-cent boundaries where bankers' rounding bites.
	entity := &invoice.Invoice{
		Lines: []*invoice.InvoiceLine{
			{Amount: decimal.RequireFromString("10.005")},
			{Amount: decimal.RequireFromString("10.015")},
			{Amount: decimal.RequireFromString("10.025")},
		},
	}
	syncInvoiceTotalsFromLines(entity)
	entity.SyncMinorAmounts()

	assert.Empty(t, minorUnitErrors(entity))
}
