package listquery

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/schema"
)

type plan struct {
	cat   *catalog
	ref   *keel.Ref
	paths *keel.DynQuery
}

type field struct {
	expr  keel.DynExpr
	col   *schema.Column
	base  bool
	index int32
}

func newPlan[T any](q *keel.Query[T]) *plan {
	ref := q.TableRef()
	return &plan{cat: catalogFor[T](ref.Table), ref: ref}
}

func (p *plan) baseColumn(index int32) keel.Col {
	return keel.Col{Ref: p.ref, Index: index}
}

func (p *plan) resolve(name string) (field, bool) {
	head, rest, nested := strings.Cut(name, ".")
	if !nested {
		index, ok := p.cat.fields[name]
		if !ok {
			return field{}, false
		}
		return field{
			expr:  keel.DynOf(p.baseColumn(index)),
			col:   &p.cat.table.Columns[index],
			base:  true,
			index: index,
		}, true
	}

	if strings.Contains(rest, ".") {
		return field{}, false
	}

	rel, ok := p.cat.relation(head)
	if !ok || !rel.toOne() {
		return field{}, false
	}

	index, ok := rel.fields[rest]
	if !ok {
		return field{}, false
	}

	col := &rel.target.Columns[index]
	expr := p.relationPaths().Path(rel.name, col.Name)
	if expr.Err() != nil {
		return field{}, false
	}

	return field{expr: expr, col: col, index: index}, true
}

func (p *plan) relationPaths() *keel.DynQuery {
	if p.paths == nil {
		p.paths = keel.DynFrom(p.cat.table)
	}

	return p.paths
}

func (f field) enum() bool {
	return len(f.col.Enum) > 0
}
