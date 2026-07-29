package dispatchconsolerepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const testWindowStart = int64(1_700_000_000)

const testWindowEnd = int64(1_700_172_800)

func renderMoveFilters(t *testing.T, filter *repositories.DispatchBoardFilter) string {
	t.Helper()

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	bunDB := bun.NewDB(db, pgdialect.New())

	q := bunDB.NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		ColumnExpr("sm.id").
		Join(shipmentJoin).
		Join(customerJoin)
	q = joinMoveStopEdges(q)
	q = applyMoveFilters(q, filter)

	return q.String()
}

// TestMoveFiltersKeepPastDueOpenMoves pins the invariant the console exists for: an
// uncovered move whose appointments have already lapsed is the most urgent row on the
// board. The window's lower bound trims finished work, so it must never be allowed to
// silently empty the board of overdue loads.
func TestMoveFiltersKeepPastDueOpenMoves(t *testing.T) {
	t.Parallel()

	sql := renderMoveFilters(t, &repositories.DispatchBoardFilter{
		WindowStart: testWindowStart,
		WindowEnd:   testWindowEnd,
	})

	require.Contains(
		t,
		sql,
		"COALESCE(dest.window_end, dest.window_start, orig.window_start) >= 1700000000",
		"window lower bound must still be applied",
	)
	require.Regexp(
		t,
		`window_start\) >= 1700000000\) OR \(.*'New', 'Assigned', 'InTransit'`,
		sql,
		"open moves must be exempt from the window lower bound",
	)
	require.Contains(
		t,
		sql,
		"COALESCE(orig.window_start, 0) <= 1700172800",
		"window upper bound must still bound the planning horizon",
	)
}

// TestMoveFiltersDefaultToOpenStatuses guards the board against defaulting to every move
// ever created when a caller supplies no status filter.
func TestMoveFiltersDefaultToOpenStatuses(t *testing.T) {
	t.Parallel()

	sql := renderMoveFilters(t, &repositories.DispatchBoardFilter{})

	require.Contains(t, sql, "'New', 'Assigned', 'InTransit'")
	require.NotContains(t, sql, "COALESCE(dest.window_end")
	require.NotContains(t, sql, "COALESCE(orig.window_start, 0) <=")
}

// TestMoveFiltersHonorExplicitStatuses keeps an explicit status request authoritative;
// the exemption is scoped to the window bound, not to what the caller asked for.
func TestMoveFiltersHonorExplicitStatuses(t *testing.T) {
	t.Parallel()

	sql := renderMoveFilters(t, &repositories.DispatchBoardFilter{
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusCompleted},
		WindowStart:  testWindowStart,
	})

	require.Regexp(t, `sm"?\."?status"? IN \('Completed'\)`, sql)
	require.Contains(t, sql, "COALESCE(dest.window_end, dest.window_start, orig.window_start) >=")
}
