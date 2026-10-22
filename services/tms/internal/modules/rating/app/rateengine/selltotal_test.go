package rateengine

import (
	"testing"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSellSideRecordsHaveNoSellTotal(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d, perMileRule("2.15", 100))

	_, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))
	require.NoError(t, err)

	record := string(d.evaluator.Requests()[0].Record)
	assert.Contains(t, record, `"mode":"Sell"`)
	assert.Contains(t, record, `"sellTotal":null`)
}

func TestFallbackTemplateRecordHasNoSellTotal(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d)

	entity := testShipment()
	entity.FormulaTemplateID = flatTemplateID
	entity.TariffInputs = tarifftypes.Inputs{"rate": rules.MoneyValue(decimal.NewFromInt(900))}

	_, err := d.svc.RateShipment(t.Context(), rateRequest(entity))
	require.NoError(t, err)

	require.Len(t, d.evaluator.Requests(), 1)
	assert.Contains(t, string(d.evaluator.Requests()[0].Record), `"sellTotal":null`)
}
