package compiler

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func computedCol(
	id, label string,
	op report.ComputedOp,
	leftID, rightID string,
) report.ColumnSpec {
	return report.ColumnSpec{
		ID:    id,
		Kind:  report.ColumnKindComputed,
		Label: label,
		Computed: &report.ComputedSpec{
			Op:      op,
			LeftID:  leftID,
			RightID: rightID,
		},
	}
}

func TestCompileComputedDivide(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	spec := computedCol(
		"c_per_pound", "Revenue per Pound",
		report.ComputedOpDivide, "c_total", "c_weight",
	)
	spec.Computed.Format = reportcatalog.FormatMoney

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
			measure("c_weight", reportcatalog.AggSum, "weight"),
			spec,
		},
		Sort: []report.SortSpec{
			{ColumnID: "c_per_pound", Direction: "desc"},
		},
	})

	want := "SELECT t1.name AS c0, SUM(t0.total_charge_amount) AS c1, SUM(t0.weight) AS c2, " +
		"(SUM(t0.total_charge_amount))::numeric / NULLIF((SUM(t0.weight))::numeric, 0) AS c3 " +
		"FROM shipments AS t0 " +
		"LEFT JOIN customers AS t1 ON t1.id = t0.customer_id" +
		" AND t1.organization_id = ? AND t1.business_unit_id = ? " +
		"WHERE t0.organization_id = ? AND t0.business_unit_id = ? " +
		"GROUP BY t1.name " +
		"ORDER BY c3 DESC " +
		"LIMIT 100000"
	assert.Equal(t, want, compiled.SQL)

	require.Len(t, compiled.Columns, 4)
	assert.Equal(t, "Revenue per Pound", compiled.Columns[3].Label)
	assert.Equal(t, reportcatalog.FieldDecimal, compiled.Columns[3].Type)
	assert.Equal(t, reportcatalog.FormatMoney, compiled.Columns[3].Format)
}

func TestCompileComputedIntArithmetic(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_status", "status"),
			measure("c_weight", reportcatalog.AggSum, "weight"),
			measure("c_pieces", reportcatalog.AggSum, "pieces"),
			computedCol(
				"c_handling", "Handling Units",
				report.ComputedOpAdd, "c_weight", "c_pieces",
			),
		},
	})

	assert.Contains(t, compiled.SQL, "(SUM(t0.weight)) + (SUM(t0.pieces)) AS c3")
	require.Len(t, compiled.Columns, 4)
	assert.Equal(t, reportcatalog.FieldInt, compiled.Columns[3].Type)
}

func TestCompileComputedAcrossLateral(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
			measure("c_miles", reportcatalog.AggSum, "distance", "moves"),
			computedCol(
				"c_rpm", "Revenue per Mile",
				report.ComputedOpDivide, "c_total", "c_miles",
			),
		},
	})

	assert.Contains(t, compiled.SQL,
		"(SUM(t0.total_charge_amount))::numeric / NULLIF((SUM(l0.agg_0))::numeric, 0) AS c3")
	assert.Contains(t, compiled.SQL, "LEFT JOIN LATERAL (SELECT SUM(w0.distance) AS agg_0")
}

func TestCompileComputedPivot(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
			measure("c_weight", reportcatalog.AggSum, "weight"),
			computedCol(
				"c_per_pound", "Revenue per Pound",
				report.ComputedOpDivide, "c_total", "c_weight",
			),
		},
		Pivot: &report.PivotSpec{
			Ref:        report.FieldRef{Field: "status"},
			Values:     []string{"New", "InTransit"},
			MeasureIDs: []string{"c_per_pound"},
		},
	})

	assert.Contains(t, compiled.SQL,
		"(SUM(t0.total_charge_amount) FILTER (WHERE t0.status = ?))::numeric / "+
			"NULLIF((SUM(t0.weight) FILTER (WHERE t0.status = ?))::numeric, 0)")

	statusBinds := 0
	for _, arg := range compiled.Args {
		if arg == "New" || arg == "InTransit" {
			statusBinds++
		}
	}
	assert.Equal(t, 4, statusBinds, "each pivot cell binds its bucket value once per operand")

	pivotIDs := make([]string, 0, len(compiled.Columns))
	for _, col := range compiled.Columns {
		pivotIDs = append(pivotIDs, col.ID)
	}
	assert.Contains(t, pivotIDs, "c_per_pound:New")
	assert.Contains(t, pivotIDs, "c_per_pound:InTransit")
}

