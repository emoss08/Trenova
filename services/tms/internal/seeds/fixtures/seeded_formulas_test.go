package fixtures

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeededLibraries_CheckAsLibrariesAndPassTheirTests(t *testing.T) {
	t.Parallel()

	for _, entry := range seededLibraries() {
		checked, err := checkSeedSource(
			t.Context(), rules.TariffKindLibrary, formulalibrary.Path(entry.name), entry.source, nil,
		)
		require.NoError(t, err)
		assert.Positive(t, checked.testCount, "%s is tested", entry.name)
		require.NotNil(t, checked.check.Header)
		assert.Equal(t, rules.TariffKindLibrary, checked.check.Header.Kind)
	}
}

func TestSeededTemplates_CheckAndPassTheirTests(t *testing.T) {
	t.Parallel()

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)
	require.NotEmpty(t, catalog)
	for _, entry := range catalog {
		checked, err := checkSeedSource(t.Context(), rules.TariffKindTariff, entry.Name, entry.Source, nil)
		require.NoError(t, err)
		assert.Positive(t, checked.testCount, "%s is tested", entry.Name)
	}
}

func TestSeededLibraries_AreImportableByATariff(t *testing.T) {
	t.Parallel()

	library := seededLibraries()[0]
	source := `tariff "Hazmat lane"

use org.accessorials.{ hazmat, residential }

input rate: Money/mi

charge "Linehaul" (quantity: shipment.distance, rate: rate)
if shipment.hazmat {
  charge "Hazmat" = hazmat(shipment.distance)
}
charge "Residential" = residential(shipment.weight ?? 0 lb)

test "a hazmat load bills the library's fee" {
  given input.rate = $2/mi
  given shipment.distance = 100 mi
  given shipment.hazmat = true
  given shipment.weight = 12_000 lb
  expect charge("Hazmat") == $185
  expect charge("Residential") == none
}
`
	_, err := checkSeedSource(t.Context(), rules.TariffKindTariff, "Hazmat lane", source,
		&rules.TariffContext{Libraries: []rules.TariffLibrary{{
			Path:    formulalibrary.Path(library.name),
			Version: "1",
			Source:  library.source,
		}}})
	require.NoError(t, err)
}

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
		inputs := seededTariffInputs(lane.def.template, decimal.RequireFromString(lane.def.rate))
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
		formulatemplate.StandardPerMile:  seededRate(formulatemplate.StandardPerMile, decimal.Zero),
		formulatemplate.StandardPerStop:  seededRate(formulatemplate.StandardPerStop, decimal.Zero),
		formulatemplate.StandardFlatRate: decimal.NewFromInt(6200),
	} {
		cases = append(cases, seeded{template: name, values: seededTariffInputs(name, rate)})
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
