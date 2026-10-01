package resolver

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var benchmarkConnectionEdges []testConnectionEdge

func BenchmarkEntityCursorEdgesWithCursorValues(b *testing.B) {
	sort := []pagination.CursorSortField{
		{Field: "name", Direction: "asc"},
		{Field: "createdAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}

	for _, count := range []int{20, 50, 100} {
		b.Run(fmt.Sprintf("count=%d", count), func(b *testing.B) {
			items, values := benchmarkConnectionCursorItems(count)
			provider := testCursorValueProvider{values: values}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				edges, err := entityCursorEdges(
					items,
					sort,
					provider,
					func(node testCursorEntity, cursor string) testConnectionEdge {
						return testConnectionEdge{
							node:   node.node,
							cursor: cursor,
						}
					},
				)
				if err != nil {
					b.Fatal(err)
				}
				benchmarkConnectionEdges = edges
			}
		})
	}
}

func BenchmarkEntityCursorEdgesWithSortFallback(b *testing.B) {
	sort := []pagination.CursorSortField{
		{Field: "node", Direction: "asc"},
		{Field: "createdAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}

	for _, count := range []int{20, 50, 100} {
		b.Run(fmt.Sprintf("count=%d", count), func(b *testing.B) {
			items := benchmarkConnectionFallbackItems(count)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				edges, err := entityCursorEdges(
					items,
					sort,
					nil,
					func(node benchmarkConnectionEntity, cursor string) testConnectionEdge {
						return testConnectionEdge{
							node:   node.Node,
							cursor: cursor,
						}
					},
				)
				if err != nil {
					b.Fatal(err)
				}
				benchmarkConnectionEdges = edges
			}
		})
	}
}

func BenchmarkMappedEntityCursorEdgesWithCursorValues(b *testing.B) {
	sort := []pagination.CursorSortField{
		{Field: "name", Direction: "asc"},
		{Field: "createdAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}

	for _, count := range []int{20, 50, 100} {
		b.Run(fmt.Sprintf("count=%d", count), func(b *testing.B) {
			items, values := benchmarkConnectionCursorItems(count)
			provider := testCursorValueProvider{values: values}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				edges, _, err := mappedEntityCursorEdges(
					items,
					sort,
					provider,
					func(item testCursorEntity) (string, error) {
						return item.node, nil
					},
					func(node string, cursor string) testConnectionEdge {
						return testConnectionEdge{
							node:   node,
							cursor: cursor,
						}
					},
				)
				if err != nil {
					b.Fatal(err)
				}
				benchmarkConnectionEdges = edges
			}
		})
	}
}

func benchmarkConnectionCursorItems(count int) ([]testCursorEntity, [][]any) {
	items := make([]testCursorEntity, 0, count)
	values := make([][]any, 0, count)
	for i := range count {
		id := pulid.MustNew("trac_")
		name := fmt.Sprintf("TRC-%05d", i)
		createdAt := int64(1710000000000 + i)
		items = append(items, testCursorEntity{
			node:      name,
			id:        id,
			createdAt: createdAt,
		})
		values = append(values, []any{name, createdAt, id.String()})
	}

	return items, values
}

type benchmarkConnectionEntity struct {
	Node      string   `json:"node"`
	ID        pulid.ID `json:"id"`
	CreatedAt int64    `json:"createdAt"`
}

func (e benchmarkConnectionEntity) GetID() pulid.ID {
	return e.ID
}

func (e benchmarkConnectionEntity) GetCreatedAt() int64 {
	return e.CreatedAt
}

func benchmarkConnectionFallbackItems(count int) []benchmarkConnectionEntity {
	items := make([]benchmarkConnectionEntity, 0, count)
	for i := range count {
		items = append(items, benchmarkConnectionEntity{
			Node:      fmt.Sprintf("TRC-%05d", i),
			ID:        pulid.MustNew("trac_"),
			CreatedAt: int64(1710000000000 + i),
		})
	}

	return items
}
