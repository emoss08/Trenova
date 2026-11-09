package rollupread

import (
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/analytics/rollup"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
)

const Bucket = "bucket"

type Table struct {
	Ref  *keel.Ref
	cols map[string]int32
}

func Of(r *rollup.Rollup) *Table {
	t := r.Table()
	cols := make(map[string]int32, len(t.Columns))
	for i := range t.Columns {
		cols[t.Columns[i].Name] = int32(i)
	}

	return &Table{Ref: &keel.Ref{Table: t}, cols: cols}
}

func (t *Table) Has(name string) bool {
	_, ok := t.cols[name]
	return ok
}

func (t *Table) Col(name string) keel.Col {
	idx, ok := t.cols[name]
	if !ok {
		panic(fmt.Sprintf("rollupread: %s has no column %s", t.Ref.Table.Name, name))
	}

	return keel.Col{Ref: t.Ref, Index: idx}
}

func (t *Table) Count(name string) keel.Int64Expr { return CountOf(t.Col(name)) }

func (t *Table) Amount(name string) keel.NumericExpr { return AmountOf(t.Col(name)) }

func (t *Table) Bucket() keel.TimeExpr { return keel.TimeOf(t.Col(Bucket)) }

func (t *Table) As(alias string) *Table {
	return &Table{
		Ref:  keel.Alias(t.Ref, alias, func(ref *keel.Ref) *keel.Ref { return ref }),
		cols: t.cols,
	}
}

func CountOf(c keel.Col) keel.Int64Expr {
	return keel.CastInt64(keel.NumericOf(keel.Sum(c)).Coalesce(0))
}

func AmountOf(c keel.Col) keel.NumericExpr {
	return keel.NumericOf(keel.Sum(c)).Coalesce(0)
}

func OrganizationZone(organizationID keel.Expr) keel.StringExpr {
	orgs, zone, _ := keel.ColumnOf(tenantdb.Organizations.Timezone)
	_, id, _ := keel.ColumnOf(tenantdb.Organizations.ID)
	return keel.StringOf(keel.Raw(
		"coalesce((SELECT z."+orgs.Columns[zone].Quoted+" FROM "+orgs.Quoted+" z WHERE z."+
			orgs.Columns[id].Quoted+" = ?), 'UTC')",
		organizationID,
	))
}

func LocalHour(epoch keel.NumberValue, zone keel.StringValue) keel.Int64Expr {
	return keel.CastInt64(keel.DatePart("hour", keel.ToTimestamp(epoch).AtTimeZone(zone)))
}
