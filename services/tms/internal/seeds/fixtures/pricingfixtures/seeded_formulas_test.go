package pricingfixtures

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeededLibraries_CheckAsLibrariesAndPassTheirTests(t *testing.T) {
	t.Parallel()

	for _, entry := range seededLibraries() {
		checked, err := checkSeedSource(
			t.Context(),
			rules.TariffKindLibrary,
			formulalibrary.Path(entry.name),
			entry.source,
			nil,
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
		checked, err := checkSeedSource(
			t.Context(),
			rules.TariffKindTariff,
			entry.Name,
			entry.Source,
			nil,
		)
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
