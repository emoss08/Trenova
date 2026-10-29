package invoicechargegroup_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicechargegroup"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func amount(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func rate(value string) decimal.NullDecimal {
	return decimal.NewNullDecimal(amount(value))
}

func TestChargeGroupsSumEveryLineBilledTheSameWay(t *testing.T) {
	t.Parallel()

	perMile := pulid.MustNew("ft_")
	perCWT := pulid.MustNew("ft_")
	fuel := pulid.MustNew("acc_")
	detention := pulid.MustNew("acc_")
	first, second, third := pulid.MustNew("shp_"), pulid.MustNew("shp_"), pulid.MustNew("shp_")

	lines := []*invoice.InvoiceLine{
		{
			ShipmentID: first, LineNumber: 1, Type: invoice.InvoiceLineTypeFreight,
			Description: "Freight charge", Quantity: amount("1"), Amount: amount("1200"),
			Rate: rate("3.75"), FormulaTemplateID: perMile, FormulaTemplateName: "Per mile",
		},
		{
			ShipmentID: first, LineNumber: 2, Type: invoice.InvoiceLineTypeAccessorial,
			Description: "Fuel surcharge", Quantity: amount("1"), Amount: amount("138"),
			AccessorialChargeID: fuel, ChargeCode: "FSC",
			ChargeMethod: accessorialcharge.MethodPercentage, Rate: rate("11.5"),
			RateBasisAmount: rate("1200"),
		},
		{
			ShipmentID: second, LineNumber: 3, Type: invoice.InvoiceLineTypeFreight,
			Description: "Freight charge", Quantity: amount("1"), Amount: amount("900"),
			Rate: rate("3.60"), FormulaTemplateID: perMile, FormulaTemplateName: "Per mile",
		},
		{
			ShipmentID: second, LineNumber: 4, Type: invoice.InvoiceLineTypeAccessorial,
			Description: "Fuel surcharge", Quantity: amount("1"), Amount: amount("103.50"),
			AccessorialChargeID: fuel, ChargeCode: "FSC",
			ChargeMethod: accessorialcharge.MethodPercentage, Rate: rate("11.5"),
			RateBasisAmount: rate("900"),
		},
		{
			ShipmentID: second, LineNumber: 5, Type: invoice.InvoiceLineTypeAccessorial,
			Description: "Detention", Quantity: amount("2"), Amount: amount("150"),
			AccessorialChargeID: detention, ChargeCode: "DET",
			ChargeMethod: accessorialcharge.MethodPerUnit, RateUnit: "$/h",
			Rate: rate("75"),
		},
		{
			ShipmentID: third, LineNumber: 6, Type: invoice.InvoiceLineTypeFreight,
			Description: "Freight charge", Quantity: amount("1"), Amount: amount("640"),
			Rate: rate("3.05"), FormulaTemplateID: perCWT, FormulaTemplateName: "Per CWT",
		},
		{
			ShipmentID: third, LineNumber: 7, Type: invoice.InvoiceLineTypeAccessorial,
			Description: "Detention", Quantity: amount("4"), Amount: amount("300"),
			AccessorialChargeID: detention, ChargeCode: "DET",
			ChargeMethod: accessorialcharge.MethodPerUnit, RateUnit: "$/h",
			Rate: rate("75"),
		},
	}

	groups := invoicechargegroup.ChargeGroups(lines)
	require.Len(t, groups, 4)

	mile := groups[0]
	assert.Equal(t, invoice.InvoiceLineTypeFreight, mile.Type)
	assert.Equal(t, perMile, mile.FormulaTemplateID)
	assert.Equal(t, "Per mile", mile.FormulaTemplateName)
	assert.Equal(t, 2, mile.LineCount)
	assert.Equal(t, 2, mile.ShipmentCount)
	assert.True(t, mile.Amount.Equal(amount("2100")))
	assert.False(t, mile.Rate.Valid, "each load had its own base rate")
	assert.False(t, mile.RateBasisAmount.Valid)

	cwt := groups[1]
	assert.Equal(t, perCWT, cwt.FormulaTemplateID)
	assert.Equal(t, 1, cwt.LineCount)
	assert.True(t, cwt.Rate.Decimal.Equal(amount("3.05")), "one line keeps its rate")

	fsc := groups[2]
	assert.Equal(t, invoice.InvoiceLineTypeAccessorial, fsc.Type)
	assert.Equal(t, "FSC", fsc.ChargeCode)
	assert.Equal(t, accessorialcharge.MethodPercentage, fsc.ChargeMethod)
	assert.Equal(t, 2, fsc.ShipmentCount)
	assert.True(t, fsc.Rate.Decimal.Equal(amount("11.5")), "the same percentage on every load")
	assert.False(t, fsc.RateBasisAmount.Valid, "two loads had two bases")
	assert.True(t, fsc.Amount.Equal(amount("241.50")))

	det := groups[3]
	assert.Equal(t, "$/h", det.RateUnit)
	assert.True(t, det.Quantity.Equal(amount("6")), "hours add up across loads")
	assert.True(t, det.Rate.Decimal.Equal(amount("75")))
	assert.True(t, det.Amount.Equal(amount("450")))
	assert.Equal(t, "Detention", det.Description)
}

