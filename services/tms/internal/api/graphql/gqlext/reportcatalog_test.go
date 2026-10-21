package gqlext

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryProtectedFieldIsGatedInTheReportCatalog(t *testing.T) {
	t.Parallel()

	gated := 0
	for object, fields := range loadProtectedFields(t) {
		resource, mapped := protectedFieldResources[object]
		require.Truef(t, mapped,
			"%s has protected fields but no resource in protectedFieldResources", object)
		for field, group := range fields {
			for i := range reportcatalog.Default.Entities {
				entity := &reportcatalog.Default.Entities[i]
				if entity.Resource != resource {
					continue
				}
				catalogField, ok := entity.Field(field)
				if !ok {
					continue
				}
				gated++
				assert.Equalf(t, group, catalogField.Protected,
					"%s.%s is @protected(group: %s), so the report catalog's %s.%s must declare "+
						"protected: %s in reportcatalog.yml; then run `task generate-reportcatalog`",
					object, field, group, entity.Key, field, group)
			}
		}
	}
	assert.Positive(t, gated, "no protected field was found in the report catalog")
}

func TestEveryProtectedCatalogFieldIsProtectedInTheSchema(t *testing.T) {
	t.Parallel()

	protected := make(map[permission.Resource]map[string]permission.FieldGroup, 3)
	for object, fields := range loadProtectedFields(t) {
		resource := protectedFieldResources[object]
		if protected[resource] == nil {
			protected[resource] = make(map[string]permission.FieldGroup, len(fields))
		}
		for field, group := range fields {
			protected[resource][field] = group
		}
	}

	for i := range reportcatalog.Default.Entities {
		entity := &reportcatalog.Default.Entities[i]
		for j := range entity.Fields {
			field := &entity.Fields[j]
			if field.Protected == "" {
				continue
			}
			assert.Equalf(t, protected[entity.Resource][field.Key], field.Protected,
				"report catalog %s.%s declares protected: %s, but no %s type marks %q "+
					"@protected(group: %s); protect it in the schema or drop it from the catalog",
				entity.Key, field.Key, field.Protected, entity.Resource, field.Key, field.Protected)
		}
	}
}
