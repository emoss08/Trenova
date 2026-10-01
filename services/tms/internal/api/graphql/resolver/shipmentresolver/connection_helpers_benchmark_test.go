package shipmentresolver

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"

	"github.com/emoss08/trenova/pkg/pagination"
)

var benchmarkConnectionEdges []resolvertest.TestConnectionEdge

func BenchmarkMappedEntityCursorEdgesWithCursorValues(b *testing.B) {
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
				edges, _, err := mappedEntityCursorEdges(
					items,
					sort,
					provider,
					func(item resolvertest.TestCursorEntity) (string, error) {
						return item.Node, nil
					},
					func(node string, cursor string) resolvertest.TestConnectionEdge {
						return resolvertest.TestConnectionEdge{
							Node:   node,
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
