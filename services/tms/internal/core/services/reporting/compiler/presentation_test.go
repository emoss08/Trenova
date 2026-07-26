package compiler

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func intValue(v int) *int { return &v }

func floatValue(v float64) *float64 { return &v }

func TestTransformRoundsMeasureInSQL(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	avg := measure("c_avg", reportcatalog.AggAvg, "totalChargeAmount")
	avg.Transform = &report.TransformSpec{
		Op:        report.TransformRound,
		Precision: intValue(2),
	}

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_status", "status"),
			avg,
		},
	})

	assert.Contains(t, compiled.SQL, "ROUND((AVG(t0.total_charge_amount))::numeric, 2) AS c1")
	assert.Equal(t, reportcatalog.FieldDecimal, compiled.Columns[1].Type)
}

func TestTransformCeilingUsesScaledRounding(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sum := measure("c_miles", reportcatalog.AggSum, "distance")
	sum.Ref = report.FieldRef{Path: []string{"moves"}, Field: "distance"}
	sum.Transform = &report.TransformSpec{
		Op:        report.TransformCeil,
		Precision: intValue(1),
	}

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_status", "status"),
			sum,
		},
	})

	assert.Contains(t, compiled.SQL, "CEIL(")
	assert.Contains(t, compiled.SQL, "* 10) / 10")
}

func TestTransformOnDimensionDrivesGroupBy(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	dimension := dim("c_customer", "name", "customer")
	dimension.Transform = &report.TransformSpec{Op: report.TransformUpper}

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dimension,
			measure("c_count", reportcatalog.AggCount, "id"),
		},
	})

	assert.Contains(t, compiled.SQL, "UPPER((t1.name)) AS c0")
	assert.Contains(t, compiled.SQL, "GROUP BY UPPER((t1.name))")
}

func TestTransformScaleMultipliesValue(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sum := measure("c_weight", reportcatalog.AggSum, "weight")
	sum.Transform = &report.TransformSpec{
		Op:     report.TransformScale,
		Factor: floatValue(0.0005),
	}

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_status", "status"),
			sum,
		},
	})

	assert.Contains(t, compiled.SQL, "(SUM(t0.weight))::numeric * 0.0005")
}

func TestTransformRejectsTextOpOnNumericColumn(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sum := measure("c_total", reportcatalog.AggSum, "totalChargeAmount")
	sum.Transform = &report.TransformSpec{Op: report.TransformUpper}

	_, err := c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_status", "status"),
			sum,
		},
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "only applies to text columns")
}

func TestTransformRejectsFactorOnRounding(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sum := measure("c_total", reportcatalog.AggSum, "totalChargeAmount")
	sum.Transform = &report.TransformSpec{
		Op:     report.TransformRound,
		Factor: floatValue(2),
	}

	_, err := c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_status", "status"), sum},
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not take a multiplier")
}

func TestDisplaySpecOverridesCatalogDefaults(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	sum := measure("c_total", reportcatalog.AggSum, "totalChargeAmount")
	sum.Display = &reportfmt.Spec{
		Style:    reportfmt.StyleNumber,
		Decimals: intValue(0),
		Notation: reportfmt.NotationCompact,
	}

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_status", "status"), sum},
	})

	display := compiled.Columns[1].Display
	assert.Equal(t, reportfmt.StyleNumber, display.Style)
	assert.Equal(t, 0, display.Decimals)
	assert.Equal(t, reportfmt.NotationCompact, display.Notation)
}

func TestDisplayDefaultsFollowFormatHintAndBucket(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	bucketed := dim("c_month", "createdAt")
	bucketed.Bucket = report.DateBucketMonth

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			bucketed,
			measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
			measure("c_count", reportcatalog.AggCount, "id"),
		},
	})

	assert.Equal(t, reportfmt.StyleDate, compiled.Columns[0].Display.Style)
	assert.Equal(t, reportfmt.DateStyleMonthYear, compiled.Columns[0].Display.DateStyle)
	assert.Equal(t, reportfmt.StyleCurrency, compiled.Columns[1].Display.Style)
	assert.Equal(t, 2, compiled.Columns[1].Display.Decimals)
	assert.Equal(t, reportfmt.StyleNumber, compiled.Columns[2].Display.Style)
	assert.Equal(t, 0, compiled.Columns[2].Display.Decimals)
}

func TestDisplayRejectsStyleThatCannotApply(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	dimension := dim("c_customer", "name", "customer")
	dimension.Display = &reportfmt.Spec{Style: reportfmt.StyleCurrency}

	_, err := c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dimension},
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not valid for string columns")
}

// Two `code` fields reached through different edges must not both land in the
// export as "Code" — the header has to say which one it is.
func TestDefaultLabelsQualifyAmbiguousFields(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_service", "code", "serviceType"),
			dim("c_shipment_type", "code", "shipmentType"),
			dim("c_customer_code", "code", "customer"),
		},
	})

	labels := make([]string, 0, len(compiled.Columns))
	for i := range compiled.Columns {
		labels = append(labels, compiled.Columns[i].Label)
	}

	assert.Equal(t, []string{"Service Type Code", "Shipment Type Code", "Customer Code"}, labels)
}

func TestDefaultLabelsDescribeAggregates(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_status", "status"),
			measure("c_count", reportcatalog.AggCount, "id"),
			measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
			measure("c_avg", reportcatalog.AggAvg, "totalChargeAmount"),
			measure("c_last", reportcatalog.AggMax, "createdAt"),
		},
	})

	assert.Equal(t, "Status", compiled.Columns[0].Label)
	assert.Equal(t, "Shipments Count", compiled.Columns[1].Label)
	assert.Equal(t, "Total Charge", compiled.Columns[2].Label)
	assert.Equal(t, "Average Total Charge", compiled.Columns[3].Label)
	assert.Equal(t, "Latest Created At", compiled.Columns[4].Label)
}

func TestDuplicateLabelsAreDisambiguated(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	explicit := dim("c_a", "name", "customer")
	explicit.Label = "Account"
	duplicate := dim("c_b", "code", "customer")
	duplicate.Label = "Account"

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{explicit, duplicate},
	})

	assert.Equal(t, "Account", compiled.Columns[0].Label)
	assert.Equal(t, "Account (c_b)", compiled.Columns[1].Label)
}

func TestPivotLabelsUseAuthoredValueNames(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	miles := measure("c_miles", reportcatalog.AggSum, "distance")

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment_move",
		Columns: []report.ColumnSpec{
			dim("c_status", "status"),
			miles,
		},
		Pivot: &report.PivotSpec{
			Ref:        report.FieldRef{Field: "loaded"},
			Values:     []string{"true", "false"},
			Labels:     []string{"Loaded", "Empty"},
			MeasureIDs: []string{"c_miles"},
		},
	})

	require.Len(t, compiled.Columns, 3)
	assert.Equal(t, "Total Miles (Loaded)", compiled.Columns[1].Label)
	assert.Equal(t, "Total Miles (Empty)", compiled.Columns[2].Label)
}
