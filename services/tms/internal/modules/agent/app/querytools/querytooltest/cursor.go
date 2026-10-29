package querytooltest

import (
	"maps"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/testutil/pagetest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func PageAfter[T pagination.CursorEntity](
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

func CursorIDs[T pagination.CursorEntity](items []T) []pulid.ID {
	ids := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.GetID())
	}

	return ids
}

type CursorPagingCase struct {
	Query    func(params map[string]any) (any, error)
	Received func() pagination.CursorInfo
	Args     map[string]any
	Changed  map[string]any
	IDs      []pulid.ID
	Limit    int
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

func (c CursorPagingCase) call(extra map[string]any, args map[string]any) (any, error) {
	params := make(map[string]any, len(args)+len(extra)+1)
	maps.Copy(params, args)
	maps.Copy(params, extra)
	params[querytool.ParamLimit] = float64(c.Limit)

	return c.Query(params)
}

func AssertCursorPaging(t *testing.T, c CursorPagingCase) {
	t.Helper()
	require.Greater(t, len(c.IDs), c.Limit, "the case needs more rows than one page")

	result, err := c.call(nil, c.Args)
	require.NoError(t, err)
	first := readCursorPage(t, result)
	assert.Empty(t, c.Received().Cursor.Keyset, "the first page starts from the top")
	assert.Equal(t, c.Limit, first.Count)
	assert.True(t, first.HasMore)
	require.NotEmpty(t, first.NextCursor, "a full page names where the next one starts")
	assert.Nil(t, first.NextOffset, "a cursor list never offers an offset")

	seen := first.Count
	after := first.NextCursor
	for pages := 1; after != ""; pages++ {
		require.Less(t, pages, len(c.IDs), "paging never reached the last page")

		result, err = c.call(map[string]any{querytool.ParamAfter: after}, c.Args)
		require.NoError(t, err)
		page := readCursorPage(t, result)
		assert.Equal(t, c.IDs[seen-1].String(), c.Received().Cursor.Keyset,
			"after reaches the repository as the last row of the previous page")
		assert.Equal(t, page.HasMore, page.NextCursor != "")
		seen += page.Count
		after = page.NextCursor
	}
	assert.Equal(t, len(c.IDs), seen, "the pages together hold every row once")

	_, err = c.call(map[string]any{querytool.ParamAfter: first.NextCursor}, c.Changed)
	require.ErrorIs(t, err, querytool.ErrStaleCursor, "a cursor is refused under changed arguments")

	_, err = c.call(map[string]any{querytool.ParamOffset: float64(c.Limit)}, c.Args)
	require.ErrorIs(t, err, querytool.ErrOffsetOnCursorList, "an offset is refused, never ignored")
	_, err = c.call(
		map[string]any{querytool.ParamOffset: float64(0), querytool.ParamAfter: first.NextCursor},
		c.Args,
	)
	require.NoError(t, err, "an offset of nought asks for nothing and is let through")

	for _, malformed := range []string{
		"not a cursor",
		pagination.EncodeKeyCursor("0000000000000000", "x"),
		first.NextCursor + "x",
	} {
		_, err = c.call(map[string]any{querytool.ParamAfter: malformed}, c.Args)
		require.ErrorIs(t, err, querytool.ErrStaleCursor, "malformed cursor %q", malformed)
	}
}
