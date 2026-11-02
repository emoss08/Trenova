package listquery

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/pagination"
)

type Window struct {
	Fields []string
	Cap    int
}

func (w *Window) appliesTo(filter *pagination.QueryOptions) bool {
	if w == nil || w.Cap <= 0 || filter == nil {
		return false
	}
	for _, f := range filter.FieldFilters {
		if slices.Contains(w.Fields, f.Field) {
			return true
		}
	}
	for _, group := range filter.FilterGroups {
		for _, f := range group.Filters {
			if slices.Contains(w.Fields, f.Field) {
				return true
			}
		}
	}

	return false
}

func bindWindow[T any](
	ctx context.Context,
	db *keel.DB,
	p *plan,
	q keel.Query[*T],
	req *Request[T],
) (keel.Query[*T], error) {
	if !req.Window.appliesTo(req.Filter) {
		return q, nil
	}

	return capMatches(ctx, db, p, q, req.Window.Cap)
}

func capMatches[T any](
	ctx context.Context,
	db *keel.DB,
	p *plan,
	q keel.Query[*T],
	limit int,
) (keel.Query[*T], error) {
	id, ok := p.idColumn()
	if !ok {
		return q, nil
	}

	matched, err := q.Clone().CountUpTo(ctx, db, int64(limit)+1)
	if err != nil {
		return nil, err
	}
	if matched > int64(limit) {
		return q.CustomPlan(), nil
	}

	return keel.QueryOf[*T](q.TableRef()).Where(id.InQuery(q.Pick(id).Limit(limit + 1))), nil
}

func (p *plan) idColumn() (keel.Col, bool) {
	index, ok := p.cat.Fields["id"]
	if !ok {
		return keel.Col{}, false
	}

	return p.baseColumn(index), true
}
