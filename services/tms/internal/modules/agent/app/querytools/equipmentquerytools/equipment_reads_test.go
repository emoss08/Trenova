package equipmentquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqllisttool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
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

	for _, name := range []string{toolListTrailers, querytool.ToolListTractors} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reader := &querytooltest.KeyedReader{}
			tool, err := graphqllisttool.New(
				querytooltest.ListSpecNamed(t, ReadSpecs(), name),
				reader,
				&mocks.DenyResourcesPermissionEngine{},
			)
			require.NoError(t, err)

			properties, _ := tool.ParamSchema()["properties"].(map[string]any)
			require.Contains(t, properties, "hasInProgressMove")
			assert.Contains(t, tool.Description(), "free now (hasInProgressMove false)")

			_, err = tool.Query(t.Context(), querytooltest.DispatchCall(map[string]any{
				"hasInProgressMove": false,
			}))
			require.NoError(t, err)
			filters, _ := querytooltest.SentInput(t, reader)[graphqlread.InputFieldFilters].([]map[string]any)
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

	for _, name := range []string{toolListTrailers, querytool.ToolListTractors} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			read := graphqlread.Reads[name]
			reader := &querytooltest.KeyedReader{
				Answers: map[string]*graphqlreadports.GraphQLReadResult{
					read.Operation: {Data: map[string]any{read.RootField: map[string]any{
						graphqlread.ConnectionEdges: []any{
							map[string]any{graphqlread.ConnectionNode: map[string]any{
								"id": "tr_1", "code": "1285", "licensePlateNumber": "CRN4714",
								"status": "OutOfService",
							}},
						},
						graphqlread.ConnectionPageInfo: map[string]any{
							graphqlread.PageHasNext: false,
						},
					}}},
				},
			}
			tool, err := graphqllisttool.New(
				querytooltest.ListSpecNamed(t, ReadSpecs(), name),
				reader,
				&mocks.DenyResourcesPermissionEngine{},
			)
			require.NoError(t, err)

			properties, _ := tool.ParamSchema()["properties"].(map[string]any)
			assert.Contains(t, properties, "unitNumber")
			assert.NotContains(t, properties, "code")
			assert.Contains(t, tool.Description(), "Name a unit by its unitNumber")

			result, err := tool.Query(t.Context(), querytooltest.DispatchCall(map[string]any{
				"unitNumber": "1285",
			}))
			require.NoError(t, err)
			filters, _ := querytooltest.SentInput(t, reader)[graphqlread.InputFieldFilters].([]map[string]any)
			require.NotEmpty(t, filters)
			assert.Equal(t, "code", filters[0]["field"], "the unit number narrows by the code")

			outcome := querytooltest.EncodedDocument(t, result)
			row := querytooltest.ObjectAt(t, outcome["items"], 0)
			assert.Equal(t, "1285", row["unitNumber"])
			assert.NotContains(t, row, "code")
			assert.Contains(t, outcome["columns"], "unitNumber")
		})
	}
}
