package standardcatalog_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice/standardcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadParsesEveryTemplate(t *testing.T) {
	t.Parallel()

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)
	require.NotEmpty(t, catalog)

	seen := make(map[string]struct{}, len(catalog))
	for _, entry := range catalog {
		assert.NotEmpty(t, entry.Name)
		assert.NotEmpty(t, entry.Description, "description for %s", entry.Name)
		assert.NotEmpty(t, entry.Expression, "expression for %s", entry.Name)
		assert.Equal(t, "shipment", entry.SchemaID, "schema for %s", entry.Name)

		_, duplicate := seen[entry.Name]
		assert.False(t, duplicate, "duplicate template %s", entry.Name)
		seen[entry.Name] = struct{}{}
	}
}

func TestCatalogCoversStandardNames(t *testing.T) {
	t.Parallel()

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)

	names := make(map[string]struct{}, len(catalog))
	for _, entry := range catalog {
		names[entry.Name] = struct{}{}
	}

	// The rate sheet importer and the formula-only migration backfill match on
	// these exact names, so the catalog must always contain them.
	standards := []string{
		formulatemplate.StandardFlatRate,
		formulatemplate.StandardPerMile,
		formulatemplate.StandardPerStop,
		formulatemplate.StandardPerPound,
		formulatemplate.StandardPerPallet,
		formulatemplate.StandardPerPiece,
		formulatemplate.StandardPerLinearFoot,
		formulatemplate.StandardPerCwt,
		formulatemplate.StandardPerHour,
		formulatemplate.StandardPercentOfSellRate,
	}

	for _, name := range standards {
		_, ok := names[name]
		assert.True(t, ok, "catalog is missing standard template %q", name)
	}
}

func TestPerHourHasNoManualDefaultClobberingComputedHours(t *testing.T) {
	t.Parallel()

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)

	for _, entry := range catalog {
		if entry.Name != formulatemplate.StandardPerHour {
			continue
		}

		for _, variable := range entry.VariableDefinitions {
			if variable.Name == "totalHours" {
				assert.Nil(t, variable.DefaultValue,
					"a totalHours default would shadow the computed stop-time hours")
			}
		}
		return
	}

	t.Fatal("Per Hour template not found in catalog")
}
