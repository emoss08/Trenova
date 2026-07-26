package compiler

import (
	"context"
	"math"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goalCol builds a calculation whose right-hand side is a constant target
// rather than a second measure — the shape a variance-vs-goal column takes.
func goalCol(id, label string, op report.ComputedOp, leftID string, goal float64) report.ColumnSpec {
	return report.ColumnSpec{
		ID:    id,
		Kind:  report.ColumnKindComputed,
		Label: label,
		Computed: &report.ComputedSpec{
			Op:         op,
			LeftID:     leftID,
			RightValue: &goal,
		},
	}
}

func goalDef(columns ...report.ColumnSpec) *report.Definition {
	return &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: append([]report.ColumnSpec{
			dim("c_status", "status"),
			measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
		}, columns...),
	}
}

// The point of the feature: revenue measured against a target nobody stores.
func TestCompileVarianceAgainstConstantGoal(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	spec := goalCol("c_var", "Variance to Goal", report.ComputedOpSubtract, "c_total", 250000)
	spec.Computed.Format = reportcatalog.FormatMoney

	compiled := mustCompile(t, c, goalDef(spec))

	assert.Contains(t, compiled.SQL, "(SUM(t0.total_charge_amount)) - (250000) AS c2")
	require.Len(t, compiled.Columns, 3)
	assert.Equal(t, "Variance to Goal", compiled.Columns[2].Label)
	assert.Equal(t, reportcatalog.FormatMoney, compiled.Columns[2].Format)
}

// Attainment is the other half: actual over target, rendered as a percentage.
func TestCompileAttainmentAgainstConstantGoal(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	spec := goalCol("c_att", "Goal Attainment", report.ComputedOpDivide, "c_total", 250000)
	spec.Computed.Format = reportcatalog.FormatPercent

	compiled := mustCompile(t, c, goalDef(spec))

	assert.Contains(t, compiled.SQL,
		"(SUM(t0.total_charge_amount))::numeric / NULLIF((250000)::numeric, 0) AS c2")
	assert.Equal(t, reportcatalog.FieldDecimal, compiled.Columns[2].Type)
}

// A target is not always the right operand: "how much is left to hit goal"
// reads goal minus actual.
func TestCompileGoalOnTheLeftSide(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	goal := 500.0
	spec := report.ColumnSpec{
		ID:    "c_remaining",
		Kind:  report.ColumnKindComputed,
		Label: "Loads Remaining",
		Computed: &report.ComputedSpec{
			Op:        report.ComputedOpSubtract,
			LeftValue: &goal,
			RightID:   "c_loads",
		},
	}

	compiled := mustCompile(t, c, goalDef(
		measure("c_loads", reportcatalog.AggCount, "id"),
		spec,
	))

	assert.Contains(t, compiled.SQL, "(500) - (COUNT(t0.id)) AS c3")
}

// The literal is inlined rather than bound, because the whole expression is
// re-emitted per pivot cell; a placeholder would desync the argument list.
func TestGoalLiteralIsInlinedNotBound(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, goalDef(
		goalCol("c_var", "Variance", report.ComputedOpSubtract, "c_total", 250000),
	))

	for _, arg := range compiled.Args {
		assert.NotEqual(t, 250000.0, arg, "the target never reaches the argument list")
	}
}

// A pivoted goal column repeats the constant per bucket while the measure side
// still binds its own bucket predicate.
func TestGoalSurvivesPivot(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
			goalCol("c_att", "Attainment", report.ComputedOpDivide, "c_total", 1000),
		},
		Pivot: &report.PivotSpec{
			Ref:        report.FieldRef{Field: "status"},
			Values:     []string{"New", "InTransit"},
			MeasureIDs: []string{"c_att"},
		},
	})

	assert.Contains(t, compiled.SQL,
		"(SUM(t0.total_charge_amount) FILTER (WHERE t0.status = ?))::numeric / "+
			"NULLIF((1000)::numeric, 0)")

	statusBinds := 0
	for _, arg := range compiled.Args {
		if arg == "New" || arg == "InTransit" {
			statusBinds++
		}
	}
	assert.Equal(t, 2, statusBinds, "only the measure side binds a bucket value")
}

