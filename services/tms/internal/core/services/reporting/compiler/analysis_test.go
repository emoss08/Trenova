package compiler

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compileError(t *testing.T, c *Compiler, def *report.Definition) string {
	t.Helper()
	_, err := c.Compile(context.Background(), newRequest(def))
	require.Error(t, err)
	return err.Error()
}

func loadedMovesMiles(id string, loaded bool) report.ColumnSpec {
	col := measure(id, reportcatalog.AggSum, "distance", "moves")
	col.Filter = &report.FilterGroup{
		Op: report.BoolOpAnd,
		Filters: []report.FieldFilter{{
			Ref:      report.FieldRef{Path: []string{"moves"}, Field: "loaded"},
			Operator: dbtype.OpEqual,
			Value:    loaded,
		}},
	}
	return col
}

func TestMeasureFilterNarrowsAggregate(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	charge := measure("c_edi", reportcatalog.AggSum, "totalChargeAmount")
	charge.Filter = &report.FilterGroup{
		Op: report.BoolOpAnd,
		Filters: []report.FieldFilter{{
			Ref:      report.FieldRef{Field: "status"},
			Operator: dbtype.OpEqual,
			Value:    "Completed",
		}},
	}

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_all", reportcatalog.AggSum, "totalChargeAmount"),
			charge,
		},
	})

	assert.Contains(t, compiled.SQL, "SUM(t0.total_charge_amount) AS c1")
	assert.Contains(t,
		compiled.SQL,
		"SUM(t0.total_charge_amount) FILTER (WHERE t0.status = ?) AS c2",
	)
}

// A measure that aggregates children has to filter those children, not the
// parent rows — otherwise "loaded miles" would return every mile of any
// shipment that happened to have one loaded move.
func TestMeasureFilterOnToManyPathPushesIntoLateral(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_status", "status"),
			loadedMovesMiles("c_loaded", true),
			loadedMovesMiles("c_empty", false),
		},
	})

	assert.Contains(t, compiled.SQL, "LEFT JOIN LATERAL (SELECT SUM(w0.distance) FILTER (WHERE w0.loaded = ?)")
	assert.Contains(t, compiled.SQL, "SUM(w0.distance) FILTER (WHERE w0.loaded = ?) AS agg_1")
	// Both filters bind inside the single lateral, ahead of its tenant scope.
	assert.Equal(t, []any{true, false, testOrgID, testBuID, testOrgID, testBuID}, compiled.Args)
}

func TestMeasureFilterRejectsOffPathReference(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	col := measure("c_miles", reportcatalog.AggSum, "distance", "moves")
	col.Filter = &report.FilterGroup{
		Op: report.BoolOpAnd,
		Filters: []report.FieldFilter{{
			Ref:      report.FieldRef{Field: "status"},
			Operator: dbtype.OpEqual,
			Value:    "Completed",
		}},
	}

	message := compileError(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_status2", "proNumber"),
			col,
		},
	})

	assert.Contains(t, message, "move")
}

func TestMeasureFilterRejectedOnDimension(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	dimension := dim("c_status", "status")
	dimension.Filter = &report.FilterGroup{
		Op: report.BoolOpAnd,
		Filters: []report.FieldFilter{{
			Ref:      report.FieldRef{Field: "status"},
			Operator: dbtype.OpEqual,
			Value:    "Completed",
		}},
	}

	message := compileError(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dimension,
			measure("c_count", reportcatalog.AggCount, "id"),
		},
	})

	assert.Contains(t, message, "Only measure columns can carry their own filter")
}

// A calculation over two filtered measures is how a share-of-total lands in a
// single column, so the filters have to survive into the division.
func TestComputedColumnInheritsOperandFilters(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_status", "status"),
			loadedMovesMiles("c_empty", false),
			measure("c_total", reportcatalog.AggSum, "distance", "moves"),
			{
				ID:    "c_deadhead",
				Kind:  report.ColumnKindComputed,
				Label: "Deadhead %",
				Computed: &report.ComputedSpec{
					Op:      report.ComputedOpDivide,
					LeftID:  "c_empty",
					RightID: "c_total",
					Format:  reportcatalog.FormatPercent,
				},
			},
		},
	})

	assert.Contains(t, compiled.SQL, "NULLIF((SUM(l0.agg_1))::numeric, 0)")
	assert.Contains(t, compiled.SQL, "FILTER (WHERE w0.loaded = ?) AS agg_0")
}

func TestTotalsQueryDropsGroupingAndKeepsMeasures(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Totals:    true,
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_revenue", reportcatalog.AggSum, "totalChargeAmount"),
			measure("c_avg", reportcatalog.AggAvg, "totalChargeAmount"),
		},
	})

	require.True(t, compiled.HasTotals())
	assert.Contains(t, compiled.TotalsSQL, "SELECT NULL AS c0")
	assert.Contains(t, compiled.TotalsSQL, "SUM(t0.total_charge_amount) AS c1")
	// A grand average is the average of every row, not the average of the
	// per-group averages the table shows.
	assert.Contains(t, compiled.TotalsSQL, "AVG(t0.total_charge_amount) AS c2")
	assert.NotContains(t, compiled.TotalsSQL, "GROUP BY")
	assert.NotContains(t, compiled.TotalsSQL, "LIMIT")
}

