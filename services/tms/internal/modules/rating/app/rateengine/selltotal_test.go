package rateengine

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatrixCellBindsTheSellTotalOnTheBuySide(t *testing.T) {
	t.Parallel()

	d := setup(t)
	matrix, cell := distanceMatrix("2.15")
	rule := matrixRule(matrix)
	rule.PartyType = rateagreement.PartyTypeCarrier
	expectRules(d, rule)
	expectMatrix(d, matrix, cell)

	req := rateRequest(testShipment())
	req.PartyType = rateagreement.PartyTypeCarrier
	req.PartyID = pulid.MustNew("car_")
	req.SellTotal = decimal.NewNullDecimal(decimal.NewFromInt(2_000))

	_, err := d.svc.RateShipment(t.Context(), req)
	require.NoError(t, err)

	require.Len(t, d.calcRequests, 1)
	assertBound(t, "2000", d.calcRequests[0].Overrides["sellTotal"])
	assert.Empty(t, d.calcRequests[0].UnavailableInputs)
}

func TestSellSideFormulasAreToldTheSellTotalIsUnknown(t *testing.T) {
	t.Parallel()

	d := setup(t)
	matrix, cell := distanceMatrix("2.15")
	expectRules(d, matrixRule(matrix))
	expectMatrix(d, matrix, cell)

	_, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))
	require.NoError(t, err)

	require.Len(t, d.calcRequests, 1)
	assert.NotContains(t, d.calcRequests[0].Overrides, "sellTotal")
	assert.Contains(t, d.calcRequests[0].UnavailableInputs, "sellTotal")
}

func TestFallbackTemplateIsToldTheSellTotalIsUnknown(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d)
	expectTemplates(d)

	entity := testShipment()
	entity.FormulaTemplateID = flatTemplateID

	_, err := d.svc.RateShipment(t.Context(), rateRequest(entity))
	require.NoError(t, err)

	require.Len(t, d.calcRequests, 1)
	assert.Contains(t, d.calcRequests[0].UnavailableInputs, "sellTotal")
}
