package canned_test

import (
	"testing"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryCannedReportCompilesAgainstTheCatalog(t *testing.T) {
	t.Parallel()

	catalog, err := reportengine.NewCatalog()
	require.NoError(t, err)
	registry, err := canned.Default()
	require.NoError(t, err)
	require.NotEmpty(t, registry.All())

	cfg := &config.ReportingConfig{}
	limits := reportengine.DefinitionLimits(cfg)
	complexity := reportengine.ComplexityLimits(cfg)

	for _, entry := range registry.All() {
		t.Run(entry.Key, func(t *testing.T) {
			t.Parallel()

			ctx := dbscope.WithTenant(t.Context(), dbscope.Tenant{
				OrganizationID: pulid.ID("org_test"),
				BusinessUnitID: pulid.ID("bu_test"),
			})
			compiled, compileErr := analytics.Compile(
				ctx,
				catalog,
				nil,
				entry.Definition,
				nil,
				analytics.CompileOptions{Limits: limits},
			)
			require.NoError(t, compileErr,
				"canned report %s does not compile with its default parameters", entry.Key)
			assert.LessOrEqual(t, compiled.Complexity.Joins, complexity.MaxJoins,
				"canned report %s joins more relations than a run allows", entry.Key)
			assert.LessOrEqual(t, compiled.Complexity.ToMany, complexity.MaxToMany,
				"canned report %s reads more to-many relations than a run allows", entry.Key)
		})
	}
}