func TestTotalsSkippedWithoutGrouping(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Totals:    true,
		Columns: []report.ColumnSpec{
			measure("c_revenue", reportcatalog.AggSum, "totalChargeAmount"),
		},
	})

	assert.False(t, compiled.HasTotals())
}

func TestTotalsRejectedWithMeasureFilters(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	message := compileError(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Totals:    true,
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_revenue", reportcatalog.AggSum, "totalChargeAmount"),
		},
		Having: &report.FilterGroup{
			Op: report.BoolOpAnd,
			Filters: []report.FieldFilter{{
				Ref:      report.FieldRef{Field: "totalChargeAmount"},
				Operator: dbtype.OpGreaterThan,
				Agg:      reportcatalog.AggSum,
				Value:    1000,
			}},
		},
	})

	assert.Contains(t, message, "cannot be combined with measure filters")
}

func pivotedDefinition(sortColumn string) *report.Definition {
	return &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_miles", reportcatalog.AggSum, "distance", "moves"),
		},
		Pivot: &report.PivotSpec{
			Ref:          report.FieldRef{Field: "status"},
			Values:       []string{"New", "Completed"},
			MeasureIDs:   []string{"c_miles"},
			IncludeOther: true,
		},
		Sort: []report.SortSpec{{
			ColumnID:  sortColumn,
			Direction: dbtype.SortDirectionDesc,
		}},
	}
}

func TestSortByPivotCell(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, pivotedDefinition("c_miles:Completed"))

	assert.Contains(t, compiled.SQL, "ORDER BY c2 DESC")
}

func TestSortByPivotOtherBucket(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, pivotedDefinition("c_miles:__other__"))

	assert.Contains(t, compiled.SQL, "ORDER BY c3 DESC")
}

func TestSortByUnknownPivotValueRejected(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	message := compileError(t, c, pivotedDefinition("c_miles:Billed"))

	assert.Contains(t, message, "which this report does not produce")
}

// Sorting by the measure itself has no column to point at once the measure is
// spread across buckets, so the builder is told to pick one.
func TestSortByPivotedMeasureRejected(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	message := compileError(t, c, pivotedDefinition("c_miles"))

	assert.Contains(t, message, "sort by one of them instead")
}

func TestFilterTransformMatchesNormalizedDimension(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
		Filters: &report.FilterGroup{
			Op: report.BoolOpAnd,
			Filters: []report.FieldFilter{{
				Ref:       report.FieldRef{Path: []string{"customer"}, Field: "name"},
				Operator:  dbtype.OpEqual,
				Value:     "ACME LOGISTICS",
				Transform: &report.TransformSpec{Op: report.TransformUpper},
			}},
		},
	})

	assert.Contains(t, compiled.SQL, "UPPER((t1.name)) = ?")
}

func chartDefinition(chart report.ChartSpec) *report.Definition {
	return &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_revenue", reportcatalog.AggSum, "totalChargeAmount"),
		},
		Charts: []report.ChartSpec{chart},
	}
}

func TestChartCompilesAlongsideColumns(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, chartDefinition(report.ChartSpec{
		ID:        "ch1",
		Type:      report.ChartBar,
		XColumnID: "c_customer",
		SeriesIDs: []string{"c_revenue"},
	}))

	assert.Len(t, compiled.Columns, 2)
}

func TestChartRejectsMeasureOnCategoryAxis(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	message := compileError(t, c, chartDefinition(report.ChartSpec{
		ID:        "ch1",
		Type:      report.ChartLine,
		XColumnID: "c_revenue",
		SeriesIDs: []string{"c_revenue"},
	}))

	assert.Contains(t, message, "Charts group by a dimension column")
}

func TestChartRejectsDimensionAsSeries(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	message := compileError(t, c, chartDefinition(report.ChartSpec{
		ID:        "ch1",
		Type:      report.ChartBar,
		XColumnID: "c_customer",
		SeriesIDs: []string{"c_customer"},
	}))

	assert.Contains(t, message, "cannot be plotted as a measure")
}

func TestKPIChartRejectsCategoryAxis(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	message := compileError(t, c, chartDefinition(report.ChartSpec{
		ID:        "ch1",
		Type:      report.ChartKPI,
		XColumnID: "c_customer",
		SeriesIDs: []string{"c_revenue"},
	}))

	assert.Contains(t, message, "has no category axis")
}

// Pivot cells are addressable outputs, so a chart may plot one bucket.
func TestChartPlotsPivotCell(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	def := pivotedDefinition("c_miles:Completed")
	def.Charts = []report.ChartSpec{{
		ID:        "ch1",
		Type:      report.ChartBar,
		XColumnID: "c_customer",
		SeriesIDs: []string{"c_miles:New", "c_miles:Completed"},
		Stacked:   true,
	}}

	_, err := c.Compile(context.Background(), newRequest(def))
	require.NoError(t, err)
}

func TestColumnIDRejectsPivotSeparator(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	message := compileError(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c:pro", "proNumber")},
	})

	assert.Contains(t, message, "cannot contain ':'")
}
