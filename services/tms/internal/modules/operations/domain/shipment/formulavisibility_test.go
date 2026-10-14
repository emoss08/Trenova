package shipment

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ratedShipment() *Shipment {
	return &Shipment{
		ID:              pulid.MustNew("shp_"),
		FormulaTemplate: &formulatemplate.FormulaTemplate{Expression: "weight * 2"},
		RatingDetail: &RatingDetail{
			FormulaTemplateName: "Per pound",
			Expression:          "weight * 2",
			Result:              42,
		},
	}
}

func TestShipment_WithoutFormulaContent(t *testing.T) {
	t.Parallel()

	original := ratedShipment()
	redacted := original.WithoutFormulaContent()

	require.NotSame(t, original, redacted)
	assert.Nil(t, redacted.FormulaTemplate)
	assert.Empty(t, redacted.RatingDetail.Expression)
	assert.Equal(t, "Per pound", redacted.RatingDetail.FormulaTemplateName)
	assert.InDelta(t, 42, redacted.RatingDetail.Result, 0)

	assert.NotNil(t, original.FormulaTemplate)
	assert.Equal(t, "weight * 2", original.RatingDetail.Expression)
}

func TestShipment_WithoutFormulaContentLeavesCleanShipmentAlone(t *testing.T) {
	t.Parallel()

	var missing *Shipment
	assert.Nil(t, missing.WithoutFormulaContent())

	plain := &Shipment{ID: pulid.MustNew("shp_"), RatingDetail: &RatingDetail{Result: 1}}
	assert.Same(t, plain, plain.WithoutFormulaContent())
}

func TestWithoutFormulaContent_Slice(t *testing.T) {
	t.Parallel()

	plain := &Shipment{ID: pulid.MustNew("shp_")}
	rated := ratedShipment()
	items := []*Shipment{plain, rated, nil}

	redacted := WithoutFormulaContent(items)

	require.Len(t, redacted, 3)
	assert.Same(t, plain, redacted[0])
	assert.NotSame(t, rated, redacted[1])
	assert.Nil(t, redacted[1].FormulaTemplate)
	assert.Empty(t, redacted[1].RatingDetail.Expression)
	assert.Nil(t, redacted[2])
	assert.Same(t, rated, items[1])
	assert.Equal(t, "weight * 2", items[1].RatingDetail.Expression)

	clean := []*Shipment{plain}
	assert.Equal(t, &clean[0], &WithoutFormulaContent(clean)[0])
	assert.Empty(t, WithoutFormulaContent(nil))
}
