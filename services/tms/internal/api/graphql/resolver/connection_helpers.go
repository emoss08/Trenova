package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/pagination"
)

func pageInfo(hasNextPage bool, endCursor *string) *gqlmodel.PageInfo {
	return &gqlmodel.PageInfo{
		HasNextPage: hasNextPage,
		EndCursor:   endCursor,
	}
}

func totalCountPtr(total int) *int {
	return &total
}

func entityCursorEdges[TNode pagination.CursorEntity, TEdge any](
	items []TNode,
	sort []pagination.CursorSortField,
	cursorValues pagination.CursorValueProvider,
	build func(TNode, string) TEdge,
) ([]TEdge, error) {
	edges := make([]TEdge, len(items))
	for i, item := range items {
		cursor, err := encodeConnectionCursor(item, sort, cursorValuesAt(cursorValues, i))
		if err != nil {
			return nil, err
		}
		edges[i] = build(item, cursor)
	}

	return edges, nil
}

func mappedEntityCursorEdges[TEntity pagination.CursorEntity, TNode any, TEdge any](
	items []TEntity,
	sort []pagination.CursorSortField,
	cursorValues pagination.CursorValueProvider,
	mapNode func(TEntity) (TNode, error),
	build func(TNode, string) TEdge,
) ([]TEdge, *string, error) {
	if len(items) == 0 {
		return []TEdge{}, nil, nil
	}

	edges := make([]TEdge, len(items))
	var endCursor *string
	for i, item := range items {
		node, err := mapNode(item)
		if err != nil {
			return nil, nil, err
		}

		cursor, err := encodeConnectionCursor(item, sort, cursorValuesAt(cursorValues, i))
		if err != nil {
			return nil, nil, err
		}
		edges[i] = build(node, cursor)
		if i == len(items)-1 {
			endCursor = &cursor
		}
	}

	return edges, endCursor, nil
}

func encodeConnectionCursor[TNode pagination.CursorEntity](
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

func cursorValuesAt(provider pagination.CursorValueProvider, index int) []any {
	if provider == nil {
		return nil
	}

	values, ok := provider.CursorValuesAt(index)
	if !ok {
		return nil
	}

	return values
}

func lastEdgeCursor[TEdge any](edges []TEdge, cursor func(TEdge) string) *string {
	if len(edges) == 0 {
		return nil
	}

	endCursor := cursor(edges[len(edges)-1])
	return &endCursor
}
