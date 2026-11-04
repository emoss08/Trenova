package detentionrollup

import (
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/analytics/rollup"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
)

// Table is a rollup's table as a read sees it: one reference, and its columns by the names the
// declarations give them.
type Table struct {
	Ref  *keel.Ref
	cols map[string]int32
}

func tableOf(r *rollup.Rollup) *Table {
	t := r.Table()
	cols := make(map[string]int32, len(t.Columns))
	for i := range t.Columns {
		cols[t.Columns[i].Name] = int32(i)
	}

	return &Table{Ref: &keel.Ref{Table: t}, cols: cols}
}

var (
	DailyTable           = tableOf(Daily)
	FacilityDailyTable   = tableOf(FacilityDaily)
	FacilityMonthlyTable = tableOf(FacilityMonthly)
	CustomerDailyTable   = tableOf(CustomerDaily)
	CustomerMonthlyTable = tableOf(CustomerMonthly)
	PolicyDailyTable     = tableOf(PolicyDaily)
	PolicyMonthlyTable   = tableOf(PolicyMonthly)
	ArrivalDailyTable    = tableOf(ArrivalDaily)
)

// Source is what a read sums: a rollup table, or a window's whole months and edge days read
// together (View).
type Source interface {
	Col(name string) keel.Col
	Count(name string) keel.Int64Expr
	Amount(name string) keel.NumericExpr
}

// Has reports whether the table stores a column of that name.
func (t *Table) Has(name string) bool {
	_, ok := t.cols[name]
	return ok
}

// Col is the named column. A name the declaration never gave is a programming error, which the
// package's tests catch for every name a read uses.
func (t *Table) Col(name string) keel.Col {
	idx, ok := t.cols[name]
	if !ok {
		panic(fmt.Sprintf("detentionrollup: %s has no column %s", t.Ref.Table.Name, name))
	}

	return keel.Col{Ref: t.Ref, Index: idx}
}

// Count is the named count column summed over the rows read: zero when none are.
func (t *Table) Count(name string) keel.Int64Expr { return countOf(t.Col(name)) }

// Amount is the named sum column summed over the rows read: zero when none are.
func (t *Table) Amount(name string) keel.NumericExpr { return amountOf(t.Col(name)) }

func countOf(c keel.Col) keel.Int64Expr {
	return keel.CastInt64(keel.NumericOf(keel.Sum(c)).Coalesce(0))
}

func amountOf(c keel.Col) keel.NumericExpr {
	return keel.NumericOf(keel.Sum(c)).Coalesce(0)
}

// Bucket is the day each row adds up.
func (t *Table) Bucket() keel.TimeExpr {
	return keel.TimeOf(t.Col(Bucket))
}

// As is the table under another name, for a subquery over the table its outer query reads.
func (t *Table) As(alias string) *Table {
	return &Table{
		Ref:  keel.Alias(t.Ref, alias, func(ref *keel.Ref) *keel.Ref { return ref }),
		cols: t.cols,
	}
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
	monthly, daily *Table,
	w detentionports.IntelligenceWindow,
	columns []string,
	where func(t *Table) []keel.Pred,
) *View {
	part := func(t *Table, from, to time.Time, last bool) keel.Query[*keel.Row] {
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
func (v *View) Query() keel.Query[*keel.Row] { return v.cte.Query().With(v.cte) }

// CTE is the view's definition, for a read that defines further CTEs over it.
func (v *View) CTE() *keel.CTETable { return v.cte }

// Rows starts a read of the view inside another CTE of the same statement, which lists the view
// in its own WITH.
func (v *View) Rows() keel.Query[*keel.Row] { return v.cte.Query() }

// Col is the named column of the view; a name the view was not built with is a programming error.
func (v *View) Col(name string) keel.Col {
	if !v.cols[name] {
		panic(fmt.Sprintf("detentionrollup: the view has no column %s", name))
	}

	return v.cte.Col(name).Col
}

func (v *View) Count(name string) keel.Int64Expr { return countOf(v.Col(name)) }

func (v *View) Amount(name string) keel.NumericExpr { return amountOf(v.Col(name)) }
