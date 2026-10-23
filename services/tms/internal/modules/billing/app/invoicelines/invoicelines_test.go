package invoicelines_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicelines"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func forShipment(
	t *testing.T,
	billType billingqueue.BillType,
	shp *shipment.Shipment,
	start int,
) []*invoice.InvoiceLine {
	t.Helper()

	lines, err := invoicelines.ForShipment(t.Context(), billType, shp, start)
	require.NoError(t, err)

	return lines
}

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func detention() *accessorialcharge.AccessorialCharge {
	return &accessorialcharge.AccessorialCharge{
		ID:          pulid.MustNew("acc_"),
		Code:        "DET",
		Description: "Detention",
		Method:      accessorialcharge.MethodPerUnit,
		RateUnit:    accessorialcharge.RateUnitHour,
	}
}

func testShipment(charges ...*shipment.AdditionalCharge) *shipment.Shipment {
	return &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		ProNumber:           "SEED-DET-011",
		BOL:                 "BOL-2026-0211",
		FreightChargeAmount: decimal.NewNullDecimal(dec("2450")),
		FreightLines: []shipment.FreightLine{{
			Label:        "Per Mile",
			Quantity:     new(dec("700")),
			QuantityUnit: "mi",
			Rate:         new(dec("3.5")),
			RateUnit:     "$/mi",
			Amount:       dec("2450"),
		}},
		AdditionalCharges: charges,
	}
}

func TestForShipmentNamesEachAccessorialAndHowItWasCalculated(t *testing.T) {
	t.Parallel()

	det := detention()
	fuel := &accessorialcharge.AccessorialCharge{
		ID:          pulid.MustNew("acc_"),
		Code:        "FSC",
		Description: "Fuel Surcharge",
		Method:      accessorialcharge.MethodPercentage,
	}
	lumper := &accessorialcharge.AccessorialCharge{
		ID:          pulid.MustNew("acc_"),
		Code:        "LMP",
		Description: "Lumper Fee",
		Method:      accessorialcharge.MethodFlat,
	}
	shp := testShipment(
		&shipment.AdditionalCharge{
			AccessorialChargeID: det.ID,
			AccessorialCharge:   det,
			Method:              accessorialcharge.MethodPerUnit,
			Amount:              dec("75"),
			Unit:                2,
		},
		&shipment.AdditionalCharge{
			AccessorialChargeID: fuel.ID,
			AccessorialCharge:   fuel,
			Method:              accessorialcharge.MethodPercentage,
			Amount:              dec("10"),
			Unit:                1,
		},
		&shipment.AdditionalCharge{
			AccessorialChargeID: lumper.ID,
			AccessorialCharge:   lumper,
			Method:              accessorialcharge.MethodFlat,
			Amount:              dec("150"),
			Unit:                1,
		},
	)
	shp.RatingDetail = &shipment.RatingDetail{FormulaTemplateName: "Per Mile"}

	lines := forShipment(t, billingqueue.BillTypeInvoice, shp, 4)
	require.Len(t, lines, 4)

	freight := lines[0]
	assert.Equal(t, 4, freight.LineNumber)
	assert.Equal(t, invoice.InvoiceLineTypeFreight, freight.Type)
	assert.Equal(t, "Per Mile", freight.Description)
	assert.True(t, freight.Quantity.Equal(dec("700")))
	assert.True(t, freight.UnitPrice.Equal(dec("3.5")))
	assert.Equal(t, "$/mi", freight.RateUnit)
	assert.True(t, freight.Amount.Equal(dec("2450")))
	assert.Equal(t, "Per Mile", freight.FormulaTemplateName)
	require.True(t, freight.Rate.Valid)
	assert.True(t, freight.Rate.Decimal.Equal(dec("3.5")))
	assert.Equal(t, shp.ID, freight.ShipmentID)

	perUnit := lines[1]
	assert.Equal(t, 5, perUnit.LineNumber)
	assert.Equal(t, "Detention", perUnit.Description)
	assert.Equal(t, "DET", perUnit.ChargeCode)
	assert.Equal(t, det.ID, perUnit.AccessorialChargeID)
	assert.Equal(t, accessorialcharge.MethodPerUnit, perUnit.ChargeMethod)
	assert.Equal(t, "$/h", perUnit.RateUnit)
	assert.True(t, perUnit.Rate.Decimal.Equal(dec("75")))
	assert.True(t, perUnit.Quantity.Equal(dec("2")))
	assert.True(t, perUnit.Amount.Equal(dec("150")))
	assert.False(t, perUnit.RateBasisAmount.Valid)

	percentage := lines[2]
	assert.Equal(t, "Fuel Surcharge", percentage.Description)
	assert.Equal(t, accessorialcharge.MethodPercentage, percentage.ChargeMethod)
	assert.True(t, percentage.Rate.Decimal.Equal(dec("10")))
	require.True(t, percentage.RateBasisAmount.Valid)
	assert.True(t, percentage.RateBasisAmount.Decimal.Equal(dec("2450")))
	assert.True(t, percentage.Amount.Equal(dec("245")))
	assert.Empty(t, percentage.RateUnit)

	flat := lines[3]
	assert.Equal(t, "Lumper Fee", flat.Description)
	assert.Equal(t, "LMP", flat.ChargeCode)
	assert.Empty(t, flat.RateUnit)
	assert.True(t, flat.Amount.Equal(dec("150")))
}

