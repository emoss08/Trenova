package ratefixtures

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/seeds/fixtures/pricingfixtures"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeededContractValues_FitTheStandardTemplatesConstraints(t *testing.T) {
	t.Parallel()

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)
	sources := make(map[string]string, len(catalog))
	for _, entry := range catalog {
		sources[entry.Name] = entry.Source
	}

	type seeded struct {
		template string
		values   map[string]rules.RawJSON
	}
	cases := make([]seeded, 0, 16)
	for _, lane := range seededLanes() {
		inputs := pricingfixtures.SeededTariffInputs(
			lane.def.template,
			decimal.RequireFromString(lane.def.rate),
		)
		inputs[formulatemplate.StandardInputMinimumCharge] = rules.MoneyValue(
			decimal.RequireFromString(lane.def.minCharge),
		)
		cases = append(cases, seeded{template: lane.def.template, values: inputs})
	}
	for _, rate := range carrierBuyRates {
		cases = append(cases, seeded{
			template: formulatemplate.StandardPerMile,
			values: map[string]rules.RawJSON{
				formulatemplate.StandardInputRate: rules.QuantityValue(
					decimal.RequireFromString(rate),
					rules.UnitMoney+"/mi",
				),
				formulatemplate.StandardInputMinimumCharge: rules.MoneyValue(
					decimal.RequireFromString(carrierMinCharge),
				),
			},
		})
	}
	for name, rate := range map[string]decimal.Decimal{
		formulatemplate.StandardPerMile:  pricingfixtures.SeededRate(formulatemplate.StandardPerMile, decimal.Zero),
		formulatemplate.StandardPerStop:  pricingfixtures.SeededRate(formulatemplate.StandardPerStop, decimal.Zero),
		formulatemplate.StandardFlatRate: decimal.NewFromInt(6200),
	} {
		cases = append(
			cases,
			seeded{template: name, values: pricingfixtures.SeededTariffInputs(name, rate)},
		)
	}

	for _, entry := range cases {
		check, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
			Source:  sources[entry.template],
			Kind:    rules.TariffKindTariff,
			Context: &rules.TariffContext{},
			Values:  &rules.TariffInputValues{Rule: entry.values},
		})
		require.NoError(t, err)
		assert.True(t, check.OK, "%s: %+v", entry.template, check.Diagnostics)
	}
}
