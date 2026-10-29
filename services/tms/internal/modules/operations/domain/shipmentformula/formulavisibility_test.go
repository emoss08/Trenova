package shipmentformula

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ratedShipment() *shipment.Shipment {
	return &shipment.Shipment{
		ID:              pulid.MustNew("shp_"),
		FormulaTemplate: &formulatemplate.FormulaTemplate{Name: "Per pound"},
		RatingDetail: &shipment.RatingDetail{
			FormulaTemplateName: "Per pound",
			Trace:               &shipment.RatingTrace{Statements: rules.RawJSON(`["weight * 2"]`)},
			Bill:                &tarifftypes.Bill{Total: decimal.NewFromInt(42)},
		},
	}
}

func TestWithhold(t *testing.T) {
	t.Parallel()

	original := ratedShipment()
	redacted := Withhold(original)

	require.NotSame(t, original, redacted)
	assert.Nil(t, redacted.FormulaTemplate)
	assert.Empty(t, redacted.RatingDetail.Trace)
	assert.Equal(t, "Per pound", redacted.RatingDetail.FormulaTemplateName)
	assert.True(t, redacted.RatingDetail.Bill.Total.Equal(decimal.NewFromInt(42)),
		"the bill is the price, not the formula")

	assert.NotNil(t, original.FormulaTemplate)
	assert.NotEmpty(t, original.RatingDetail.Trace)
}

func TestWithholdLeavesCleanShipmentAlone(t *testing.T) {
	t.Parallel()

	var missing *shipment.Shipment
	assert.Nil(t, Withhold(missing))

	plain := &shipment.Shipment{
		ID:           pulid.MustNew("shp_"),
		RatingDetail: &shipment.RatingDetail{Bill: &tarifftypes.Bill{Total: decimal.NewFromInt(1)}},
	}
	assert.Same(t, plain, Withhold(plain))
}

func TestWithholdAll(t *testing.T) {
	t.Parallel()

	plain := &shipment.Shipment{ID: pulid.MustNew("shp_")}
	rated := ratedShipment()
	items := []*shipment.Shipment{plain, rated, nil}

	redacted := WithholdAll(items)

	require.Len(t, redacted, 3)
	assert.Same(t, plain, redacted[0])
	assert.NotSame(t, rated, redacted[1])
	assert.Nil(t, redacted[1].FormulaTemplate)
	assert.Empty(t, redacted[1].RatingDetail.Trace)
	assert.Nil(t, redacted[2])
	assert.Same(t, rated, items[1])
	assert.NotEmpty(t, items[1].RatingDetail.Trace)

	clean := []*shipment.Shipment{plain}
	assert.Equal(t, &clean[0], &WithholdAll(clean)[0])
	assert.Empty(t, WithholdAll(nil))
}
