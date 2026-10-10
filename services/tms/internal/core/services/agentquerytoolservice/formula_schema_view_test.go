package agentquerytoolservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formulaassistantservice"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func schemaReference() *formulaassistantservice.Reference {
	return &formulaassistantservice.Reference{
		SchemaID: "shipment",
		Variables: []formulatemplatetypes.SchemaVariableInfo{
			{
				Name: "requiresTemperatureControl", Type: "boolean", Computed: true,
				Description: "Whether the shipment requires temperature control. True when a minimum or maximum is set.",
			},
			{Name: "totalDistance", Type: "number", Description: "Total distance in miles."},
			{Name: "hasHazmat", Type: "boolean", Description: "Whether any commodity is hazardous."},
		},
		Functions: []formulatemplatetypes.SchemaFunctionInfo{
			{Name: "ceil", Signature: "ceil(x)", Description: "Rounds up.", Example: "ceil(1.2) = 2"},
		},
	}
}

func TestSchemaView_ListsEverythingShortened(t *testing.T) {
	t.Parallel()

	view := schemaView(schemaReference(), "")

	require.Len(t, view.Variables, 3)
	assert.Equal(t, "Whether the shipment requires temperature control.", view.Variables[0].Description)
	require.Len(t, view.Functions, 1)
	assert.Equal(t, "ceil(x)", view.Functions[0].Signature)
	assert.Empty(t, view.Functions[0].Example, "examples come with a query")
	assert.Contains(t, view.Note, "query")
}

func TestSchemaView_FindsAReeferFieldByWhatPeopleCallIt(t *testing.T) {
	t.Parallel()

	view := schemaView(schemaReference(), "reefer")

	require.Len(t, view.Variables, 1)
	assert.Equal(t, "requiresTemperatureControl", view.Variables[0].Name)
	assert.True(t, strings.HasSuffix(view.Variables[0].Description, "is set."), "in full")
}

func TestSchemaView_SaysWhenNoFieldHoldsIt(t *testing.T) {
	t.Parallel()

	view := schemaView(schemaReference(), "team service")

	assert.Empty(t, view.Variables)
	assert.Empty(t, view.Functions)
	assert.Contains(t, view.Note, "no shipment field holds it")
}

// "baseRate totalDistance" found nothing: the query term was "baserate"
// while names were split into "base" and "rate", so gpt-6-luna was told the
// existing formula's own variables were not shipment fields and asked the
// person for a rate.
func TestSchemaView_FindsAVariableByItsOwnName(t *testing.T) {
	t.Parallel()

	reference := schemaReference()
	reference.Variables = append(reference.Variables, formulatemplatetypes.SchemaVariableInfo{
		Name: "baseRate", Type: "number", Description: "Base rate per unit for freight charge calculation",
	})

	view := schemaView(reference, "baseRate totalDistance")

	names := make([]string, 0, len(view.Variables))
	for _, variable := range view.Variables {
		names = append(names, variable.Name)
	}
	assert.ElementsMatch(t, []string{"baseRate", "totalDistance"}, names)
}