func TestForShipmentFallsBackWhenTheAccessorialIsMissingOrStale(t *testing.T) {
	t.Parallel()

	stale := detention()
	shp := testShipment(
		&shipment.AdditionalCharge{
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodFlat,
			Amount:              dec("50"),
			Unit:                1,
		},
		&shipment.AdditionalCharge{
			AccessorialChargeID: pulid.MustNew("acc_"),
			AccessorialCharge:   stale,
			Method:              accessorialcharge.MethodFlat,
			Amount:              dec("25"),
			Unit:                1,
		},
	)

	lines := forShipment(t, billingqueue.BillTypeInvoice, shp, 1)
	require.Len(t, lines, 3)

	for _, line := range lines[1:] {
		assert.Equal(t, invoicelines.AccessorialDescription, line.Description)
		assert.Empty(t, line.ChargeCode)
		assert.False(t, line.AccessorialChargeID.IsNil())
	}
}

func TestForShipmentUsesTheCodeWhenAnAccessorialHasNoDescription(t *testing.T) {
	t.Parallel()

	unnamed := &accessorialcharge.AccessorialCharge{ID: pulid.MustNew("acc_"), Code: "TONU"}
	shp := testShipment(&shipment.AdditionalCharge{
		AccessorialChargeID: unnamed.ID,
		AccessorialCharge:   unnamed,
		Method:              accessorialcharge.MethodFlat,
		Amount:              dec("250"),
		Unit:                1,
	})

	lines := forShipment(t, billingqueue.BillTypeInvoice, shp, 1)

	assert.Equal(t, "TONU", lines[1].Description)
}

func TestForShipmentNegatesCreditMemoAmountsButNotRates(t *testing.T) {
	t.Parallel()

	det := detention()
	shp := testShipment(&shipment.AdditionalCharge{
		AccessorialChargeID: det.ID,
		AccessorialCharge:   det,
		Method:              accessorialcharge.MethodPerUnit,
		Amount:              dec("75"),
		Unit:                2,
	})

	lines := forShipment(t, billingqueue.BillTypeCreditMemo, shp, 1)

	assert.True(t, lines[0].Amount.Equal(dec("-2450")))
	assert.True(t, lines[1].Amount.Equal(dec("-150")))
	assert.True(t, lines[1].UnitPrice.Equal(dec("-75")))
	assert.True(t, lines[1].Rate.Decimal.Equal(dec("75")))
}

