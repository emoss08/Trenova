package listquery

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	defaultSortField = "createdAt"
	idSortField      = "id"
)

func (p *plan) orders(sorts []domaintypes.SortField) []keel.Order {
	sorts = capped(sorts, MaxSortFields)
	out := make([]keel.Order, 0, len(sorts))
	for _, sort := range sorts {
		if !sort.Direction.IsValid() {
			continue
		}

		fd, ok := p.resolve(sort.Field)
		if !ok {
			continue
		}

		if sort.Direction == dbtype.SortDirectionDesc {
			out = append(out, fd.expr.Desc())
			continue
		}
		out = append(out, fd.expr.Asc())
	}

	return out
}

func (p *plan) keysetOrder(
	sorts []domaintypes.SortField,
) ([]keel.Order, []pagination.CursorSortField, error) {
	normalized := normalizeKeysetSort(sorts)
	orders := make([]keel.Order, 0, len(normalized))
	shape := make([]pagination.CursorSortField, 0, len(normalized))

	for _, sort := range normalized {
		if !sort.Direction.IsValid() {
			return nil, nil, errortypes.NewValidationError(
				"sort.direction",
				errortypes.ErrInvalid,
				"Sort direction is not supported",
			)
		}

		fd, ok := p.resolve(sort.Field)
		if !ok || !fd.base {
			return nil, nil, errortypes.NewValidationError(
				"sort.field",
				errortypes.ErrInvalid,
				"Sort field is not supported",
			)
		}

		col := p.baseColumn(fd.index)
		order := col.Asc()
		if sort.Direction == dbtype.SortDirectionDesc {
			order = col.Desc()
		}
		if fd.col.MayBeNull() {
			order = order.NullsLast()
		}

		orders = append(orders, order)
		shape = append(shape, pagination.CursorSortField{
			Field:     sort.Field,
			Direction: string(sort.Direction),
		})
	}

	return orders, shape, nil
}

func normalizeKeysetSort(sorts []domaintypes.SortField) []domaintypes.SortField {
	normalized := make([]domaintypes.SortField, 0, min(len(sorts)+1, MaxSortFields))
	seen := make(map[string]bool, len(sorts)+1)
	nonIDCount := 0

	if len(sorts) == 0 {
		normalized = append(normalized, domaintypes.SortField{
			Field:     defaultSortField,
			Direction: dbtype.SortDirectionDesc,
		})
		seen[defaultSortField] = true
		nonIDCount++
	}

	for _, sort := range sorts {
		if sort.Field == "" || seen[sort.Field] {
			continue
		}
		if sort.Field != idSortField && nonIDCount >= MaxSortFields-1 {
			continue
		}
		normalized = append(normalized, sort)
		seen[sort.Field] = true
		if sort.Field != idSortField {
			nonIDCount++
		}
	}

	if !seen[idSortField] {
		direction := dbtype.SortDirectionDesc
		if len(normalized) > 0 {
			direction = normalized[len(normalized)-1].Direction
		}
		normalized = append(normalized, domaintypes.SortField{
			Field:     idSortField,
			Direction: direction,
		})
	}

	return normalized
}
