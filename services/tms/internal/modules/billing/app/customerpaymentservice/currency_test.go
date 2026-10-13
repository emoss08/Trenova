package customerpaymentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCreditTargetRefusesAnotherCurrency(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	memo := &invoice.Invoice{
		CustomerID: customerID, BillType: billingqueue.BillTypeCreditMemo,
		Status: invoice.StatusPosted, CurrencyCode: "CAD", TotalAmountMinor: -5_000,
	}
	target := &invoice.Invoice{
		Number: "INV-9", CustomerID: customerID, BillType: billingqueue.BillTypeInvoice,
		Status: invoice.StatusPosted, CurrencyCode: "USD", TotalAmountMinor: 10_000,
	}

	multiErr := validateCreditTarget(memo, target, 1_000, 0)
	require.NotNil(t, multiErr)
	assert.Contains(t, multiErr.Error(), "billed in USD, but this credit memo is in CAD")

	target.CurrencyCode = "CAD"
	assert.Nil(t, validateCreditTarget(memo, target, 1_000, 0))
}
