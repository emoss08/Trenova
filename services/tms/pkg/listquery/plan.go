package listquery

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/pkg/listcatalog"
)

type plan struct {
	cat      *listcatalog.Catalog
	ref      *keel.Ref
	computed map[string]keel.Expr
}

type field struct {
	expr  keel.DynExpr
	col   *schema.Column
	base  bool
	index int32
}

func newPlan[T any](q *keel.Query[*T], computed map[string]keel.Expr) *plan {
	ref := q.TableRef()
	return &plan{cat: listcatalog.For[T](ref.Table), ref: ref, computed: computed}
}

func (p *plan) baseColumn(index int32) keel.Col {
	return keel.Col{Ref: p.ref, Index: index}
}

func (p *plan) resolve(name string) (field, bool) {
	head, rest, nested := strings.Cut(name, ".")
	if !nested {
		index, ok := p.cat.Fields[name]
		if !ok {
			return p.resolveComputed(name)
		}
		return field{
			expr:  keel.DynOf(p.baseColumn(index)),
			col:   &p.cat.Table.Columns[index],
			base:  true,
			index: index,
		}, true
	}

	if strings.Contains(rest, ".") {
		return field{}, false
	}

	rel, ok := p.cat.Relation(head)
	if !ok || !rel.ToOne() {
		return field{}, false
	}

	index, ok := rel.Fields[rest]
	if !ok {
		return field{}, false
	}

	col := &rel.Target.Columns[index]
	expr := keel.ResolvePath(p, rel.Name, col.Name)
	if expr.Err() != nil {
		return field{}, false
	}

	return field{expr: expr, col: col, index: index}, true
}

func (p *plan) resolveComputed(name string) (field, bool) {
	index, ok := p.cat.Computed[name]
	if !ok {
		return field{}, false
	}
	expr, ok := p.computed[name]
	if !ok || expr == nil {
		return field{}, false
	}

	return field{expr: keel.DynOf(expr), col: &p.cat.Table.Columns[index], index: index}, true
}

func (p *plan) TableRef() *keel.Ref {
	return p.ref
}

func (f field) enum() bool {
	return len(f.col.Enum) > 0
}
