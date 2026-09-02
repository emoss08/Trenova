package formula_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatrixLookup_ExplainsWhatMatched(t *testing.T) {
	t.Parallel()

	lookup := formula.NewMatrixLookup([]*repositories.RateMatrixLookupData{
		exactLookupMatrix("fsc", map[string]string{"DIESEL": "0.35"}),
		rangeLookupMatrix("miles", []bandDef{
			{min: "0", max: "500", value: "2.5"},
			{min: "500", max: "", value: "2.0"},
		}),
	})

	explainer, ok := lookup.(formulatemplatetypes.LookupExplainer)
	require.True(t, ok, "the matrix lookup can explain its matches")

	exact, ok := explainer.ExplainLookup("fsc", "DIESEL")
	require.True(t, ok)
	assert.Equal(t, "DIESEL", exact.MatchedKey)
	assert.Nil(t, exact.BandMin)

	band, ok := explainer.ExplainLookup("miles", 750)
	require.True(t, ok)
	require.NotNil(t, band.BandMin)
	assert.True(t, band.BandMin.Equal(decimal.NewFromInt(500)))
	assert.Nil(t, band.BandMax, "the top band is open-ended")

	_, ok = explainer.ExplainLookup("miles", 9999999)
	assert.True(t, ok, "an open-ended top band matches anything above its floor")

	_, ok = explainer.ExplainLookup("nope", 1)
	assert.False(t, ok)
}
