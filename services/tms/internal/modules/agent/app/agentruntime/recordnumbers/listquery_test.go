package recordnumbers

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordnumberports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shipmentListSchema(fields ...string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
			"filters": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"field":    map[string]any{"type": "string", "enum": fields},
						"operator": map[string]any{"type": "string", "enum": []string{"eq", "in"}},
						"value":    map[string]any{"type": "string"},
						"values": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
	}
}

func listQuery(args map[string]any, schema map[string]any) *ListQuery {
	return &ListQuery{
		Actor:      runtimetest.Actor(),
		DataAccess: permission.SensitivityInternal,
		Resource:   permission.ResourceShipment,
		Schema:     schema,
		Args:       args,
	}
}

func twoShipments() (*fakeNumbers, pulid.ID, pulid.ID) {
	first, second := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	records := defaultNumbers()
	records.records = recordnumberports.NumberedRecords{shipmentKind: {
		"SEED-SHP-008": {{ID: first, Number: "SEED-SHP-008"}},
		"SEED-SHP-010": {{ID: second, Number: "SEED-SHP-010"}},
	}}

	return records, first, second
}

func TestListQuery_ListsExactlyTheRecordsAQueryOfNumbersNames(t *testing.T) {
	t.Parallel()

	records, _, _ := twoShipments()
	listed, err := newResolver(records, nil).ListQuery(t.Context(), listQuery(map[string]any{
		"query":  "SEED-SHP-008 and SEED-SHP-010",
		"fields": []any{"proNumber", "totalCharge"},
	}, shipmentListSchema("status", "proNumber")))

	require.NoError(t, err)
	require.Len(t, records.asked, 1)
	assert.Equal(t, map[recordkind.Kind][]string{
		shipmentKind: {"SEED-SHP-008", "SEED-SHP-010"},
	}, records.asked[0])
	assert.Equal(t, map[string]any{
		"fields": []any{"proNumber", "totalCharge"},
		"filters": []any{map[string]any{
			"field":    "proNumber",
			"operator": "in",
			"values":   []any{"SEED-SHP-008", "SEED-SHP-010"},
		}},
	}, listed.Args)
	assert.Equal(t, "The words were PRO numbers, so this lists exactly the shipments they "+
		"name: SEED-SHP-008, SEED-SHP-010.", listed.Note)
}

func TestListQuery_KeepsTheFiltersAlreadyAsked(t *testing.T) {
	t.Parallel()

	records, _, _ := twoShipments()
	status := map[string]any{"field": "status", "operator": "in", "values": []any{"Delayed"}}
	listed, err := newResolver(records, nil).ListQuery(t.Context(), listQuery(map[string]any{
		"query":   "SEED-SHP-008, SEED-SHP-010",
		"filters": []any{status},
	}, shipmentListSchema("status", "proNumber")))

	require.NoError(t, err)
	assert.Equal(t, []any{status, map[string]any{
		"field":    "proNumber",
		"operator": "in",
		"values":   []any{"SEED-SHP-008", "SEED-SHP-010"},
	}}, listed.Args["filters"])
}

func TestListQuery_LeavesTheSearchAlone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		query  string
		schema map[string]any
		reads  int
	}{
		{name: "words that are not numbers", query: "sunbelt chicago", reads: 0},
		{name: "a number beside a word", query: "SEED-SHP-008 sunbelt", reads: 0},
		{name: "an id", query: "shp_01M7V48EQGE9TBM60WEQJJGQTR", reads: 0},
		{name: "a number no record carries", query: "SEED-SHP-008 SEED-SHP-099", reads: 1},
		{
			name:   "a list that cannot filter on the number",
			query:  "SEED-SHP-008",
			schema: shipmentListSchema("status"),
			reads:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			records, _, _ := twoShipments()
			schema := tt.schema
			if schema == nil {
				schema = shipmentListSchema("status", "proNumber")
			}
			args := map[string]any{"query": tt.query}
			listed, err := newResolver(records, nil).ListQuery(t.Context(), listQuery(args, schema))

			require.NoError(t, err)
			assert.Equal(t, args, listed.Args)
			assert.Empty(t, listed.Note)
			assert.Len(t, records.asked, tt.reads)
		})
	}
}

func TestListQuery_LeavesTheSearchAloneWhenTheNumberIsWithheld(t *testing.T) {
	t.Parallel()

	records, _, _ := twoShipments()
	args := map[string]any{"query": "SEED-SHP-008"}
	listed, err := newResolver(records, map[string]bool{"shipment:read": true}).
		ListQuery(t.Context(), listQuery(args, shipmentListSchema("proNumber")))

	require.NoError(t, err)
	assert.Equal(t, args, listed.Args)
	assert.Empty(t, records.asked)
}

func TestListQuery_ReportsAFailedRead(t *testing.T) {
	t.Parallel()

	records := defaultNumbers()
	records.err = errors.New("connection reset")
	_, err := newResolver(records, nil).ListQuery(t.Context(), listQuery(
		map[string]any{"query": "SEED-SHP-008"}, shipmentListSchema("proNumber"),
	))

	require.ErrorContains(t, err, "connection reset")
}