func TestChargeGroupsKeepFreightBeforeAccessorialsBeforeMemos(t *testing.T) {
	t.Parallel()

	lines := []*invoice.InvoiceLine{
		{
			LineNumber:  1,
			Type:        invoice.InvoiceLineTypeMemo,
			Description: "Goodwill credit",
			Amount:      amount("-50"),
		},
		{
			LineNumber:  2,
			Type:        invoice.InvoiceLineTypeAccessorial,
			Description: "Lumper",
			ChargeCode:  "LUMP",
			Amount:      amount("185"),
		},
		{
			LineNumber:  3,
			Type:        invoice.InvoiceLineTypeFreight,
			Description: "Freight charge",
			Amount:      amount("900"),
		},
		{
			LineNumber:  4,
			Type:        invoice.InvoiceLineTypeMemo,
			Description: "Rate correction",
			Amount:      amount("-25"),
		},
	}

	groups := invoicechargegroup.ChargeGroups(lines)
	require.Len(t, groups, 4)

	assert.Equal(t, invoice.InvoiceLineTypeFreight, groups[0].Type)
	assert.Equal(t, invoice.InvoiceLineTypeAccessorial, groups[1].Type)
	assert.Equal(t, "Goodwill credit", groups[2].Description)
	assert.Equal(
		t,
		"Rate correction",
		groups[3].Description,
		"memo lines are told apart by what they say",
	)
	assert.Equal(t, 0, groups[2].ShipmentCount, "a memo line bills no shipment")
}

func TestChargeGroupsTellLegacyAccessorialsApartByTheirDescription(t *testing.T) {
	t.Parallel()

	lines := []*invoice.InvoiceLine{
		{
			LineNumber:  1,
			Type:        invoice.InvoiceLineTypeAccessorial,
			Description: "Lumper",
			Amount:      amount("185"),
		},
		{
			LineNumber:  2,
			Type:        invoice.InvoiceLineTypeAccessorial,
			Description: "Tarp",
			Amount:      amount("75"),
		},
		{
			LineNumber:  3,
			Type:        invoice.InvoiceLineTypeAccessorial,
			Description: "Lumper",
			Amount:      amount("120"),
		},
	}

	groups := invoicechargegroup.ChargeGroups(lines)
	require.Len(t, groups, 2)

	assert.Equal(t, "Lumper", groups[0].Description)
	assert.Equal(t, 2, groups[0].LineCount)
	assert.True(t, groups[0].Amount.Equal(amount("305")))
	assert.Equal(t, "Tarp", groups[1].Description)
}

func TestChargeGroupsOfNoLinesIsEmpty(t *testing.T) {
	t.Parallel()

	assert.Empty(t, invoicechargegroup.ChargeGroups(nil))
	assert.Empty(t, invoicechargegroup.ChargeGroups([]*invoice.InvoiceLine{nil}))
}

func TestBilledShipmentIDsNameEachShipmentOnceHeaderFirst(t *testing.T) {
	t.Parallel()

	header, other, third := pulid.MustNew("shp_"), pulid.MustNew("shp_"), pulid.MustNew("shp_")
	invoices := []*invoice.Invoice{
		{
			ShipmentID: header,
			Lines:      []*invoice.InvoiceLine{{ShipmentID: other}, {ShipmentID: header}},
		},
		nil,
		{Lines: []*invoice.InvoiceLine{nil, {ShipmentID: third}, {ShipmentID: other}, {}}},
	}

	assert.Equal(
		t,
		[]pulid.ID{header, other, third},
		invoicechargegroup.BilledShipmentIDs(invoices...),
	)
	assert.Empty(t, invoicechargegroup.BilledShipmentIDs())
}
