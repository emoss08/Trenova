package invoicelines_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/invoicelines"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
		BaseRate:            decimal.NewNullDecimal(dec("3.5")),
		AdditionalCharges:   charges,
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

	lines := invoicelines.ForShipment(billingqueue.BillTypeInvoice, shp, 4)
	require.Len(t, lines, 4)

	freight := lines[0]
	assert.Equal(t, 4, freight.LineNumber)
	assert.Equal(t, invoice.InvoiceLineTypeFreight, freight.Type)
	assert.Equal(t, invoicelines.FreightDescription, freight.Description)
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
	assert.Equal(t, accessorialcharge.RateUnitHour, perUnit.RateUnit)
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

	lines := invoicelines.ForShipment(billingqueue.BillTypeInvoice, shp, 1)
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

	lines := invoicelines.ForShipment(billingqueue.BillTypeInvoice, shp, 1)

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

	lines := invoicelines.ForShipment(billingqueue.BillTypeCreditMemo, shp, 1)

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
	shp.BaseRate = decimal.NewNullDecimal(decimal.Zero)

	lines := invoicelines.ForShipment(billingqueue.BillTypeInvoice, shp, 1)

	assert.Equal(t, "Flat Rate", lines[0].FormulaTemplateName)
	assert.False(t, lines[0].Rate.Valid)
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
	alreadyLoaded := &shipment.AdditionalCharge{AccessorialChargeID: loaded.ID, AccessorialCharge: loaded}

	repo := mocks.NewMockAccessorialChargeRepository(t)
	repo.On("GetByID", mock.Anything, mock.MatchedBy(func(req repositories.GetAccessorialChargeByIDRequest) bool {
		return req.ID == det.ID && *req.TenantInfo == tenantInfo
	})).Return(det, nil).Once()
	repo.On("GetByID", mock.Anything, mock.MatchedBy(func(req repositories.GetAccessorialChargeByIDRequest) bool {
		return req.ID == current.ID
	})).Return(current, nil).Once()

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

	repo := mocks.NewMockAccessorialChargeRepository(t)
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

	require.NoError(t, invoicelines.HydrateAccessorials(t.Context(), nil, pagination.TenantInfo{}, testShipment(charge)))
	assert.Nil(t, charge.AccessorialCharge)
}
