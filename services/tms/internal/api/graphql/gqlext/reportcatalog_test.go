package gqlext

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restOnlyProtected are the protected fields of kinds no GraphQL type shows.
// The REST routes withhold them through the entity
// (TestEveryMarginFieldARESTResponseCarriesIsWithheld in responseguard), and
// the report catalog gates them like any other.
var restOnlyProtected = map[permission.Resource]map[string]permission.FieldGroup{
	permission.ResourceRateQuote: {
		"costAmount":    permission.FieldGroupMargins,
		"marginAmount":  permission.FieldGroupMargins,
		"marginPercent": permission.FieldGroupMargins,
	},
	permission.ResourceDispatchControl: {
		"tenderCounterMarginFloorPercent": permission.FieldGroupMargins,
	},
}

func TestEveryRESTOnlyProtectedFieldIsConfidentialInTheRegistry(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	for resource, fields := range restOnlyProtected {
		for field := range fields {
			assert.Equalf(t, permission.SensitivityConfidential,
				registry.GetFieldSensitivity(resource.String(), field),
				"%s.%s is protected, so the registry must classify it confidential",
				resource, field)
		}
	}
}

func schemaProtected(t *testing.T) map[permission.Resource]map[string]permission.FieldGroup {
	t.Helper()

	protected := make(map[permission.Resource]map[string]permission.FieldGroup, 8)
	for resource, fields := range restOnlyProtected {
		protected[resource] = make(map[string]permission.FieldGroup, len(fields))
		for field, group := range fields {
			protected[resource][field] = group
		}
	}
	declared := protectedTypes()
	for object, fields := range loadProtectedFields(t) {
		resource := declared[object].resource
		if resource == "" {
			continue
		}
		if protected[resource] == nil {
			protected[resource] = make(map[string]permission.FieldGroup, len(fields))
		}
		for field, group := range fields {
			protected[resource][field] = group
		}
	}
	return protected
}

func TestReportsProtectWhatTheSchemaProtects(t *testing.T) {
	t.Parallel()

	catalog, err := reportengine.NewCatalog()
	require.NoError(t, err)
	kinds, err := reportaccess.New(reportaccess.Params{Catalog: catalog})
	require.NoError(t, err)

	protected := schemaProtected(t)
	gated := 0
	for _, entity := range catalog.Entities() {
		resource := kinds.Resource(entity)
		for _, field := range entity.Fields {
			key := reportaccess.APIKey(entity.Table.Columns[field.Column].Name)
			want, schemaProtects := protected[resource][key]
			got, reportProtects := kinds.Protection(field)
			if schemaProtects {
				gated++
				assert.Equalf(t, want, got,
					"%s.%s is @protected(group: %s), so reports on %s must gate it in that group",
					resource, key, want, entity.Key)
				continue
			}
			assert.Falsef(t, reportProtects,
				"reports gate %s.%s in %s, but no %s type marks %q @protected; protect it in "+
					"the schema or narrow the protected path",
				entity.Key, key, got, resource, key)
		}
	}
	assert.Positive(t, gated, "no protected field was found in the report catalog")
}
