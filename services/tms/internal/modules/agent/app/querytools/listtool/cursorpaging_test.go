package listtool

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorScope_IgnoresLimitAndAfterAndKeyOrder(t *testing.T) {
	t.Parallel()

	first, err := querytool.CursorScope("worker_list", map[string]any{
		"query": "ortiz", "status": "Active", querytool.ParamLimit: float64(5),
	})
	require.NoError(t, err)
	second, err := querytool.CursorScope("worker_list", map[string]any{
		"status": "Active", "query": "ortiz", querytool.ParamLimit: float64(9), querytool.ParamAfter: "abc",
	})
	require.NoError(t, err)
	assert.Equal(t, first, second)

	otherTool, err := querytool.CursorScope("tractor_list", map[string]any{
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
	scope, err := querytool.CursorScope("worker_list", params)
	require.NoError(t, err)
	params[querytool.ParamAfter] = pagination.EncodeKeyCursor(scope, inner)

	window, err := querytool.ReadCursorWindow("worker_list", params, DefaultLimit, MaxLimit)
	require.NoError(t, err)
	assert.Equal(t, inner, window.Info.After)
	assert.False(t, window.Info.IncludeTotalCount)
	assert.False(t, window.First())

	_, err = querytool.ReadCursorWindow("tractor_list", params, DefaultLimit, MaxLimit)
	require.ErrorIs(t, err, querytool.ErrStaleCursor)
}
