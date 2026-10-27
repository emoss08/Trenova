package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
equipment_list_trailers with status Available listed TRL-001 while it ran
SEED-DET-001; Haiku assigned it, and the next stop actual was refused "Trailer
is currently in progress on another move". Status is stored, not read from
the open assignments, so each row carries the move in progress and the list
narrows to the units free now.
*/
func TestEquipmentLists_NarrowToTheUnitsFreeNow(t *testing.T) {
	t.Parallel()

	for _, name := range []string{toolListTrailers, toolListTractors} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reader := &keyedReader{}
			tool, err := newGraphQLListTool(keyedSpec(t, name), reader,
				&mocks.DenyResourcesPermissionEngine{})
			require.NoError(t, err)

			properties, _ := tool.ParamSchema()["properties"].(map[string]any)
			require.Contains(t, properties, "hasInProgressMove")
			assert.Contains(t, tool.Description(), "free now (hasInProgressMove false)")

			_, err = tool.Query(t.Context(), dispatchCall(map[string]any{
				"hasInProgressMove": false,
			}))
			require.NoError(t, err)
			filters, _ := sentInput(t, reader)[inputFieldFilters].([]map[string]any)
			assert.Contains(t, filters, map[string]any{
				"field": "inProgressMoveId", "operator": "isnull", "value": nil,
			})
		})
	}
}
