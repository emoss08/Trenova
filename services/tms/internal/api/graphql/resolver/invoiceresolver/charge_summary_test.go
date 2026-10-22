package invoiceresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChargeSummarySumsTheLinesTheInvoiceWasReadWith(t *testing.T) {
	t.Parallel()

	template := pulid.MustNew("ft_")
	obj := &invoice.Invoice{Lines: []*invoice.InvoiceLine{
		{
			Type: invoice.InvoiceLineTypeFreight, Amount: decimal.RequireFromString("900"),
			Quantity: decimal.NewFromInt(1), FormulaTemplateID: template, FormulaTemplateName: "Per mile",
		},
		{
			Type: invoice.InvoiceLineTypeAccessorial, Amount: decimal.RequireFromString("150"),
			Quantity: decimal.NewFromInt(2), ChargeCode: "DET",
			ChargeMethod: accessorialcharge.MethodPerUnit, RateUnit: "$/h",
			Rate: decimal.NewNullDecimal(decimal.RequireFromString("75")),
		},
	}}

	groups, err := (&InvoiceResolver{&Deps{}}).ChargeSummary(t.Context(), obj)
	require.NoError(t, err)
	require.Len(t, groups, 2)

	group := &InvoiceChargeGroupResolver{&Deps{}}
	freightID, err := group.FormulaTemplateID(t.Context(), groups[0])
	require.NoError(t, err)
	require.NotNil(t, freightID)
	assert.Equal(t, template.String(), *freightID)
	freightMethod, err := group.ChargeMethod(t.Context(), groups[0])
	require.NoError(t, err)
	assert.Nil(t, freightMethod, "freight is rated by its template, not charged by a method")
	freightCode, err := group.ChargeCode(t.Context(), groups[0])
	require.NoError(t, err)
	assert.Nil(t, freightCode)

	unit, err := group.RateUnit(t.Context(), groups[1])
	require.NoError(t, err)
	require.NotNil(t, unit)
	assert.Equal(t, "$/h", *unit)
	detentionID, err := group.FormulaTemplateID(t.Context(), groups[1])
	require.NoError(t, err)
	assert.Nil(t, detentionID)
	rate, err := group.Rate(t.Context(), groups[1])
	require.NoError(t, err)
	require.NotNil(t, rate)
	assert.Equal(t, "75", *rate)
	quantity, err := group.Quantity(t.Context(), groups[1])
	require.NoError(t, err)
	assert.Equal(t, "2", quantity)
}

func TestChargeSummaryOfAnInvoiceWithNoLinesIsEmpty(t *testing.T) {
	t.Parallel()

	groups, err := (&InvoiceResolver{&Deps{}}).ChargeSummary(
		t.Context(),
		&invoice.Invoice{Lines: []*invoice.InvoiceLine{}},
	)
	require.NoError(t, err)
	assert.Empty(t, groups)
	assert.NotNil(t, groups, "a non-null list, as the schema promises")
}
