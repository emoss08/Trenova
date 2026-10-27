package listquery

import (
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
)

const (
	MaxFilters          = 50
	MaxSortFields       = 10
	MaxAggregateFilters = 10

	secondsPerDay = 86400
)

func (p *plan) where(filter *pagination.QueryOptions) []keel.Pred {
	if filter == nil {
		return nil
	}

	preds := make([]keel.Pred, 0, len(filter.FieldFilters)+len(filter.FilterGroups)+1)
	for _, f := range capped(filter.FieldFilters, MaxFilters) {
		if pred, ok := p.predicate(f); ok && pred != nil {
			preds = append(preds, pred)
		}
	}

	for _, group := range capped(filter.FilterGroups, MaxFilters) {
		if pred := p.group(group); pred != nil {
			preds = append(preds, pred)
		}
	}

	for _, af := range capped(filter.AggregateFilters, MaxAggregateFilters) {
		if pred, ok := p.aggregate(af); ok {
			preds = append(preds, pred)
		}
	}

	if search := p.search(filter.Query); search != nil {
		preds = append(preds, search)
	}

	return preds
}

func capped[E any](items []E, limit int) []E {
	if len(items) > limit {
		return items[:limit]
	}

	return items
}

func (p *plan) group(group domaintypes.FilterGroup) keel.Pred {
	alternatives := make([]keel.Pred, 0, len(group.Filters))
	for _, f := range group.Filters {
		if pred, ok := p.predicate(f); ok && pred != nil {
			alternatives = append(alternatives, pred)
		}
	}

	if len(alternatives) == 0 {
		return nil
	}

	return keel.Or(alternatives...)
}

func (p *plan) predicate(f domaintypes.FieldFilter) (keel.Pred, bool) {
	fd, ok := p.resolve(f.Field)
	if !ok {
		return nil, false
	}

	switch f.Operator { //nolint:exhaustive // count operators are aggregate filters
	case dbtype.OpIsNull:
		return fd.expr.IsNull(), true
	case dbtype.OpIsNotNull:
		return fd.expr.IsNotNull(), true
	case dbtype.OpEqual, dbtype.OpNotEqual:
		return p.equality(fd, f.Operator, f.Value)
	case dbtype.OpGreaterThan,
		dbtype.OpGreaterThanOrEqual,
		dbtype.OpLessThan,
		dbtype.OpLessThanOrEqual:
		return comparison(fd, f.Operator, f.Value)
	case dbtype.OpContains,
		dbtype.OpStartsWith,
		dbtype.OpEndsWith,
		dbtype.OpLike,
		dbtype.OpILike:
		return pattern(fd, f.Operator, f.Value)
	case dbtype.OpIn, dbtype.OpNotIn:
		return membership(fd, f.Operator, f.Value)
	case dbtype.OpDateRange:
		return dateRange(fd, f.Value)
	case dbtype.OpLastNDays:
		start := time.Now().AddDate(0, 0, -extractDays(f.Value)).Unix()
		return comparison(fd, dbtype.OpGreaterThanOrEqual, start)
	case dbtype.OpNextNDays:
		end := time.Now().AddDate(0, 0, extractDays(f.Value)).Unix()
		return comparison(fd, dbtype.OpLessThanOrEqual, end)
	case dbtype.OpToday:
		return dayWindow(fd, time.Now())
	case dbtype.OpYesterday:
		return dayWindow(fd, time.Now().AddDate(0, 0, -1))
	case dbtype.OpTomorrow:
		return dayWindow(fd, time.Now().AddDate(0, 0, 1))
	default:
		return nil, false
	}
}

func (p *plan) equality(fd field, op dbtype.Operator, value any) (keel.Pred, bool) {
	if s, isString := value.(string); isString && s == "" {
		return nil, true
	}

	text := fd.enum() || !scalarKind(fd)
	expr := fd.expr
	if text {
		expr = expr.Cast(keel.TypeString)
	}

	kind, ok := kindOf(fd.col, text)
	if !ok {
		return nil, false
	}

	v, ok := convert(kind, value)
	if !ok {
		return nil, false
	}

	if op == dbtype.OpNotEqual {
		return expr.Ne(v), true
	}

	return expr.Eq(v), true
}

func scalarKind(fd field) bool {
	_, ok := kindOf(fd.col, false)
	return ok
}

func comparison(fd field, op dbtype.Operator, value any) (keel.Pred, bool) {
	if fd.enum() {
		return nil, false
	}

	kind, ok := kindOf(fd.col, false)
	if !ok {
		return nil, false
	}

	v, ok := convert(kind, value)
	if !ok {
		return nil, false
	}

	switch op { //nolint:exhaustive // only ordering operators reach here
	case dbtype.OpGreaterThan:
		return fd.expr.Gt(v), true
	case dbtype.OpGreaterThanOrEqual:
		return fd.expr.Gte(v), true
	case dbtype.OpLessThan:
		return fd.expr.Lt(v), true
	case dbtype.OpLessThanOrEqual:
		return fd.expr.Lte(v), true
	default:
		return nil, false
	}
}