func TestForShipmentNamesTheFormulaFromTheLoadedTemplate(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	shp := testShipment()
	shp.FormulaTemplateID = templateID
	shp.FormulaTemplate = &formulatemplate.FormulaTemplate{ID: templateID, Name: "Flat Rate"}
	shp.FreightLines = nil

	lines := forShipment(t, billingqueue.BillTypeInvoice, shp, 1)

	assert.Equal(t, "Flat Rate", lines[0].FormulaTemplateName)
	assert.Equal(t, templateID, lines[0].FormulaTemplateID)
	assert.False(t, lines[0].Rate.Valid)
}

func TestForShipmentKeepsTheTemplateTheRatingUsed(t *testing.T) {
	t.Parallel()

	ratedWith := pulid.MustNew("ft_")
	assignedSince := pulid.MustNew("ft_")
	shp := testShipment()
	shp.RatingDetail = &shipment.RatingDetail{
		FormulaTemplateID:   ratedWith.String(),
		FormulaTemplateName: "Per Mile",
	}
	shp.FormulaTemplateID = assignedSince
	shp.FormulaTemplate = &formulatemplate.FormulaTemplate{ID: assignedSince, Name: "Per CWT"}

	freight := forShipment(t, billingqueue.BillTypeInvoice, shp, 1)[0]

	assert.Equal(t, ratedWith, freight.FormulaTemplateID)
	assert.Equal(t, "Per Mile", freight.FormulaTemplateName)
}

func TestForShipmentNamesTheTemplateOfALegacyRatingByItsName(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	tests := []struct {
		name     string
		assigned string
		wantID   pulid.ID
	}{
		{
			name:     "the assigned template has the rating's name",
			assigned: "Per Mile",
			wantID:   templateID,
		},
		{name: "the assigned template was renamed or replaced", assigned: "Per CWT", wantID: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			shp := testShipment()
			shp.RatingDetail = &shipment.RatingDetail{FormulaTemplateName: "Per Mile"}
			shp.FormulaTemplateID = templateID
			shp.FormulaTemplate = &formulatemplate.FormulaTemplate{
				ID:   templateID,
				Name: tt.assigned,
			}

			freight := forShipment(t, billingqueue.BillTypeInvoice, shp, 1)[0]

			assert.Equal(t, tt.wantID, freight.FormulaTemplateID)
			assert.Equal(t, "Per Mile", freight.FormulaTemplateName)
		})
	}
}

func TestForShipmentLeavesOutATemplateIDThatIsNotOne(t *testing.T) {
	t.Parallel()

	shp := testShipment()
	shp.RatingDetail = &shipment.RatingDetail{
		FormulaTemplateID:   "per-mile",
		FormulaTemplateName: "Per Mile",
	}

	freight := forShipment(t, billingqueue.BillTypeInvoice, shp, 1)[0]

	assert.True(t, freight.FormulaTemplateID.IsNil())
	assert.Equal(t, "Per Mile", freight.FormulaTemplateName)
}

func TestForShipmentNamesNoTemplateForAccessorialLines(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	fuel := &accessorialcharge.AccessorialCharge{
		ID:     pulid.MustNew("acc_"),
		Code:   "FSC",
		Method: accessorialcharge.MethodPercentage,
	}
	shp := testShipment(&shipment.AdditionalCharge{
		AccessorialChargeID: fuel.ID,
		AccessorialCharge:   fuel,
		Method:              accessorialcharge.MethodPercentage,
		Amount:              dec("10"),
		Unit:                1,
	})
	shp.RatingDetail = &shipment.RatingDetail{
		FormulaTemplateID:   templateID.String(),
		FormulaTemplateName: "Per Mile",
	}

	lines := forShipment(t, billingqueue.BillTypeInvoice, shp, 1)
	require.Len(t, lines, 2)

	assert.Equal(t, templateID, lines[0].FormulaTemplateID)
	assert.True(t, lines[1].FormulaTemplateID.IsNil())
	assert.Empty(t, lines[1].FormulaTemplateName)
}

