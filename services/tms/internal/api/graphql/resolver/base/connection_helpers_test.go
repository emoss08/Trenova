package base

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityCursorEdges_EncodesEntityCursor(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("trac_")
	createdAt := int64(1780415883)

	edges, err := EntityCursorEdges(
		[]resolvertest.TestCursorEntity{
			{
				Node:      "TRC-1",
				ID:        id,
				CreatedAt: createdAt,
			},
		},
		nil,
		nil,
		func(node resolvertest.TestCursorEntity, cursor string) resolvertest.TestConnectionEdge {
			return resolvertest.TestConnectionEdge{
				Node:   node.Node,
				Cursor: cursor,
			}
		},
	)
	require.NoError(t, err)
	require.Len(t, edges, 1)

	assert.Equal(t, "TRC-1", edges[0].Node)
	decoded, err := pagination.DecodeCursor(edges[0].Cursor)
	require.NoError(t, err)
	assert.Equal(t, createdAt, decoded.CreatedAt)
	assert.Equal(t, id, decoded.ID)
}

func TestEntityCursorEdges_UsesExplicitCursorValues(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("trac_")
	edges, err := EntityCursorEdges(
		[]resolvertest.TestCursorEntity{
			{
				Node:      "TRC-1",
				ID:        id,
				CreatedAt: 1780415883,
			},
		},
		[]pagination.CursorSortField{
			{Field: "name", Direction: "asc"},
			{Field: "id", Direction: "asc"},
		},
		resolvertest.TestCursorValueProvider{Values: [][]any{{"ordered-name", id.String()}}},
		func(node resolvertest.TestCursorEntity, cursor string) resolvertest.TestConnectionEdge {
			return resolvertest.TestConnectionEdge{
				Node:   node.Node,
				Cursor: cursor,
			}
		},
	)
	require.NoError(t, err)
	require.Len(t, edges, 1)

	decoded, err := pagination.DecodeCursor(edges[0].Cursor)
	require.NoError(t, err)
	assert.Equal(t, []any{"ordered-name", id.String()}, decoded.Values)
}

func TestPageInfo_EmptyEndCursorStaysNil(t *testing.T) {
	t.Parallel()

	info := PageInfo(
		true,
		LastEdgeCursor([]resolvertest.TestConnectionEdge{}, func(edge resolvertest.TestConnectionEdge) string {
			return edge.Cursor
		}),
	)

	assert.True(t, info.HasNextPage)
	assert.Nil(t, info.EndCursor)
}

func TestTotalCountPtr_ReturnsStablePointer(t *testing.T) {
	t.Parallel()

	count := new(42)

	require.NotNil(t, count)
	assert.Equal(t, 42, *count)
}
