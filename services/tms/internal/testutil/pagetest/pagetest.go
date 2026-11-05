package pagetest

import (
	"github.com/emoss08/trenova/pkg/pagination"
)

var Binding = pagination.BindCursor(pagination.TenantInfo{}, "pagetest", nil)

func Keyset(cursor pagination.CursorInfo) string {
	if cursor.After == "" {
		return ""
	}

	key, err := pagination.DecodeKeyCursor(Binding, cursor.After)
	if err != nil {
		return cursor.After
	}

	return key
}

func Page[T pagination.CursorEntity](
	rows []T,
	cursor pagination.CursorInfo,
) *pagination.CursorListResult[T] {
	start := 0
	if after := Keyset(cursor); after != "" {
		for i, row := range rows {
			if row.GetID().String() == after {
				start = i + 1
			}
		}
	}

	limit := cursor.Limit
	if limit <= 0 {
		limit = pagination.DefaultLimit
	}
	end := min(start+limit, len(rows))
	items := rows[start:end]

	result := &pagination.CursorListResult[T]{
		Items:       items,
		HasNextPage: end < len(rows),
		EdgeCursors: make([]string, len(items)),
	}
	for i, item := range items {
		result.EdgeCursors[i] = Cursor(item)
	}
	if cursor.IncludeTotalCount {
		total := len(rows)
		result.TotalCount = &total
		result.TotalCountExact = true
	}

	return result
}

func Cursor(item pagination.CursorEntity) string {
	return KeysetCursor(item.GetID().String())
}

func KeysetCursor(keyset string) string {
	return pagination.EncodeKeyCursor(Binding, keyset)
}

func Of[T pagination.CursorEntity](items []T, limit int) *pagination.CursorListResult[T] {
	return OfTotal(items, limit, nil)
}

func OfTotal[T pagination.CursorEntity](
	items []T,
	limit int,
	total *int,
) *pagination.CursorListResult[T] {
	hasNextPage := len(items) > limit
	if hasNextPage {
		items = items[:limit]
	}

	result := &pagination.CursorListResult[T]{
		Items:           items,
		HasNextPage:     hasNextPage,
		TotalCount:      total,
		TotalCountExact: total != nil,
		EdgeCursors:     make([]string, len(items)),
	}
	for i, item := range items {
		result.EdgeCursors[i] = Cursor(item)
	}

	return result
}
