package rules_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const linehaulTariff = `tariff "Per mile"

/// Paid per loaded mile.
input rate: Money/mi = $2.35/mi
input minimumCharge: Money = $0

charge "Linehaul" (quantity: shipment.distance, rate: rate)

minimum minimumCharge

test "the minimum applies to a short load" {
  given shipment.distance = 100 mi
  given input.minimumCharge = $450
  expect total == $450
}
`

func TestEvaluateTariffsRatesABill(t *testing.T) {
	t.Parallel()

	result, err := rules.EvaluateTariffs(t.Context(), &rules.TariffEvaluateRequest{
		Context: &rules.TariffContext{},
		Record:  nil,
		Programs: []rules.TariffProgram{{
			ID:     "rating",
			Source: linehaulTariff,
			Kind:   rules.TariffKindTariff,
			Inputs: map[string]rules.RawJSON{"minimumCharge": rules.MoneyValue(decimal.NewFromInt(450))},
		}},
	})
	require.NoError(t, err)
	rated := result.Result("rating")
	require.NotNil(t, rated)
	require.Equal(t, rules.TariffRated, rated.Status, "%+v", rated.Error)
	require.NotNil(t, rated.Bill)
	assert.Equal(t, "450", rated.Bill.Total.Value.String())
	assert.Equal(t, rules.TariffLineMinimum, rated.Bill.Lines[len(rated.Bill.Lines)-1].Kind)
}

func TestEvaluateTariffsRefusesWithTheAuthorsText(t *testing.T) {
	t.Parallel()

	result, err := rules.EvaluateTariffs(t.Context(), &rules.TariffEvaluateRequest{
		Context: &rules.TariffContext{},
		Programs: []rules.TariffProgram{{
			ID:     "rating",
			Source: "tariff \"t\"\n\nlet w = shipment.weight ?? fail \"No weight\"\ncharge \"Linehaul\" = $1 * w.value\n",
		}},
	})
	require.NoError(t, err)
	refused := result.Result("rating")
	require.Equal(t, rules.TariffRefused, refused.Status)
	assert.Equal(t, "No weight", refused.Refusal.Text)
}

func TestTestTariffRunsTheFilesTests(t *testing.T) {
	t.Parallel()

	run, err := rules.TestTariff(t.Context(), &rules.TariffTestRequest{
		Source:  linehaulTariff,
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)
	require.Len(t, run.Tests, 1)
	assert.True(t, run.Passed(), "%+v", run.Tests)
}

func TestFormatTariffCanonicalises(t *testing.T) {
	t.Parallel()

	formatted, err := rules.FormatTariff(t.Context(), "tariff \"t\"\ncharge \"A\"=$1+$2\n")
	require.NoError(t, err)
	require.NotNil(t, formatted.Formatted)
	assert.Contains(t, *formatted.Formatted, `charge "A" = $1 + $2`)
}
