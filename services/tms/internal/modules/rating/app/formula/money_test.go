package formula_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_MoneyArithmeticCarriesNoBinaryNoise(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	cases := map[string]string{
		`round(0.145 * 100)`:           "15",
		`round(1.005, 2)`:              "1.01",
		`roundUp(0.145 * 100)`:         "15",
		`roundTo(0.145 * 100, 1)`:      "15",
		`0.1 + 0.2`:                    "0.3",
		`sum([0.1, 0.2, 0.3])`:         "0.6",
		`sum(0.1, 0.2, 0.3)`:           "0.6",
		`avg([0.1, 0.2, 0.3])`:         "0.2",
		`1.15 * 100`:                   "115",
		`sum(map([0.7, 0.1], # * 10))`: "8",
	}

	for expression, want := range cases {
		t.Run(expression, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, want, evaluate(t, svc, shippedOnMarchFirst(), expression).String())
		})
	}
}

func TestService_Rate_BillsTheSumWithoutBinaryNoise(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: activeTemplate(`0.1 + 0.2`),
		Entity:   shippedOnMarchFirst(),
	})
	require.NoError(t, err)
	assert.Equal(t, "0.3", resp.Amount.String())
	assert.Equal(t, "0.3", resp.Rounding.UnroundedAmount.String())
	assert.False(t, resp.Rounding.Applied)
}

func TestService_Rate_BreakdownLinesAreRoundedAndReconcileToTheBill(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	template := activeTemplate(`10.004 + 20.004 + 30.004`)
	template.RoundingMode = ratetypes.RoundingModeHalfUp
	template.RoundingPrecision = 2
	template.BreakdownDefinitions = []*formulatypes.BreakdownDefinition{
		{Name: "linehaul", Label: "Linehaul", Expression: `10.004`},
		{Name: "fuel", Label: "Fuel", Expression: `20.004`},
		{Name: "accessorials", Label: "Accessorials", Expression: `30.004`},
	}

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: template,
		Entity:   shippedOnMarchFirst(),
	})
	require.NoError(t, err)
	assert.Equal(t, "60.01", resp.Amount.String())

	require.Len(t, resp.Breakdown, 4)
	assert.Equal(t, []formulatemplatetypes.BreakdownAmount{
		{Name: "linehaul", Label: "Linehaul", Amount: decimal.RequireFromString("10")},
		{Name: "fuel", Label: "Fuel", Amount: decimal.RequireFromString("20")},
		{Name: "accessorials", Label: "Accessorials", Amount: decimal.RequireFromString("30")},
		{
			Name:   "roundingRemainder",
			Label:  "Rounding",
			Amount: decimal.RequireFromString("0.01"),
		},
	}, normalizedLines(resp.Breakdown))

	total := decimal.Zero
	for _, line := range resp.Breakdown {
		total = total.Add(line.Amount)
	}
	assert.True(t, total.Equal(resp.Amount), "%s != %s", total, resp.Amount)
}

func TestService_Rate_LinesThatDoNotReconcileGetNoRemainder(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	template := activeTemplate(`100`)
	template.BreakdownDefinitions = []*formulatypes.BreakdownDefinition{
		{Name: "linehaul", Label: "Linehaul", Expression: `60.004`},
	}

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: template,
		Entity:   shippedOnMarchFirst(),
	})
	require.NoError(t, err)
	require.Len(t, resp.Breakdown, 1, "a missing line is not rounding drift")
	assert.Equal(t, "60", resp.Breakdown[0].Amount.String())
}

func normalizedLines(
	lines []formulatemplatetypes.BreakdownAmount,
) []formulatemplatetypes.BreakdownAmount {
	normalized := make([]formulatemplatetypes.BreakdownAmount, len(lines))
	for i, line := range lines {
		line.Amount = decimal.RequireFromString(line.Amount.String())
		normalized[i] = line
	}
	return normalized
}
