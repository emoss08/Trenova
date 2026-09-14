package invoice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestInvoiceLineCopyChargeDetail(t *testing.T) {
	t.Parallel()

	source := &invoice.InvoiceLine{
		AccessorialChargeID: pulid.MustNew("acc_"),
		ChargeCode:          "DET",
		ChargeMethod:        accessorialcharge.MethodPerUnit,
		RateUnit:            accessorialcharge.RateUnitHour,
		Rate:                decimal.NewNullDecimal(decimal.RequireFromString("75")),
		RateBasisAmount:     decimal.NewNullDecimal(decimal.RequireFromString("2450")),
		FormulaTemplateName: "Per Mile",
		Description:         "Detention",
		Amount:              decimal.RequireFromString("150"),
	}
	target := &invoice.InvoiceLine{Description: "Credit", Amount: decimal.RequireFromString("-150")}

	target.CopyChargeDetail(source)
	target.CopyChargeDetail(nil)

	assert.Equal(t, source.AccessorialChargeID, target.AccessorialChargeID)
	assert.Equal(t, "DET", target.ChargeCode)
	assert.Equal(t, accessorialcharge.MethodPerUnit, target.ChargeMethod)
	assert.Equal(t, accessorialcharge.RateUnitHour, target.RateUnit)
	assert.True(t, target.Rate.Decimal.Equal(source.Rate.Decimal))
	assert.True(t, target.RateBasisAmount.Decimal.Equal(source.RateBasisAmount.Decimal))
	assert.Equal(t, "Per Mile", target.FormulaTemplateName)
	assert.Equal(t, "Credit", target.Description)
	assert.True(t, target.Amount.Equal(decimal.RequireFromString("-150")))
}
