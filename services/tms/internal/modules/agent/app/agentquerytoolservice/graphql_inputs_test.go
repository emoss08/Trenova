package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableShapedInputsHoldTheTablesFields(t *testing.T) {
	t.Parallel()

	table := []graphQLField{
		{Name: inputFirst}, {Name: inputAfter}, {Name: inputQuery},
		{Name: inputFieldFilters}, {Name: inputSort}, {Name: "status"},
	}
	assert.True(t, tableShaped(table))
	assert.False(t, tableShaped(table[1:]), "an input with no first is not a table's")
}

func TestInputFieldParamsTypeTheScreensOwnFilters(t *testing.T) {
	t.Parallel()

	from := inputFieldParams(&graphQLField{Name: "startDateFrom", Type: graphQLTypeTimestamp})
	require.Len(t, from, 1)
	assert.Equal(t, paramDayFrom, from[0].Kind)

	to := inputFieldParams(&graphQLField{Name: "startDateTo", Type: graphQLTypeTimestamp})
	require.Len(t, to, 1)
	assert.Equal(t, paramDayTo, to[0].Kind)

	assert.Empty(t, inputFieldParams(&graphQLField{Name: "decidedAt", Type: graphQLTypeTimestamp}),
		"a lone instant is not a range end a day can bound")

	status := inputFieldParams(&graphQLField{
		Name: "status", Type: "PTOStatus", Kind: graphQLKindEnum, Enum: []string{"Requested"},
	})
	require.Len(t, status, 1)
	assert.Equal(t, paramEquals, status[0].Kind, "a single enum on the input takes one value")
}

func TestCompileInputsSendsOnInputParametersAsInputFields(t *testing.T) {
	t.Parallel()

	clk := clockFor(&agenttoolports.QueryToolParams{Timezone: "UTC"})
	params := []graphQLParam{
		{Name: "status", Field: "status", Kind: paramEquals, OnInput: true},
		{Name: "startDateFrom", Field: "startDateFrom", Kind: paramDayFrom, OnInput: true},
		{Name: "startDateTo", Field: "startDateTo", Kind: paramDayTo, OnInput: true},
		{Name: "code", Field: "code", Kind: paramContains},
	}
	args := map[string]any{
		"status": "Requested", "startDateFrom": "2026-10-20", "startDateTo": "2026-10-20",
		"code": "abc",
	}

	inputs, err := compileInputs(params, args, clk)
	require.NoError(t, err)
	assert.Equal(t, "Requested", inputs["status"])
	assert.Less(t, inputs["startDateFrom"], inputs["startDateTo"])
	assert.NotContains(t, inputs, "code", "a record filter is not an input field")

	filters, err := compileFilters(params, args, clk)
	require.NoError(t, err)
	require.Len(t, filters, 1, "an on-input parameter is not a field filter")
	assert.Equal(t, "code", filters[0][paramField])
}
