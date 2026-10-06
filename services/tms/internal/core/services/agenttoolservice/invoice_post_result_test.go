package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostedInvoiceResult_NamesTheInvoiceAndWhereItStands(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("inv_")
	result := postedInvoiceResult(&invoice.Invoice{
		ID:               id,
		Number:           "INV-1001",
		Status:           invoice.StatusPosted,
		TotalAmount:      decimal.NewFromInt(2300),
		CurrencyCode:     "USD",
		BillToName:       "FreshHaul Foods",
		SettlementStatus: invoice.SettlementStatusUnpaid,
	})

	require.NotNil(t, result.Record)
	assert.Equal(t, id.String(), result.Record.ID)
	assert.Equal(t, "INV-1001", result.Name)
	assert.Equal(t, "Posted, 2300.00 USD to FreshHaul Foods, unpaid", result.State)
}
