package compiler

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bandedDim(id, field string, band *reportfmt.Band, path ...string) report.ColumnSpec {
	col := dim(id, field, path...)
	col.Band = band
	return col
}

func bandedShipments(t *testing.T, c *Compiler, col report.ColumnSpec) string {
	t.Helper()
	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			col,
			measure("c_count", reportcatalog.AggCount, "id"),
		},
	})
	return compiled.SQL
}

// The boundaries are walked from the top down so a value lands in the highest
// range it clears, and the grouped value is the range's own lower bound.
func TestBandEdgesEmitDescendingCase(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sql := bandedShipments(t, c, bandedDim("c_weight", "weight",
		&reportfmt.Band{Edges: []float64{0, 10000, 40000}}))

	assert.Contains(t, sql,
		"(CASE WHEN (t0.weight)::numeric IS NULL THEN NULL"+
			" WHEN (t0.weight)::numeric >= 40000 THEN 40000"+
			" WHEN (t0.weight)::numeric >= 10000 THEN 10000"+
			" ELSE 0 END)::bigint AS c0")
}

// Grouping has to follow the band, not the raw column, or every distinct value
// would still produce its own row.
func TestBandGroupsOnTheBandedExpression(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sql := bandedShipments(t, c, bandedDim("c_weight", "weight",
		&reportfmt.Band{Width: 5000}))

	assert.Contains(t, sql, "(FLOOR((t0.weight)::numeric / 5000) * 5000)::bigint AS c0")
	assert.Contains(t, sql, "GROUP BY (FLOOR((t0.weight)::numeric / 5000) * 5000)::bigint")
}

// A whole-number column keeps its type so the row scans back as an integer.
func TestBandOnDecimalKeepsNumeric(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sql := bandedShipments(t, c, bandedDim("c_charge", "totalChargeAmount",
		&reportfmt.Band{Width: 2500.5}))

	assert.Contains(t, sql, "FLOOR((t0.total_charge_amount)::numeric / 2500.5) * 2500.5 AS c0")
	assert.NotContains(t, sql, "::bigint AS c0")
}

// Banding is what makes a high-cardinality numeric worth grouping by, so the
// catalog's "not a grouping key" flag must not block it.
func TestBandUnlocksNonGroupableNumeric(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	field, ok := shipmentField(t, "totalChargeAmount")
	require.True(t, ok)
	require.False(t, field.Groupable, "fixture assumes a charge amount is not groupable raw")

	assert.Contains(t,
		compileError(t, c, &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment",
			Columns: []report.ColumnSpec{
				dim("c_charge", "totalChargeAmount"),
				measure("c_count", reportcatalog.AggCount, "id"),
			},
		}),
		"cannot be used as a grouping dimension",
	)

	sql := bandedShipments(t, c, bandedDim("c_charge", "totalChargeAmount",
		&reportfmt.Band{Edges: []float64{0, 1000, 5000}}))
	assert.Contains(t, sql, "t0.total_charge_amount")
}

func TestBandRejections(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	tests := []struct {
		name    string
		column  report.ColumnSpec
		message string
	}{
		{
			name: "text field",
			column: bandedDim("c_status", "status",
				&reportfmt.Band{Edges: []float64{0, 1}}),
			message: "only valid on numeric fields",
		},
		{
			name: "fractional boundary on a whole-number column",
			column: bandedDim("c_weight", "weight",
				&reportfmt.Band{Edges: []float64{0, 1000.5}}),
			message: "must be whole numbers too",
		},
		{
			name: "descending boundaries",
			column: bandedDim("c_weight", "weight",
				&reportfmt.Band{Edges: []float64{5000, 1000}}),
			message: "must ascend",
		},
		{
			name: "a single boundary is not a range",
			column: bandedDim("c_weight", "weight",
				&reportfmt.Band{Edges: []float64{5000}}),
			message: "at least two boundaries",
		},
		{
			name: "width and boundaries together",
			column: bandedDim("c_weight", "weight",
				&reportfmt.Band{Width: 100, Edges: []float64{0, 1000}}),
			message: "not both",
		},
		{
			name: "negative width",
			column: bandedDim("c_weight", "weight",
				&reportfmt.Band{Width: -100}),
			message: "positive, finite number",
		},
		{
			name: "both a date bucket and a band",
			column: func() report.ColumnSpec {
				col := bandedDim("c_weight", "weight", &reportfmt.Band{Width: 100})
				col.Bucket = report.DateBucketMonth
				return col
			}(),
			message: "date bucket or a numeric range, not both",
		},
		{
			name: "a transform on top of a band",
			column: func() report.ColumnSpec {
				col := bandedDim("c_weight", "weight", &reportfmt.Band{Width: 100})
				col.Transform = &report.TransformSpec{Op: report.TransformRound}
				return col
			}(),
			message: "cannot also carry a transform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, compileError(t, c, &report.Definition{
				IRVersion: report.CurrentIRVersion,
				Entity:    "shipment",
				Columns: []report.ColumnSpec{
					tt.column,
					measure("c_count", reportcatalog.AggCount, "id"),
				},
			}), tt.message)
		})
	}
}

// A measure aggregates values; collapsing them into ranges first would mean
// summing lower bounds.
func TestMeasureCannotBeBanded(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	col := measure("c_weight", reportcatalog.AggSum, "weight")
	col.Band = &reportfmt.Band{Width: 100}

	assert.Contains(t, compileError(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_status", "status"), col},
	}), "band")
}

// The header has to say the rows are ranges; "Weight" alone reads as a value.
func TestBandedColumnLabelsAsRange(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			bandedDim("c_weight", "weight", &reportfmt.Band{Width: 5000}),
			measure("c_count", reportcatalog.AggCount, "id"),
		},
	})

	assert.Equal(t, "Weight (Range)", compiled.Columns[0].Label)
	require.NotNil(t, compiled.Columns[0].Display.Band)
	assert.InDelta(t, 5000, compiled.Columns[0].Display.Band.Width, 0)
}

func shipmentField(t *testing.T, key string) (*reportcatalog.Field, bool) {
	t.Helper()
	entity, ok := reportcatalog.Default.Entity("shipment")
	require.True(t, ok)
	return entity.Field(key)
}