func TestComputedValidation(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	base := func(spec report.ColumnSpec) *report.Definition {
		return &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment",
			Columns: []report.ColumnSpec{
				dim("c_status", "status"),
				measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
				spec,
			},
		}
	}

	tests := []struct {
		name    string
		def     *report.Definition
		wantErr string
	}{
		{
			name: "operand is a dimension",
			def: base(computedCol(
				"c_bad", "Bad", report.ComputedOpDivide, "c_total", "c_status",
			)),
			wantErr: "must be a measure column",
		},
		{
			name: "operand unknown",
			def: base(computedCol(
				"c_bad", "Bad", report.ComputedOpDivide, "c_total", "c_missing",
			)),
			wantErr: "does not reference a valid column",
		},
		{
			name: "self reference",
			def: base(computedCol(
				"c_bad", "Bad", report.ComputedOpDivide, "c_bad", "c_total",
			)),
			wantErr: "cannot reference itself",
		},
		{
			name: "missing label",
			def: base(computedCol(
				"c_bad", "", report.ComputedOpDivide, "c_total", "c_total",
			)),
			wantErr: "require a label",
		},
		{
			name: "invalid operator",
			def: base(computedCol(
				"c_bad", "Bad", report.ComputedOp("modulo"), "c_total", "c_total",
			)),
			wantErr: "must be add, subtract, multiply, or divide",
		},
		{
			name: "invalid format hint",
			def: func() *report.Definition {
				spec := computedCol(
					"c_bad", "Bad", report.ComputedOpDivide, "c_total", "c_total",
				)
				spec.Computed.Format = reportcatalog.FormatHint("currency")
				return base(spec)
			}(),
			wantErr: "Unknown format hint",
		},
		{
			name: "computed expression on a measure column",
			def: func() *report.Definition {
				spec := measure("c_bad", reportcatalog.AggSum, "weight")
				spec.Computed = &report.ComputedSpec{
					Op: report.ComputedOpAdd, LeftID: "c_total", RightID: "c_total",
				}
				return base(spec)
			}(),
			wantErr: "Only computed columns may carry a computed expression",
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

func TestRefParamValidation(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	def := func(param report.ParameterDef) *report.Definition {
		return &report.Definition{
			IRVersion:  report.CurrentIRVersion,
			Entity:     "shipment",
			Columns:    []report.ColumnSpec{dim("c_pro", "proNumber")},
			Parameters: []report.ParameterDef{param},
		}
	}

	tests := []struct {
		name    string
		param   report.ParameterDef
		wantErr string
	}{
		{
			name:    "ref param without refEntity",
			param:   report.ParameterDef{Name: "cust", Type: reportcatalog.FieldRef},
			wantErr: "must declare a reference entity",
		},
		{
			name: "ref param with unknown entity",
			param: report.ParameterDef{
				Name: "cust", Type: reportcatalog.FieldRef, RefEntity: "starship",
			},
			wantErr: "unknown entity",
		},
		{
			name: "non-ref param with refEntity",
			param: report.ParameterDef{
				Name: "days", Type: reportcatalog.FieldInt, RefEntity: "customer",
			},
			wantErr: "is not a ref parameter",
		},
		{
			name: "ref param with allow-list",
			param: report.ParameterDef{
				Name: "cust", Type: reportcatalog.FieldRef, RefEntity: "customer",
				AllowedValues: []string{"cus_1"},
			},
			wantErr: "cannot carry an allow-list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Compile(context.Background(), newRequest(def(tt.param)))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestCompileRefParamBinding(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	def := &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
		Filters: &report.FilterGroup{
			Op: report.BoolOpAnd,
			Filters: []report.FieldFilter{
				{
					Ref:      report.FieldRef{Path: []string{"customer"}, Field: "id"},
					Operator: "eq",
					Param:    "customerId",
				},
			},
		},
		Parameters: []report.ParameterDef{
			{
				Name:      "customerId",
				Label:     "Customer",
				Type:      reportcatalog.FieldRef,
				RefEntity: "customer",
				Required:  true,
			},
		},
	}

	req := newRequest(def)
	req.Params = map[string]any{"customerId": "cus_01hqzz"}
	compiled, err := c.Compile(context.Background(), req)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "t1.id = ?")
	assert.Contains(t, compiled.Args, "cus_01hqzz")
}