func textExpr(fd field) keel.DynExpr {
	if !fd.enum() && isText(fd) {
		return fd.expr
	}

	return fd.expr.Cast(keel.TypeString)
}

func isText(fd field) bool {
	kind, ok := kindOf(fd.col, false)
	return ok && kind == valueText
}

func pattern(fd field, op dbtype.Operator, value any) (keel.Pred, bool) {
	raw, ok := toText(value)
	if !ok {
		return nil, false
	}

	s, _ := raw.(string)
	expr := textExpr(fd)

	switch op { //nolint:exhaustive // only pattern operators reach here
	case dbtype.OpContains:
		return expr.ContainsFold(s), true
	case dbtype.OpStartsWith:
		return expr.ILike(escapeLike(s) + "%"), true
	case dbtype.OpEndsWith:
		return expr.ILike("%" + escapeLike(s)), true
	case dbtype.OpLike:
		return expr.Like(s), true
	case dbtype.OpILike:
		return expr.ILike(s), true
	default:
		return nil, false
	}
}

func membership(fd field, op dbtype.Operator, value any) (keel.Pred, bool) {
	text := fd.enum() || !scalarKind(fd)
	expr := fd.expr
	if text {
		expr = expr.Cast(keel.TypeString)
	}

	kind, ok := kindOf(fd.col, text)
	if !ok {
		return nil, false
	}

	list, ok := convertList(kind, value)
	if !ok {
		return nil, false
	}

	if op == dbtype.OpNotIn {
		return expr.NotIn(list), true
	}

	return expr.In(list), true
}

func dateRange(fd field, value any) (keel.Pred, bool) {
	bounds, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}

	preds := make([]keel.Pred, 0, 2)
	if from, hasFrom := bounds["from"]; hasFrom {
		if f, isNumber := toFloat(from); isNumber {
			pred, valid := comparison(fd, dbtype.OpGreaterThanOrEqual, f)
			if !valid {
				return nil, false
			}
			preds = append(preds, pred)
		}
	}

	if to, hasTo := bounds["to"]; hasTo {
		if f, isNumber := toFloat(to); isNumber {
			end, _ := f.(float64)
			pred, valid := comparison(fd, dbtype.OpLessThan, end+secondsPerDay)
			if !valid {
				return nil, false
			}
			preds = append(preds, pred)
		}
	}

	if len(preds) == 0 {
		return nil, true
	}

	return keel.And(preds...), true
}

func dayWindow(fd field, day time.Time) (keel.Pred, bool) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.AddDate(0, 0, 1).Add(-time.Nanosecond)

	from, ok := comparison(fd, dbtype.OpGreaterThanOrEqual, start.Unix())
	if !ok {
		return nil, false
	}

	to, ok := comparison(fd, dbtype.OpLessThanOrEqual, end.Unix())
	if !ok {
		return nil, false
	}

	return keel.And(from, to), true
}

func extractDays(value any) int {
	if v, ok := value.(map[string]any); ok {
		if days, found := v["days"]; found {
			return extractDays(days)
		}

		return 0
	}

	return intutils.IntValue(value)
}

func (p *plan) aggregate(af domaintypes.AggregateFilter) (keel.Pred, bool) {
	rel, ok := p.cat.relation(af.Relation)
	if !ok || !rel.countable() {
		return nil, false
	}

	count, ok := p.relatedCount(rel)
	if !ok {
		return nil, false
	}

	n := int64(af.Value)
	switch af.Operator { //nolint:exhaustive // only count operators are aggregate filters
	case dbtype.OpCountGt:
		return count.Gt(n), true
	case dbtype.OpCountGte:
		return count.Gte(n), true
	case dbtype.OpCountLt:
		return count.Lt(n), true
	case dbtype.OpCountLte:
		return count.Lte(n), true
	case dbtype.OpCountEq:
		return count.Eq(n), true
	default:
		return nil, false
	}
}

func (p *plan) relatedCount(rel relation) (keel.DynExpr, bool) {
	parent := &p.cat.table.Relations[rel.index]
	if len(parent.Local) == 0 || len(parent.Local) != len(parent.Remote) {
		return keel.DynExpr{}, false
	}

	sub := keel.DynFrom(rel.target)
	correlation := make([]keel.Pred, 0, len(parent.Remote))
	for i, remote := range parent.Remote {
		fk := sub.Path(rel.target.Columns[remote].Name)
		correlation = append(correlation, fk.Eq(p.baseColumn(parent.Local[i])))
	}
	sub.Select(keel.DynCount()).Where(correlation...)
	if sub.Err() != nil {
		return keel.DynExpr{}, false
	}

	return keel.DynOf(keel.Scalar(sub)), true
}
