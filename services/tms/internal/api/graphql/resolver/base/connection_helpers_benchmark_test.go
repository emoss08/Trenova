package base

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var benchmarkConnectionEdges []resolvertest.TestConnectionEdge

func BenchmarkEntityCursorEdgesWithCursorValues(b *testing.B) {
	sort := []pagination.CursorSortField{
		{Field: "name", Direction: "asc"},
		{Field: "createdAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}

	for _, count := range []int{20, 50, 100} {
		b.Run(fmt.Sprintf("count=%d", count), func(b *testing.B) {
			items, values := resolvertest.BenchmarkConnectionCursorItems(count)
			provider := resolvertest.TestCursorValueProvider{Values: values}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				edges, err := EntityCursorEdges(
					items,
					sort,
					provider,
					func(node resolvertest.TestCursorEntity, cursor string) resolvertest.TestConnectionEdge {
						return resolvertest.TestConnectionEdge{
							Node:   node.Node,
							Cursor: cursor,
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
				edges, err := EntityCursorEdges(
					items,
					sort,
					nil,
					func(node benchmarkConnectionEntity, cursor string) resolvertest.TestConnectionEdge {
						return resolvertest.TestConnectionEdge{
							Node:   node.Node,
							Cursor: cursor,
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
