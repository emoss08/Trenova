package dispatchconsolerepository

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/require"
)

const testWindowStart = int64(1_700_000_000)

const testWindowEnd = int64(1_700_172_800)

func renderMoveFilters(
	t *testing.T,
	filter *shipmentports.DispatchBoardFilter,
) (sql, args string) {
	t.Helper()

	q := boardMoves().Pick(shipmentdb.ShipmentMoves.ID).Where(moveFilters(filter)...)
	text, values := keeltest.Render(t, q)

	return text, fmt.Sprint(values)
}

// TestMoveFiltersKeepPastDueOpenMoves pins the invariant the console exists for: an
// uncovered move whose appointments have already lapsed is the most urgent row on the
// board. The window's lower bound trims finished work, so it must never be allowed to
// silently empty the board of overdue loads.
func TestMoveFiltersKeepPastDueOpenMoves(t *testing.T) {
	t.Parallel()

	sql, args := renderMoveFilters(t, &shipmentports.DispatchBoardFilter{
		WindowStart: testWindowStart,
		WindowEnd:   testWindowEnd,
	})

	require.Regexp(
		t,
		`\(\(coalesce\([a-z_]+\."scheduled_window_end", [a-z_]+\."scheduled_window_start", [a-z_]+\."scheduled_window_start"\) >= \$\d+\) OR [a-z_]+\."status" = ANY\(\$\d+\)\)`,
		sql,
		"open moves must be exempt from the window lower bound",
	)
	require.Regexp(
		t,
		`\(coalesce\([a-z_]+\."scheduled_window_start", \$\d+\) <= \$\d+\)`,
		sql,
		"window upper bound must still bound the planning horizon",
	)
	require.Contains(t, args, fmt.Sprint(testWindowStart))
	require.Contains(t, args, fmt.Sprint(testWindowEnd))
	require.Contains(t, args, "New Assigned InTransit")
}

// TestMoveFiltersDefaultToOpenStatuses guards the board against defaulting to every move
// ever created when a caller supplies no status filter.
func TestMoveFiltersDefaultToOpenStatuses(t *testing.T) {
	t.Parallel()

	sql, args := renderMoveFilters(t, &shipmentports.DispatchBoardFilter{})

	require.Contains(t, args, "New Assigned InTransit")
	require.NotContains(t, sql, "coalesce(")
	require.Equal(t, 1, strings.Count(sql, `"status" = ANY(`))
}

// TestMoveFiltersHonorExplicitStatuses keeps an explicit status request authoritative;
// the exemption is scoped to the window bound, not to what the caller asked for.
func TestMoveFiltersHonorExplicitStatuses(t *testing.T) {
	t.Parallel()

	sql, args := renderMoveFilters(t, &shipmentports.DispatchBoardFilter{
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusCompleted},
		WindowStart:  testWindowStart,
	})

	require.Contains(t, args, "[Completed]")
	require.Contains(t, sql, ") >= $")
}
