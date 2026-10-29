package selectoptionsresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/stretchr/testify/require"
)

// Every value the schema offers has to reach a resolver: an enum value with no
// registry entry is accepted by the schema and then refused at runtime with a
// validation error, which reads to the client as a broken picker.
func TestSelectOptionRegistry_CoversEverySchemaResource(t *testing.T) {
	t.Parallel()

	schema := generated.NewExecutableSchema(generated.Config{}).Schema()
	definition, ok := schema.Types["SelectOptionResource"]
	require.True(t, ok)
	require.NotEmpty(t, definition.EnumValues)

	registry := (&Deps{Core: &resolvercore.Core{}}).SelectOptionRegistry()
	for _, value := range definition.EnumValues {
		t.Run(value.Name, func(t *testing.T) {
			entry, registered := registry[gqlmodel.SelectOptionResource(value.Name)]
			require.Truef(
				t,
				registered,
				"SelectOptionResource %q has no registry entry",
				value.Name,
			)
			require.NotNilf(
				t,
				entry.Resolve,
				"SelectOptionResource %q has a nil resolver",
				value.Name,
			)
		})
	}
}
