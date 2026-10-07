package reportresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogEntityReadsInTheRequestersLanguage(t *testing.T) {
	entity := &reportcatalog.Entity{
		Key:         "customer",
		Resource:    permission.ResourceCustomer,
		Label:       "Customer",
		PluralLabel: "Customers",
		Fields: []reportcatalog.Field{{
			Key:   "status",
			Label: "Status",
			EnumValues: []reportcatalog.EnumValue{
				{Value: "Active", Label: "Active"},
			},
		}},
		Edges: []reportcatalog.Edge{{Name: "parent", Label: "Customer"}},
	}
	detail := &services.ResourcePermissionDetail{MaxSensitivity: permission.SensitivityConfidential}

	english := catalogEntityToModel(i18n.EN, entity, detail)
	spanish := catalogEntityToModel(i18n.ES, entity, detail)

	assert.Equal(t, "Customer", english.Label)
	require.Len(t, spanish.Fields, 1)
	require.Len(t, spanish.Fields[0].EnumValues, 1)
	require.Len(t, spanish.Edges, 1)

	for source, got := range map[string]string{
		"Customer":  spanish.Label,
		"Customers": spanish.PluralLabel,
		"Status":    spanish.Fields[0].Label,
		"Active":    spanish.Fields[0].EnumValues[0].Label,
	} {
		want := i18n.Translate(i18n.ES, source)
		require.NotEqual(t, source, want, "the Spanish catalog has no entry for %q", source)
		assert.Equal(t, want, got)
	}
	assert.Equal(t, spanish.Label, spanish.Edges[0].Label)
	assert.Equal(t, "customer", spanish.Key)
	assert.Equal(t, "Active", spanish.Fields[0].EnumValues[0].Value)
}
