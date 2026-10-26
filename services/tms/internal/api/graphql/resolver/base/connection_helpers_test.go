package base

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testEdge(node resolvertest.TestCursorEntity, cursor string) resolvertest.TestConnectionEdge {
	return resolvertest.TestConnectionEdge{Node: node.Node, Cursor: cursor}
}

func TestEntityCursorEdges_UsesTheResultsEdgeCursors(t *testing.T) {
	t.Parallel()

	items := []resolvertest.TestCursorEntity{
		{Node: "TRC-1", ID: pulid.MustNew("trac_"), CreatedAt: 1780415883},
		{Node: "TRC-2", ID: pulid.MustNew("trac_"), CreatedAt: 1780415884},
	}
	result := &pagination.CursorListResult[resolvertest.TestCursorEntity]{
		Items:       items,
		EdgeCursors: []string{"first", "second"},
	}

	edges, err := EntityCursorEdges(items, result, testEdge)
	require.NoError(t, err)
	require.Len(t, edges, 2)
	assert.Equal(t, "TRC-1", edges[0].Node)
	assert.Equal(t, "first", edges[0].Cursor)
	assert.Equal(t, "second", edges[1].Cursor)
}

func TestEntityCursorEdges_RefusesARowWithoutACursor(t *testing.T) {
	t.Parallel()

	items := []resolvertest.TestCursorEntity{
		{Node: "TRC-1", ID: pulid.MustNew("trac_")},
		{Node: "TRC-2", ID: pulid.MustNew("trac_")},
	}
	result := &pagination.CursorListResult[resolvertest.TestCursorEntity]{
		Items:       items,
		EdgeCursors: []string{"only-one"},
	}

	_, err := EntityCursorEdges(items, result, testEdge)
	require.ErrorIs(t, err, pagination.ErrNoEdgeCursor)

	_, err = EntityCursorEdges(items, nil, testEdge)
	require.ErrorIs(t, err, pagination.ErrNoEdgeCursor)
}

func TestEntityCursorConnection_EndsOnTheLastEdgesCursor(t *testing.T) {
	t.Parallel()

	total := 7
	result := &pagination.CursorListResult[resolvertest.TestCursorEntity]{
		Items: []resolvertest.TestCursorEntity{
			{Node: "TRC-1", ID: pulid.MustNew("trac_")},
			{Node: "TRC-2", ID: pulid.MustNew("trac_")},
		},
		HasNextPage:     true,
		TotalCount:      &total,
		TotalCountExact: true,
		EdgeCursors:     []string{"first", "second"},
	}

	page, err := EntityCursorConnection(result, testEdge, func(edge resolvertest.TestConnectionEdge) string {
		return edge.Cursor
	})
	require.NoError(t, err)
	require.Len(t, page.Edges, 2)
	require.NotNil(t, page.PageInfo.EndCursor)
	assert.Equal(t, "second", *page.PageInfo.EndCursor)
	assert.True(t, page.PageInfo.HasNextPage)
	assert.Equal(t, &total, page.TotalCount)
	assert.True(t, page.TotalCountExact)
}
