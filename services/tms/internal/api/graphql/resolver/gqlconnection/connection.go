package gqlconnection

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
	Edges           []TEdge
	PageInfo        *gqlmodel.PageInfo
	TotalCount      *int
	TotalCountExact bool
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
		result,
		build,
	)
	if err != nil {
		return nil, err
	}

	return &entityCursorConnectionPage[TEdge]{
		Edges:           edges,
		PageInfo:        PageInfo(result.HasNextPage, LastEdgeCursor(edges, edgeCursor)),
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
	}, nil
}

func EntityCursorEdges[TNode pagination.CursorEntity, TEdge any](
	items []TNode,
	cursors pagination.EdgeCursorProvider,
	build func(TNode, string) TEdge,
) ([]TEdge, error) {
	edges := make([]TEdge, len(items))
	for i, item := range items {
		cursor, err := EdgeCursor(cursors, i)
		if err != nil {
			return nil, err
		}
		edges[i] = build(item, cursor)
	}

	return edges, nil
}

func EdgeCursor(cursors pagination.EdgeCursorProvider, index int) (string, error) {
	if cursors == nil {
		return "", pagination.ErrNoEdgeCursor
	}

	cursor, ok := cursors.EdgeCursorAt(index)
	if !ok {
		return "", pagination.ErrNoEdgeCursor
	}

	return cursor, nil
}

func LastEdgeCursor[TEdge any](edges []TEdge, cursor func(TEdge) string) *string {
	if len(edges) == 0 {
		return nil
	}

	endCursor := cursor(edges[len(edges)-1])
	return &endCursor
}

func PageInfoFor(hasNext bool, lastCursor string) *gqlmodel.PageInfo {
	info := &gqlmodel.PageInfo{HasNextPage: hasNext}
	if lastCursor != "" {
		cursor := lastCursor
		info.EndCursor = &cursor
	}

	return info
}
