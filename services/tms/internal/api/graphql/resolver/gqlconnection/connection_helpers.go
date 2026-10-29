package gqlconnection

import (
	"github.com/emoss08/trenova/pkg/pagination"
)

func MappedEntityCursorEdges[TEntity pagination.CursorEntity, TNode any, TEdge any](
	items []TEntity,
	cursors pagination.EdgeCursorProvider,
	mapNode func(TEntity) (TNode, error),
	build func(TNode, string) TEdge,
) (edges []TEdge, endCursor *string, err error) {
	if len(items) == 0 {
		return []TEdge{}, nil, nil
	}

	edges = make([]TEdge, len(items))
	for i, item := range items {
		node, mapErr := mapNode(item)
		if mapErr != nil {
			return nil, nil, mapErr
		}

		cursor, cursorErr := EdgeCursor(cursors, i)
		if cursorErr != nil {
			return nil, nil, cursorErr
		}
		edges[i] = build(node, cursor)
		if i == len(items)-1 {
			endCursor = &cursor
		}
	}

	return edges, endCursor, nil
}
