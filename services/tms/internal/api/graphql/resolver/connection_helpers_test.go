package resolver

import (
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityCursorEdges_EncodesEntityCursor(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("trac_")
	createdAt := int64(1780415883)

	edges, err := entityCursorEdges(
		[]testCursorEntity{
			{
				node:      "TRC-1",
				id:        id,
				createdAt: createdAt,
			},
		},
		nil,
		nil,
		func(node testCursorEntity, cursor string) testConnectionEdge {
			return testConnectionEdge{
				node:   node.node,
				cursor: cursor,
			}
		},
	)
	require.NoError(t, err)
	require.Len(t, edges, 1)

	assert.Equal(t, "TRC-1", edges[0].node)
	decoded, err := pagination.DecodeCursor(edges[0].cursor)
	require.NoError(t, err)
	assert.Equal(t, createdAt, decoded.CreatedAt)
	assert.Equal(t, id, decoded.ID)
}

func TestEntityCursorEdges_UsesExplicitCursorValues(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("trac_")
	edges, err := entityCursorEdges(
		[]testCursorEntity{
			{
				node:      "TRC-1",
				id:        id,
				createdAt: 1780415883,
			},
		},
		[]pagination.CursorSortField{
			{Field: "name", Direction: "asc"},
			{Field: "id", Direction: "asc"},
		},
		testCursorValueProvider{values: [][]any{{"ordered-name", id.String()}}},
		func(node testCursorEntity, cursor string) testConnectionEdge {
			return testConnectionEdge{
				node:   node.node,
				cursor: cursor,
			}
		},
	)
	require.NoError(t, err)
	require.Len(t, edges, 1)

	decoded, err := pagination.DecodeCursor(edges[0].cursor)
	require.NoError(t, err)
	assert.Equal(t, []any{"ordered-name", id.String()}, decoded.Values)
}

type testCursorValueProvider struct {
	values [][]any
}

func (p testCursorValueProvider) CursorValuesAt(index int) ([]any, bool) {
	if index < 0 || index >= len(p.values) {
		return nil, false
	}

	return p.values[index], true
}

func TestPageInfo_EmptyEndCursorStaysNil(t *testing.T) {
	t.Parallel()

	info := pageInfo(true, lastEdgeCursor([]testConnectionEdge{}, func(edge testConnectionEdge) string {
		return edge.cursor
	}))

	assert.True(t, info.HasNextPage)
	assert.Nil(t, info.EndCursor)
}

func TestTotalCountPtr_ReturnsStablePointer(t *testing.T) {
	t.Parallel()

	count := totalCountPtr(42)

	require.NotNil(t, count)
	assert.Equal(t, 42, *count)
}

type testConnectionEdge struct {
	node   string
	cursor string
}

type testCursorEntity struct {
	node      string
	id        pulid.ID
	createdAt int64
}

func (e testCursorEntity) GetID() pulid.ID {
	return e.id
}

func (e testCursorEntity) GetCreatedAt() int64 {
	return e.createdAt
}
