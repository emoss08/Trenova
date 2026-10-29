package graphqlread

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableShapedInputsHoldTheTablesFields(t *testing.T) {
	t.Parallel()

	table := []Field{
		{Name: InputFirst}, {Name: InputAfter}, {Name: InputQuery},
		{Name: InputFieldFilters}, {Name: InputSort}, {Name: "status"},
	}
	assert.True(t, tableShaped(table))
	assert.False(t, tableShaped(table[1:]), "an input with no first is not a table's")
}

func TestInputFieldParamsTypeTheScreensOwnFilters(t *testing.T) {
	t.Parallel()

	from := inputFieldParams(&Field{Name: "startDateFrom", Type: graphQLTypeTimestamp})
	require.Len(t, from, 1)
	assert.Equal(t, ParamDayFrom, from[0].Kind)

	to := inputFieldParams(&Field{Name: "startDateTo", Type: graphQLTypeTimestamp})
	require.Len(t, to, 1)
	assert.Equal(t, ParamDayTo, to[0].Kind)

	assert.Empty(t, inputFieldParams(&Field{Name: "decidedAt", Type: graphQLTypeTimestamp}),
		"a lone instant is not a range end a day can bound")

	status := inputFieldParams(&Field{
		Name: "status", Type: "PTOStatus", Kind: graphQLKindEnum, Enum: []string{"Requested"},
	})
	require.Len(t, status, 1)
	assert.Equal(t, ParamEquals, status[0].Kind, "a single enum on the input takes one value")
}

func TestCompileInputsSendsOnInputParametersAsInputFields(t *testing.T) {
	t.Parallel()

	clk := querytool.ClockFor(&agenttoolports.QueryToolParams{Timezone: "UTC"})
	params := []Param{
		{Name: "status", Field: "status", Kind: ParamEquals, OnInput: true},
		{Name: "startDateFrom", Field: "startDateFrom", Kind: ParamDayFrom, OnInput: true},
		{Name: "startDateTo", Field: "startDateTo", Kind: ParamDayTo, OnInput: true},
		{Name: "code", Field: "code", Kind: ParamContains},
	}
	args := map[string]any{
		"status": "Requested", "startDateFrom": "2026-10-20", "startDateTo": "2026-10-20",
		"code": "abc",
	}

	inputs, err := CompileInputs(params, args, clk)
	require.NoError(t, err)
	assert.Equal(t, "Requested", inputs["status"])
	assert.Less(t, inputs["startDateFrom"], inputs["startDateTo"])
	assert.NotContains(t, inputs, "code", "a record filter is not an input field")

	filters, err := CompileFilters(params, args, clk)
	require.NoError(t, err)
	require.Len(t, filters, 1, "an on-input parameter is not a field filter")
	assert.Equal(t, "code", filters[0][querytool.ParamField])
}
