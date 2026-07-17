//go:build integration

package compiler

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func integrationCorpus() []*report.Definition {
	corpus := tripwireCorpus()

	corpus = append(corpus,
		&report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment",
			Columns: []report.ColumnSpec{
				{
					ID:     "c_month",
					Ref:    report.FieldRef{Field: "createdAt"},
					Kind:   report.ColumnKindDimension,
					Bucket: report.DateBucketMonth,
				},
				dim("c_customer", "name", "customer"),
				measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
				measure("c_avg", reportcatalog.AggAvg, "totalChargeAmount"),
			},
			Filters: &report.FilterGroup{
				Op: report.BoolOpAnd,
				Filters: []report.FieldFilter{
					{
						Ref:      report.FieldRef{Field: "createdAt"},
						Operator: dbtype.OpLastNDays,
						Value:    float64(90),
					},
					{
						Ref:      report.FieldRef{Field: "status"},
						Operator: dbtype.OpIn,
						Value:    []any{"New", "InTransit"},
					},
				},
			},
			Having: &report.FilterGroup{
				Op: report.BoolOpAnd,
				Filters: []report.FieldFilter{
					{
						Ref:      report.FieldRef{Field: "totalChargeAmount"},
						Operator: dbtype.OpGreaterThan,
						Value:    float64(0),
						Agg:      reportcatalog.AggSum,
					},
				},
			},
			Sort: []report.SortSpec{
				{ColumnID: "c_total", Direction: dbtype.SortDirectionDesc},
			},
			Limit: 1000,
		},
		&report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment",
			Columns: []report.ColumnSpec{
				dim("c_customer", "name", "customer"),
				measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
				measure("c_count", reportcatalog.AggCount, "id"),
			},
			Pivot: &report.PivotSpec{
				Ref:          report.FieldRef{Field: "status"},
				Values:       []string{"New", "InTransit", "Completed"},
				MeasureIDs:   []string{"c_total"},
				IncludeOther: true,
			},
		},
		&report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "order",
			Columns: []report.ColumnSpec{
				dim("c_num", "orderNumber"),
				measure("c_ship_count", reportcatalog.AggCount, "id", "shipments"),
				measure("c_ship_total", reportcatalog.AggSum, "totalChargeAmount", "shipments"),
			},
			Filters: &report.FilterGroup{
				Op: report.BoolOpAnd,
				Filters: []report.FieldFilter{
					{
						Ref:      report.FieldRef{Path: []string{"shipments"}, Field: "status"},
						Operator: dbtype.OpNotIn,
						Value:    []any{"Canceled"},
					},
				},
			},
		},
	)

	return corpus
}

func TestCompiledQueriesExecuteAgainstRealSchema(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	c := newTestCompiler(allowAllEngine())

	for i, def := range integrationCorpus() {
		compiled, err := c.Compile(context.Background(), newRequest(def))
		require.NoError(t, err, "corpus %d failed to compile", i)

		err = PreflightCost(ctx, db, compiled, CostLimits{
			MaxEstimatedCost: 100_000_000,
			MaxEstimatedRows: 1_000_000_000,
		})
		require.NoError(t, err, "corpus %d failed EXPLAIN:\n%s", i, compiled.SQL)

		rows, err := db.QueryContext(ctx, compiled.SQL, compiled.Args...)
		require.NoError(t, err, "corpus %d failed to execute:\n%s", i, compiled.SQL)

		cols, err := rows.Columns()
		require.NoError(t, err)
		require.Len(t, cols, len(compiled.Columns),
			"corpus %d column count mismatch", i)
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
	}
}

func TestPreflightCostRejectsAbsurdLimits(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	c := newTestCompiler(allowAllEngine())
	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
	})

	err := PreflightCost(ctx, db, compiled, CostLimits{MaxEstimatedCost: 0.000001})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too expensive")
}

var _ bun.IDB = (*bun.DB)(nil)
