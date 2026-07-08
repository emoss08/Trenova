package pagination

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
)

var benchmarkCursor string

func BenchmarkEncodeCursorFromEntityWithValues(b *testing.B) {
	sort := []CursorSortField{
		{Field: "code", Direction: "asc"},
		{Field: "createdAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}
	item := cursorTestItem{
		ID:        pulid.MustNew("item_"),
		CreatedAt: 1710000000000,
		Name:      "TR-1000",
	}
	values := []any{"TR-1000", int64(1710000000000), item.ID.String()}

	b.ReportAllocs()
	for b.Loop() {
		cursor, err := EncodeCursorFromEntityWithValues(item, sort, values)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkCursor = cursor
	}
}

func BenchmarkEncodeCursorFromEntityWithSort(b *testing.B) {
	sort := []CursorSortField{
		{Field: "name", Direction: "asc"},
		{Field: "createdAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}
	item := cursorTestItem{
		ID:        pulid.MustNew("item_"),
		CreatedAt: 1710000000000,
		Name:      "TR-1000",
	}

	b.ReportAllocs()
	for b.Loop() {
		cursor, err := EncodeCursorFromEntityWithSort(item, sort)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkCursor = cursor
	}
}

func BenchmarkCursorListResultWithCursorValues(b *testing.B) {
	sort := []CursorSortField{
		{Field: "code", Direction: "asc"},
		{Field: "createdAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}

	for _, limit := range []int{20, 50, 100} {
		b.Run(fmt.Sprintf("limit=%d", limit), func(b *testing.B) {
			items, values := benchmarkCursorItems(limit + 1)
			total := 10_000

			b.ReportAllocs()
			for b.Loop() {
				result := NewCursorListResultWithTotalCount(items, limit, &total).WithCursorSort(sort)
				if err := result.WithCursorValues(values); err != nil {
					b.Fatal(err)
				}
				benchmarkCursor = ""
				if cursorValues, ok := result.CursorValuesAt(len(result.Items) - 1); ok {
					cursor, err := EncodeCursorFromEntityWithValues(
						result.Items[len(result.Items)-1],
						result.CursorSort,
						cursorValues,
					)
					if err != nil {
						b.Fatal(err)
					}
					benchmarkCursor = cursor
				}
			}
		})
	}
}

func benchmarkCursorItems(count int) ([]cursorTestItem, [][]any) {
	items := make([]cursorTestItem, 0, count)
	values := make([][]any, 0, count)
	for i := range count {
		id := pulid.MustNew("item_")
		createdAt := int64(1710000000000 + i)
		code := fmt.Sprintf("TR-%05d", i)
		items = append(items, cursorTestItem{
			ID:        id,
			CreatedAt: createdAt,
			Name:      code,
		})
		values = append(values, []any{code, createdAt, id.String()})
	}

	return items, values
}