func TestHydrateAccessorialsLoadsMissingAndStaleDefinitionsOnce(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	det := detention()
	current := &accessorialcharge.AccessorialCharge{ID: pulid.MustNew("acc_"), Code: "LMP"}
	loaded := &accessorialcharge.AccessorialCharge{ID: pulid.MustNew("acc_"), Code: "FSC"}

	missingA := &shipment.AdditionalCharge{AccessorialChargeID: det.ID}
	missingB := &shipment.AdditionalCharge{AccessorialChargeID: det.ID}
	stale := &shipment.AdditionalCharge{AccessorialChargeID: current.ID, AccessorialCharge: det}
	alreadyLoaded := &shipment.AdditionalCharge{
		AccessorialChargeID: loaded.ID,
		AccessorialCharge:   loaded,
	}

	repo := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	repo.On("GetByID", mock.Anything, mock.MatchedBy(func(req accessorialchargeports.GetAccessorialChargeByIDRequest) bool {
		return req.ID == det.ID && *req.TenantInfo == tenantInfo
	})).
		Return(det, nil).
		Once()
	repo.On("GetByID", mock.Anything, mock.MatchedBy(func(req accessorialchargeports.GetAccessorialChargeByIDRequest) bool {
		return req.ID == current.ID
	})).
		Return(current, nil).
		Once()

	err := invoicelines.HydrateAccessorials(
		t.Context(),
		repo,
		tenantInfo,
		testShipment(missingA, stale),
		nil,
		testShipment(missingB, alreadyLoaded),
	)
	require.NoError(t, err)

	assert.Same(t, det, missingA.AccessorialCharge)
	assert.Same(t, det, missingB.AccessorialCharge)
	assert.Same(t, current, stale.AccessorialCharge)
	assert.Same(t, loaded, alreadyLoaded.AccessorialCharge)
}

func TestHydrateAccessorialsReturnsLookupErrors(t *testing.T) {
	t.Parallel()

	repo := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("database unavailable"))

	err := invoicelines.HydrateAccessorials(
		t.Context(),
		repo,
		pagination.TenantInfo{},
		testShipment(&shipment.AdditionalCharge{AccessorialChargeID: pulid.MustNew("acc_")}),
	)

	require.Error(t, err)
}

func TestHydrateAccessorialsWithoutARepositoryLeavesChargesAlone(t *testing.T) {
	t.Parallel()

	charge := &shipment.AdditionalCharge{AccessorialChargeID: pulid.MustNew("acc_")}

	require.NoError(
		t,
		invoicelines.HydrateAccessorials(
			t.Context(),
			nil,
			pagination.TenantInfo{},
			testShipment(charge),
		),
	)
	assert.Nil(t, charge.AccessorialCharge)
}

func TestForShipmentBillsOneLinePerFreightLine(t *testing.T) {
	t.Parallel()

	shp := testShipment()
	shp.FreightChargeAmount = decimal.NewNullDecimal(dec("2600"))
	shp.FreightLines = []shipment.FreightLine{
		{
			Label:        "Linehaul",
			Quantity:     new(dec("700")),
			QuantityUnit: "mi",
			Rate:         new(dec("3.5")),
			RateUnit:     "$/mi",
			Amount:       dec("2450"),
		},
		{Label: "Minimum charge adjustment", Amount: dec("150")},
	}

	lines := forShipment(t, billingqueue.BillTypeInvoice, shp, 1)

	require.Len(t, lines, 2)
	assert.Equal(t, "Linehaul", lines[0].Description)
	assert.Equal(t, 1, lines[0].LineNumber)
	assert.True(t, lines[0].Amount.Equal(dec("2450")))
	assert.Equal(t, "Minimum charge adjustment", lines[1].Description)
	assert.Equal(t, 2, lines[1].LineNumber)
	assert.True(t, lines[1].Quantity.Equal(dec("1")))
	assert.True(t, lines[1].Amount.Equal(dec("150")))
	for _, line := range lines {
		assert.Equal(t, invoice.InvoiceLineTypeFreight, line.Type)
	}
}

