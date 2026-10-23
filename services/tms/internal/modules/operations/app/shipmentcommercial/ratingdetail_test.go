package shipmentcommercial

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRatingDetailFromQuote_KeepsTheTariffsChoices(t *testing.T) {
	t.Parallel()

	chosen := "Flat"
	flat := decimal.NewFromInt(900)
	choices := []tarifftypes.Choice{{
		Label:    "Linehaul",
		Kind:     tarifftypes.ChoiceBest,
		Currency: "USD",
		Chosen:   &chosen,
		Options: []tarifftypes.ChoiceOption{
			{Label: "Flat", Status: tarifftypes.OptionChosen, Subtotal: &flat},
			{Label: "Volume", Status: tarifftypes.OptionRefused, Refusal: "Too light"},
		},
	}}

	detail := ratingDetailFromQuote(&ratequoteports.RatedShipment{
		Outcome: ratequote.OutcomeRated,
		Choices: choices,
		Quote:   &ratequote.RateQuote{RatedAt: 1_700_000_000},
	})

	require.NotNil(t, detail)
	assert.Equal(t, choices, detail.Choices)
	assert.EqualValues(t, 1_700_000_000, detail.RatedAt)
}
