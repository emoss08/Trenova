package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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

/*
Asked what in the fleet was down, Haiku named trailers by their plates
("Trailer MAH2221") and no unit by its code: the row's code read as one more
field beside the plate. Each row now names the unit by its unitNumber, which is
also what the list narrows by.
*/
func TestEquipmentLists_NameEachUnitByItsUnitNumber(t *testing.T) {
	t.Parallel()

	for _, name := range []string{toolListTrailers, toolListTractors} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			read := graphQLReads[name]
			reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
				read.Operation: {Data: map[string]any{read.RootField: map[string]any{
					connectionEdges: []any{map[string]any{connectionNode: map[string]any{
						"id": "tr_1", "code": "1285", "licensePlateNumber": "CRN4714",
						"status": "OutOfService",
					}}},
					connectionPageInfo: map[string]any{pageHasNext: false},
				}}},
			}}
			tool, err := newGraphQLListTool(keyedSpec(t, name), reader,
				&mocks.DenyResourcesPermissionEngine{})
			require.NoError(t, err)

			properties, _ := tool.ParamSchema()["properties"].(map[string]any)
			assert.Contains(t, properties, "unitNumber")
			assert.NotContains(t, properties, "code")
			assert.Contains(t, tool.Description(), "Name a unit by its unitNumber")

			result, err := tool.Query(t.Context(), dispatchCall(map[string]any{
				"unitNumber": "1285",
			}))
			require.NoError(t, err)
			filters, _ := sentInput(t, reader)[inputFieldFilters].([]map[string]any)
			require.NotEmpty(t, filters)
			assert.Equal(t, "code", filters[0]["field"], "the unit number narrows by the code")

			outcome := asMap(t, result)
			row := objectAt(t, outcome["items"], 0)
			assert.Equal(t, "1285", row["unitNumber"])
			assert.NotContains(t, row, "code")
			assert.Contains(t, outcome["columns"], "unitNumber")
		})
	}
}
