package agentquerytoolservice

import (
	"maps"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/pagetest"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pageAfter[T pagination.CursorEntity](
	items []T,
	cursor pagination.CursorInfo,
) *pagination.CursorListResult[T] {
	start := 0
	if cursor.Cursor.Keyset != "" {
		for idx, item := range items {
			if item.GetID().String() == cursor.Cursor.Keyset {
				start = idx + 1
				break
			}
		}
	}
	end := min(len(items), start+cursor.Limit+1)

	return pagetest.Of(items[start:end], cursor.Limit)
}

func cursorIDs[T pagination.CursorEntity](items []T) []pulid.ID {
	ids := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.GetID())
	}

	return ids
}

type cursorPagingCase struct {
	query    func(params map[string]any) (any, error)
	received func() pagination.CursorInfo
	args     map[string]any
	changed  map[string]any
	ids      []pulid.ID
	limit    int
}

type cursorPageRead struct {
	Count      int    `json:"count"`
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor"`
	NextOffset *int   `json:"nextOffset"`
}

func readCursorPage(t *testing.T, result any) cursorPageRead {
	t.Helper()

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	var read cursorPageRead
	require.NoError(t, sonic.Unmarshal(encoded, &read))

	return read
}

func (c cursorPagingCase) call(extra map[string]any, args map[string]any) (any, error) {
	params := make(map[string]any, len(args)+len(extra)+1)
	maps.Copy(params, args)
	maps.Copy(params, extra)
	params[paramLimit] = float64(c.limit)

	return c.query(params)
}

func assertCursorPaging(t *testing.T, c cursorPagingCase) {
	t.Helper()
	require.Greater(t, len(c.ids), c.limit, "the case needs more rows than one page")

	result, err := c.call(nil, c.args)
	require.NoError(t, err)
	first := readCursorPage(t, result)
	assert.Empty(t, c.received().Cursor.Keyset, "the first page starts from the top")
	assert.Equal(t, c.limit, first.Count)
	assert.True(t, first.HasMore)
	require.NotEmpty(t, first.NextCursor, "a full page names where the next one starts")
	assert.Nil(t, first.NextOffset, "a cursor list never offers an offset")

	seen := first.Count
	after := first.NextCursor
	for pages := 1; after != ""; pages++ {
		require.Less(t, pages, len(c.ids), "paging never reached the last page")

		result, err = c.call(map[string]any{paramAfter: after}, c.args)
		require.NoError(t, err)
		page := readCursorPage(t, result)
		assert.Equal(t, c.ids[seen-1].String(), c.received().Cursor.Keyset,
			"after reaches the repository as the last row of the previous page")
		assert.Equal(t, page.HasMore, page.NextCursor != "")
		seen += page.Count
		after = page.NextCursor
	}
	assert.Equal(t, len(c.ids), seen, "the pages together hold every row once")

	_, err = c.call(map[string]any{paramAfter: first.NextCursor}, c.changed)
	require.ErrorIs(t, err, ErrStaleCursor, "a cursor is refused under changed arguments")

	_, err = c.call(map[string]any{paramOffset: float64(c.limit)}, c.args)
	require.ErrorIs(t, err, errOffsetOnCursorList, "an offset is refused, never ignored")
	_, err = c.call(map[string]any{paramOffset: float64(0), paramAfter: first.NextCursor}, c.args)
	require.NoError(t, err, "an offset of nought asks for nothing and is let through")

	for _, malformed := range []string{
		"not a cursor",
		pagination.EncodeKeyCursor("0000000000000000", "x"),
		first.NextCursor + "x",
	} {
		_, err = c.call(map[string]any{paramAfter: malformed}, c.args)
		require.ErrorIs(t, err, ErrStaleCursor, "malformed cursor %q", malformed)
	}
}

func TestCursorScope_IgnoresLimitAndAfterAndKeyOrder(t *testing.T) {
	t.Parallel()

	first, err := cursorScope("worker_list", map[string]any{
		"query": "ortiz", "status": "Active", paramLimit: float64(5),
	})
	require.NoError(t, err)
	second, err := cursorScope("worker_list", map[string]any{
		"status": "Active", "query": "ortiz", paramLimit: float64(9), paramAfter: "abc",
	})
	require.NoError(t, err)
	assert.Equal(t, first, second)

	otherTool, err := cursorScope("tractor_list", map[string]any{
		"query": "ortiz", "status": "Active",
	})
	require.NoError(t, err)
	assert.NotEqual(t, first, otherTool, "a cursor is bound to the tool that issued it")
}

func TestReadCursorWindow_DecodesTheRepositoryCursor(t *testing.T) {
	t.Parallel()

	inner, err := pagination.EncodeKeysetCursor(nil, pulid.MustNew("wrk_").String())
	require.NoError(t, err)
	params := map[string]any{"query": "ortiz"}
	scope, err := cursorScope("worker_list", params)
	require.NoError(t, err)
	params[paramAfter] = pagination.EncodeKeyCursor(scope, inner)

	window, err := readCursorWindow("worker_list", params, defaultListLimit, maxListLimit)
	require.NoError(t, err)
	assert.Equal(t, inner, window.info.After)
	assert.False(t, window.info.IncludeTotalCount)
	assert.False(t, window.first())

	_, err = readCursorWindow("tractor_list", params, defaultListLimit, maxListLimit)
	require.ErrorIs(t, err, ErrStaleCursor)
}
