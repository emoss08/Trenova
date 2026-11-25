package graphqllisttool

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckQuery_ListWithoutTextFieldsRefusesAQueryNamingItsArguments(t *testing.T) {
	t.Parallel()

	tool := &UnpagedTool{
		spec:   readspec.List{EntityPlural: "facilities"},
		Params: []graphqlread.Param{{Name: "shipmentId"}, {Name: "urgency"}},
	}

	err := tool.checkQuery(map[string]any{"query": "S26110001005205"})

	require.EqualError(t, err, `facilities takes no free-text query, so "S26110001005205" `+
		`would match nothing; narrow with its arguments instead: shipmentId, urgency`)
	assert.NoError(t, tool.checkQuery(map[string]any{"query": "  "}))
	assert.NoError(t, tool.checkQuery(map[string]any{}))
}

func TestCheckQuery_ListWithTextFieldsTakesAQuery(t *testing.T) {
	t.Parallel()

	tool := &UnpagedTool{spec: readspec.List{Matches: []string{"occurrence.shipmentProNumber"}}}

	assert.NoError(t, tool.checkQuery(map[string]any{"query": "S26110001005205"}))
}

func TestMatchesQuery_FindsADetentionChargeByItsShipmentsProNumber(t *testing.T) {
	t.Parallel()

	row := map[string]any{"occurrence": map[string]any{
		"shipmentProNumber": "S26110001005205",
		"locationName":      "Prairie Industrial Supply Birmingham",
	}}
	paths := []string{"occurrence.shipmentProNumber", "occurrence.locationName"}

	assert.True(t, matchesQuery(row, paths, "s26110001005205"))
	assert.True(t, matchesQuery(row, paths, "birmingham"))
	assert.False(t, matchesQuery(row, paths, "kestrel"))
}
