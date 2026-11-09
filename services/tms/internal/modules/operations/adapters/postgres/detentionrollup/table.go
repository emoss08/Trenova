package detentionrollup

import (
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/rollupread"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
)

var (
	DailyTable           = rollupread.Of(Daily)
	FacilityDailyTable   = rollupread.Of(FacilityDaily)
	FacilityMonthlyTable = rollupread.Of(FacilityMonthly)
	CustomerDailyTable   = rollupread.Of(CustomerDaily)
	CustomerMonthlyTable = rollupread.Of(CustomerMonthly)
	PolicyDailyTable     = rollupread.Of(PolicyDaily)
	PolicyMonthlyTable   = rollupread.Of(PolicyMonthly)
	ArrivalDailyTable    = rollupread.Of(ArrivalDaily)
)

// Source is what a read sums: a rollup table, or a window's whole months and edge days read
// together (View).
type Source interface {
	Col(name string) keel.Col
	Count(name string) keel.Int64Expr
	Amount(name string) keel.NumericExpr
}

// View is a window read from a monthly and a daily rollup together: the whole months from the
// monthly one and the days before and after them from the daily one, three index ranges in one
// UNION ALL, as one table of the columns asked for.
type View struct {
	cte  *keel.CTETable
	cols map[string]bool
}

// Union reads columns of the window from monthly and daily, each narrowed by where.
func Union(
	name string,
	monthly, daily *rollupread.Table,
	w detentionports.IntelligenceWindow,
	columns []string,
	where func(t *rollupread.Table) []keel.Pred,
) *View {
	part := func(t *rollupread.Table, from, to time.Time, last bool) keel.Query[*keel.Row] {
		picks := make([]keel.Expr, len(columns))
		for i, c := range columns {
			picks[i] = t.Col(c)
		}
		upper := t.Bucket().Lt(to)
		if last {
			upper = t.Bucket().Lte(to)
		}
		preds := append([]keel.Pred{t.Bucket().Gte(from), upper}, where(t)...)
		return keel.RowsOf(t.Ref).Pick(picks...).Where(preds...)
	}
	at := func(s int64) time.Time { return time.Unix(s, 0) }
	cols := make(map[string]bool, len(columns))
	for _, c := range columns {
		cols[c] = true
	}

	return &View{
		cte: keel.CTE(name, columns, keel.UnionAll(
			part(monthly, at(w.MonthsFrom), at(w.MonthsEnd), false),
			part(daily, at(w.From), at(w.MonthsFrom), false),
			part(daily, at(w.MonthsEnd), at(w.To), true),
		)),
		cols: cols,
	}
}

// Query starts a read of the view.
func (v *View) Query() keel.Query[*keel.Row] { return keel.RowsOf(v.cte.TableRef()).With(v.cte) }

// CTE is the view's definition, for a read that defines further CTEs over it.
func (v *View) CTE() *keel.CTETable { return v.cte }

// Rows starts a read of the view inside another CTE of the same statement, which lists the view
// in its own WITH.
func (v *View) Rows() keel.Query[*keel.Row] { return keel.RowsOf(v.cte.TableRef()) }

// Col is the named column of the view; a name the view was not built with is a programming error.
func (v *View) Col(name string) keel.Col {
	if !v.cols[name] {
		panic(fmt.Sprintf("detentionrollup: the view has no column %s", name))
	}

	return v.cte.Col(name).Col
}

func (v *View) Count(name string) keel.Int64Expr { return rollupread.CountOf(v.Col(name)) }

func (v *View) Amount(name string) keel.NumericExpr { return rollupread.AmountOf(v.Col(name)) }
