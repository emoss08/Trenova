package benchreport

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/deskbench/benchstats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareCells_LinesUpEachModelAgainstTheEarlierRun(t *testing.T) {
	t.Parallel()

	share := func(value float64) *float64 { return &value }
	before := &Report{Summary: Summary{Cells: []CellRow{{
		Provider: "claude-haiku-5-5@low",
		PassK:    benchstats.Stat{Mean: 0.6, N: 4},
		Measures: Measures{
			ModelCalls:      benchstats.Stat{Mean: 6, N: 4},
			LaterCacheShare: share(0.31),
			FindFirstShare:  share(0.4),
		},
	}}}}
	after := &Report{Summary: Summary{Cells: []CellRow{
		{
			Provider: "claude-haiku-5-5@low",
			PassK:    benchstats.Stat{Mean: 0.65, N: 4},
			Measures: Measures{
				ModelCalls:      benchstats.Stat{Mean: 4.5, N: 4},
				LaterCacheShare: share(0.92),
				FindFirstShare:  share(0.1),
			},
		},
		{Provider: "new-cell"},
	}}}

	changes := CompareCells(before, after)

	require.Len(t, changes, 1, "a cell the earlier run did not have is not compared")
	var out strings.Builder
	renderCellChanges(&out, changes)
	rendered := out.String()
	assert.Contains(t, rendered, "CACHED, CALLS 2+")
	assert.Contains(t, rendered, "31% → 92%")
	assert.Contains(t, rendered, "40% → 10%")
}
