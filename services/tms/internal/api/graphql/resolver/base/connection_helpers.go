package base

import (
	"errors"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/pagination"
)

func PageInfo(hasNextPage bool, endCursor *string) *gqlmodel.PageInfo {
	return &gqlmodel.PageInfo{
		HasNextPage: hasNextPage,
		EndCursor:   endCursor,
	}
}

type entityCursorConnectionPage[TEdge any] struct {
	Edges      []TEdge
	PageInfo   *gqlmodel.PageInfo
	TotalCount *int
}

func EntityCursorConnection[TNode pagination.CursorEntity, TEdge any](
	result *pagination.CursorListResult[TNode],
	build func(TNode, string) TEdge,
	edgeCursor func(TEdge) string,
) (*entityCursorConnectionPage[TEdge], error) {
	if result == nil {
		return nil, errors.New("cursor list result is required")
	}

	edges, err := EntityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		build,
	)
	if err != nil {
		return nil, err
	}

	return &entityCursorConnectionPage[TEdge]{
		Edges:      edges,
		PageInfo:   PageInfo(result.HasNextPage, LastEdgeCursor(edges, edgeCursor)),
		TotalCount: result.TotalCount,
	}, nil
}

func EntityCursorEdges[TNode pagination.CursorEntity, TEdge any](
	items []TNode,
	sort []pagination.CursorSortField,
	cursorValues pagination.CursorValueProvider,
	build func(TNode, string) TEdge,
) ([]TEdge, error) {
	edges := make([]TEdge, len(items))
	for i, item := range items {
		cursor, err := EncodeConnectionCursor(item, sort, CursorValuesAt(cursorValues, i))
		if err != nil {
			return nil, err
		}
		edges[i] = build(item, cursor)
	}

	return edges, nil
}

func EncodeConnectionCursor[TNode pagination.CursorEntity](
	item TNode,
	sort []pagination.CursorSortField,
	values []any,
) (string, error) {
	if len(sort) == 0 {
		return pagination.EncodeCursorFromEntity(item)
	}
	if len(values) > 0 {
		return pagination.EncodeCursorFromEntityWithValues(item, sort, values)
	}

	return pagination.EncodeCursorFromEntityWithSort(item, sort)
}

func CursorValuesAt(provider pagination.CursorValueProvider, index int) []any {
	if provider == nil {
		return nil
	}

	values, ok := provider.CursorValuesAt(index)
	if !ok {
		return nil
	}

	return values
}

func LastEdgeCursor[TEdge any](edges []TEdge, cursor func(TEdge) string) *string {
	if len(edges) == 0 {
		return nil
	}

	endCursor := cursor(edges[len(edges)-1])
	return &endCursor
}
