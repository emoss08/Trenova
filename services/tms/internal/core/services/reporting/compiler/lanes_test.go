package compiler

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func laneReport(t *testing.T, c *Compiler, columns ...report.ColumnSpec) string {
	t.Helper()
	return mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   columns,
	}).SQL
}

// A lane accessor has to resolve to exactly one row, ordered, or the "origin"
// of a shipment would be whichever stop the planner reached first.
func TestOriginStopEmitsOrderedLateral(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sql := laneReport(t, c,
		dim("c_origin", "status", "originStop"),
		measure("c_count", reportcatalog.AggCount, "id"),
	)

	assert.Contains(t, sql, "LEFT JOIN LATERAL (SELECT t1p1.*")
	assert.Contains(t, sql, "FROM shipment_moves AS t1p0")
	assert.Contains(t, sql, "LEFT JOIN stops AS t1p1")
	assert.Contains(t, sql,
		"ORDER BY t1p0.sequence ASC NULLS LAST, t1p1.sequence ASC NULLS LAST LIMIT 1) AS t1 ON TRUE")
}

// Destination is the same walk in reverse. NULLS LAST is what keeps a move with
// no stops from winning a descending sort and picking nothing.
func TestDestinationStopOrdersDescendingNullsLast(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sql := laneReport(t, c,
		dim("c_dest", "status", "destinationStop"),
		measure("c_count", reportcatalog.AggCount, "id"),
	)

	assert.Contains(t, sql,
		"ORDER BY t1p0.sequence DESC NULLS LAST, t1p1.sequence DESC NULLS LAST LIMIT 1)")
	assert.NotContains(t, sql, "NULLS FIRST")
}

// The lateral is correlated to the shipment and tenant-scoped at every hop;
// without that a lane column would read another organization's stops.
func TestPickLateralIsCorrelatedAndTenantScoped(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sql := laneReport(t, c,
		dim("c_origin", "status", "originStop"),
		measure("c_count", reportcatalog.AggCount, "id"),
	)

	assert.Contains(t, sql, "t1p0.shipment_id = t0.id")
	assert.Contains(t, sql, "t1p0.organization_id = ?")
	assert.Contains(t, sql, "t1p1.organization_id = ?")
}

// The whole point of picking a row: a lane column must not fan the shipment out
// across its stops, so measures stay unmultiplied and no lateral aggregate is
// planned for the path.
func TestLaneColumnsDoNotFanOutMeasures(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sql := laneReport(t, c,
		dim("c_origin", "status", "originStop"),
		dim("c_dest", "status", "destinationStop"),
		measure("c_revenue", reportcatalog.AggSum, "totalChargeAmount"),
	)

	assert.Contains(t, sql, "SUM(t0.total_charge_amount)")
	assert.Equal(t, 2, strings.Count(sql, "LEFT JOIN LATERAL"),
		"one lateral per lane accessor and no aggregate lateral")
	assert.Contains(t, sql, "GROUP BY")
}

// Both ends of the lane in one report is the question people actually ask, and
// each has to get its own independent pick.
func TestOriginAndDestinationGetSeparateAliases(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sql := laneReport(t, c,
		dim("c_origin", "status", "originStop"),
		dim("c_dest", "status", "destinationStop"),
		measure("c_count", reportcatalog.AggCount, "id"),
	)

	assert.Contains(t, sql, ") AS t1 ON TRUE")
	assert.Contains(t, sql, ") AS t2 ON TRUE")
	assert.Contains(t, sql, "ASC NULLS LAST")
	assert.Contains(t, sql, "DESC NULLS LAST")
}

// A to-one edge that happens to be a pick must still be traversable onward, so
// grouping by the origin's location works like any other nested to-one hop.
func TestPickEdgeSupportsFurtherToOneHops(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	entity, ok := reportcatalog.Default.Entity("stop")
	require.True(t, ok)
	if _, hasLocation := entity.Edge("location"); !hasLocation {
		t.Skip("stop has no location edge in the catalog")
	}

	sql := laneReport(t, c,
		dim("c_origin_loc", "name", "originStop", "location"),
		measure("c_count", reportcatalog.AggCount, "id"),
	)

	assert.Contains(t, sql, "LEFT JOIN LATERAL")
	assert.Contains(t, sql, "locations AS t2")
}

func TestPickEdgeIsToOne(t *testing.T) {
	entity, ok := reportcatalog.Default.Entity("shipment")
	require.True(t, ok)

	for _, name := range []string{"originStop", "destinationStop"} {
		edge, found := entity.Edge(name)
		require.True(t, found, name)
		assert.Equal(t, reportcatalog.CardinalityOne, edge.Cardinality, name)
		assert.NotNil(t, edge.Pick, name)
		assert.Equal(t, "stop", edge.Target, name)
		assert.Empty(t, edge.Join, name, "a pick edge joins through its lateral, not a key pair")
	}
}

// The via chain carries no output columns but the query reads it, so it has to
// reach both the authorization sweep and — more sharply — the cache's
// data-version vector. Reorder a shipment's moves and its origin stop changes
// without the stops table being written at all.
func TestPickViaEntitiesAreReferenced(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_origin", "status", "originStop"),
			measure("c_count", reportcatalog.AggCount, "id"),
		},
	})

	assert.Contains(t, compiled.ReferencedEntities, "shipment_move")
	assert.Contains(t, compiled.ReferencedEntities, "stop")
	assert.Contains(t, compiled.ReferencedTables, "shipment_moves")
	assert.Contains(t, compiled.ReferencedTables, "stops")
}