// A whole-numbered target keeps an integer calculation in bigint, so the
// decoded type has to stay int or the executor reads the wrong column shape.
func TestWholeNumberedGoalKeepsIntegerResult(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, goalDef(
		measure("c_pieces", reportcatalog.AggSum, "pieces"),
		goalCol("c_var", "Variance", report.ComputedOpSubtract, "c_pieces", 100),
	))

	assert.Contains(t, compiled.SQL, "(SUM(t0.pieces)) - (100) AS c3")
	assert.Equal(t, reportcatalog.FieldInt, compiled.Columns[3].Type)
}

// A fractional target widens the same calculation to numeric, and the emitted
// literal has to agree with that or Postgres and the decoder disagree.
func TestFractionalGoalWidensToDecimal(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, goalDef(
		measure("c_pieces", reportcatalog.AggSum, "pieces"),
		goalCol("c_var", "Variance", report.ComputedOpSubtract, "c_pieces", 100.5),
	))

	assert.Contains(t, compiled.SQL, "(SUM(t0.pieces)) - (100.5) AS c3")
	assert.Equal(t, reportcatalog.FieldDecimal, compiled.Columns[3].Type)
}

// A constant carries no field, so it must not drag the calculation's
// sensitivity up or leave it below that of the measure it reads.
func TestGoalColumnInheritsOnlyTheMeasureSensitivity(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, goalDef(
		goalCol("c_var", "Variance", report.ComputedOpSubtract, "c_total", 250000),
	))

	require.Len(t, compiled.Columns, 3)
	assert.Equal(t, compiled.Columns[1].Sensitivity, compiled.Columns[2].Sensitivity)
}

func TestGoalValidation(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	withComputed := func(comp *report.ComputedSpec) *report.Definition {
		return goalDef(report.ColumnSpec{
			ID:       "c_bad",
			Kind:     report.ColumnKindComputed,
			Label:    "Bad",
			Computed: comp,
		})
	}
	f := func(v float64) *float64 { return &v }

	tests := []struct {
		name    string
		def     *report.Definition
		wantErr string
	}{
		{
			name: "operand is both a column and a target",
			def: withComputed(&report.ComputedSpec{
				Op: report.ComputedOpSubtract, LeftID: "c_total",
				RightID: "c_total", RightValue: f(10),
			}),
			wantErr: "either a measure or a target value, not both",
		},
		{
			name: "operand is neither",
			def: withComputed(&report.ComputedSpec{
				Op: report.ComputedOpSubtract, LeftID: "c_total",
			}),
			wantErr: "needs a measure or a target value",
		},
		{
			name: "both sides constant",
			def: withComputed(&report.ComputedSpec{
				Op: report.ComputedOpSubtract, LeftValue: f(10), RightValue: f(5),
			}),
			wantErr: "needs at least one measure",
		},
		{
			name: "divide by a zero target",
			def: withComputed(&report.ComputedSpec{
				Op: report.ComputedOpDivide, LeftID: "c_total", RightValue: f(0),
			}),
			wantErr: "divide by a target value of zero",
		},
		{
			name: "target is not a number",
			def: withComputed(&report.ComputedSpec{
				Op: report.ComputedOpSubtract, LeftID: "c_total",
				RightValue: f(math.NaN()),
			}),
			wantErr: "must be a finite number",
		},
		{
			name: "target is infinite",
			def: withComputed(&report.ComputedSpec{
				Op: report.ComputedOpSubtract, LeftID: "c_total",
				RightValue: f(math.Inf(1)),
			}),
			wantErr: "must be a finite number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Compile(context.Background(), newRequest(tt.def))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