func TestForShipmentSplitsAPartialFreightShareAcrossTheLinesToTheCent(t *testing.T) {
	t.Parallel()

	shp := testShipment()
	shp.FreightChargeAmount = decimal.NewNullDecimal(dec("1000"))
	shp.FreightLines = []shipment.FreightLine{
		{Label: "Linehaul", Amount: dec("666.67")},
		{Label: "Stop charges", Amount: dec("333.33")},
	}
	share := &shipment.PayerShare{Charges: []shipment.AllocatedCharge{{
		Kind:        shipment.ChargeAllocationKindFreight,
		ChargeTotal: dec("1000"),
		Amount:      dec("333.33"),
		Percent:     decimal.NewNullDecimal(dec("33.333")),
		Partial:     true,
	}}}

	lines := invoicelines.ForShipmentShare(billingqueue.BillTypeInvoice, shp, share, 1)

	require.Len(t, lines, 2)
	total := lines[0].Amount.Add(lines[1].Amount)
	assert.True(t, total.Equal(dec("333.33")), "the lines add up to the share, got %s", total)
	assert.Contains(t, lines[0].Description, "Linehaul")
	assert.Contains(t, lines[0].Description, "share")
	assert.True(t, lines[0].Quantity.Equal(dec("1")), "a partial share is billed as one amount")
}

func TestForShipmentBillsAShipmentWithoutFreightLinesAsOneLine(t *testing.T) {
	t.Parallel()

	shp := testShipment()
	shp.FreightLines = nil

	lines := forShipment(t, billingqueue.BillTypeInvoice, shp, 1)

	require.Len(t, lines, 1)
	assert.Equal(t, invoicelines.FreightDescription, lines[0].Description)
	assert.True(t, lines[0].Amount.Equal(dec("2450")))
}

func TestForShipmentItemizesATariffAccessorial(t *testing.T) {
	t.Parallel()

	stop := &accessorialcharge.AccessorialCharge{
		ID:          pulid.MustNew("acc_"),
		Code:        "XDROP",
		Description: "Extra drop",
	}
	shp := testShipment(&shipment.AdditionalCharge{
		AccessorialChargeID: stop.ID,
		AccessorialCharge:   stop,
		IsSystemGenerated:   true,
		IsTariff:            true,
		Method:              accessorialcharge.MethodFlat,
		Amount:              dec("150"),
		Unit:                1,
		Label:               "Extra drops",
		Quantity:            decimal.NewNullDecimal(dec("2")),
		Rate:                decimal.NewNullDecimal(dec("75")),
		RateUnit:            "$/stop",
	})

	lines := forShipment(t, billingqueue.BillTypeInvoice, shp, 1)

	require.Len(t, lines, 2)
	accessorial := lines[1]
	assert.Equal(t, "Extra drops", accessorial.Description)
	assert.Equal(t, "XDROP", accessorial.ChargeCode)
	assert.True(t, accessorial.Quantity.Equal(dec("2")))
	assert.True(t, accessorial.UnitPrice.Equal(dec("75")))
	assert.Equal(t, "$/stop", accessorial.RateUnit)
}

func TestRatingNotesCollectEachBillsNotesOnce(t *testing.T) {
	t.Parallel()

	first := testShipment()
	first.RatingDetail = &shipment.RatingDetail{Bill: &tarifftypes.Bill{Notes: []tarifftypes.Note{
		{Text: "Rated on the 2026 contract"}, {Text: " "},
	}}}
	second := testShipment()
	second.RatingDetail = &shipment.RatingDetail{Bill: &tarifftypes.Bill{Notes: []tarifftypes.Note{
		{Text: "Rated on the 2026 contract"}, {Text: "Fuel at the peg"},
	}}}

	assert.Equal(t,
		[]string{"Rated on the 2026 contract", "Fuel at the peg"},
		invoicelines.RatingNotes(first, nil, second, testShipment()),
	)
}
