package report_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// describe_report summarizes a column's field as "assignment.primaryWorker.firstName".
// A model that rebuilds a definition from that summary writes exactly this
// shape, and it used to decode to an empty ref that failed after a person
// had approved it.
func TestDecodeDefinition_AcceptsAFlattenedDottedField(t *testing.T) {
	t.Parallel()

	definition, err := report.DecodeDefinition(map[string]any{
		"entity": "shipment_move",
		"columns": []any{
			map[string]any{
				"id":    "first_name",
				"kind":  "dimension",
				"field": "assignment.primaryWorker.firstName",
			},
			map[string]any{"id": "moves", "kind": "measure", "agg": "count", "field": "id"},
			map[string]any{
				"id":   "fleet",
				"kind": "dimension",
				"ref":  map[string]any{"field": "assignment.primaryWorker.fleetCode.code"},
			},
		},
		"filters": map[string]any{
			"op": "and",
			"filters": []any{
				map[string]any{
					"operator": "eq",
					"ref":      map[string]any{"field": "shipment.customer.name"},
					"value":    "Acme",
				},
			},
		},
	})
	require.NoError(t, err)

	require.Len(t, definition.Columns, 3)
	assert.Equal(t, []string{"assignment", "primaryWorker"}, definition.Columns[0].Ref.Path)
	assert.Equal(t, "firstName", definition.Columns[0].Ref.Field)
	assert.Empty(t, definition.Columns[1].Ref.Path)
	assert.Equal(t, "id", definition.Columns[1].Ref.Field)
	assert.Equal(
		t,
		[]string{"assignment", "primaryWorker", "fleetCode"},
		definition.Columns[2].Ref.Path,
	)
	assert.Equal(t, "code", definition.Columns[2].Ref.Field)

	require.NotNil(t, definition.Filters)
	require.Len(t, definition.Filters.Filters, 1)
	assert.Equal(t, []string{"shipment", "customer"}, definition.Filters.Filters[0].Ref.Path)
	assert.Equal(t, "name", definition.Filters.Filters[0].Ref.Field)
}

func TestDecodeDefinition_LeavesACanonicalRefAlone(t *testing.T) {
	t.Parallel()

	definition, err := report.DecodeDefinition(map[string]any{
		"entity": "shipment_move",
		"columns": []any{
			map[string]any{
				"id":   "fleet",
				"kind": "dimension",
				"ref": map[string]any{
					"path":  []any{"assignment", "primaryWorker"},
					"field": "fleetCode.code",
				},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"assignment", "primaryWorker"}, definition.Columns[0].Ref.Path)
	assert.Equal(
		t,
		"fleetCode.code",
		definition.Columns[0].Ref.Field,
		"a ref that already has a path is not second-guessed",
	)
}

// A column with no field at all is refused up front, naming the column, so
// the model fixes its call rather than a person approving a proposal that
// fails with "unknown field".
func TestDecodeDefinition_RefusesAColumnWithNoField(t *testing.T) {
	t.Parallel()

	_, err := report.DecodeDefinition(map[string]any{
		"entity": "shipment_move",
		"columns": []any{
			map[string]any{
				"id":    "last_move",
				"kind":  "measure",
				"agg":   "max",
				"label": "Last Move",
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"last_move"`)
	assert.Contains(t, err.Error(), "ref")
}

func TestDecodeDefinition_AllowsAComputedColumnWithoutAField(t *testing.T) {
	t.Parallel()

	definition, err := report.DecodeDefinition(map[string]any{
		"entity": "shipment",
		"columns": []any{
			map[string]any{
				"id":   "revenue",
				"kind": "measure",
				"agg":  "sum",
				"ref":  map[string]any{"field": "totalChargeAmount"},
			},
			map[string]any{
				"id":       "per_mile",
				"kind":     "computed",
				"computed": map[string]any{"op": "divide", "leftId": "revenue", "rightValue": 100},
			},
		},
	})
	require.NoError(t, err)
	assert.Len(t, definition.Columns, 2)
}

func TestDefinitionJSONSchema_RequiresARefOnEveryColumn(t *testing.T) {
	t.Parallel()

	schema := report.DefinitionJSONSchema()
	columns := schema["properties"].(map[string]any)["columns"].(map[string]any)
	items := columns["items"].(map[string]any)
	assert.Contains(t, items["required"], "ref")
}
