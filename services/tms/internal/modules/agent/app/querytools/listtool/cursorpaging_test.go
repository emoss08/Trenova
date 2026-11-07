package listtool

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/testutil/pagetest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cursorTenant = pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

func cursorCall(tool string, params map[string]any) querytool.CursorCall {
	return querytool.CursorCall{Tool: tool, Tenant: cursorTenant, Params: params}
}

func TestBindCall_IgnoresLimitAndAfterAndKeyOrder(t *testing.T) {
	t.Parallel()

	first, err := querytool.BindCall(cursorCall("worker_list", map[string]any{
		"query": "ortiz", "status": "Active", querytool.ParamLimit: float64(5),
	}))
	require.NoError(t, err)
	second, err := querytool.BindCall(cursorCall("worker_list", map[string]any{
		"status": "Active", "query": "ortiz", querytool.ParamLimit: float64(9), querytool.ParamAfter: "abc",
	}))
	require.NoError(t, err)
	assert.Equal(t, first, second)

	otherTool, err := querytool.BindCall(cursorCall("tractor_list", map[string]any{
		"query": "ortiz", "status": "Active",
	}))
	require.NoError(t, err)
	assert.NotEqual(t, first, otherTool, "a cursor is bound to the tool that issued it")
}

func TestReadCursorWindow_DecodesTheRepositoryCursor(t *testing.T) {
	t.Parallel()

	inner := pagetest.KeysetCursor(pulid.MustNew("wrk_").String())
	params := map[string]any{"query": "ortiz"}
	binding, err := querytool.BindCall(cursorCall("worker_list", params))
	require.NoError(t, err)
	params[querytool.ParamAfter] = pagination.EncodeKeyCursor(binding, inner)

	window, err := querytool.ReadCursorWindow(
		cursorCall("worker_list", params),
		DefaultLimit,
		MaxLimit,
	)
	require.NoError(t, err)
	assert.Equal(t, inner, window.Info.After)
	assert.False(t, window.Info.IncludeTotalCount)
	assert.False(t, window.First())

	_, err = querytool.ReadCursorWindow(cursorCall("tractor_list", params), DefaultLimit, MaxLimit)
	require.ErrorIs(t, err, querytool.ErrStaleCursor)
}

func TestReadCursorWindow_RefusesACursorIssuedToAnotherTenant(t *testing.T) {
	t.Parallel()

	params := map[string]any{"query": "ortiz"}
	binding, err := querytool.BindCall(cursorCall("worker_list", params))
	require.NoError(t, err)
	params[querytool.ParamAfter] = pagination.EncodeKeyCursor(
		binding,
		pagetest.KeysetCursor(pulid.MustNew("wrk_").String()),
	)

	other := querytool.CursorCall{
		Tool:   "worker_list",
		Tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: cursorTenant.BuID},
		Params: params,
	}
	_, err = querytool.ReadCursorWindow(other, DefaultLimit, MaxLimit)
	require.ErrorIs(t, err, querytool.ErrStaleCursor)
}

func TestReadOffsetWindow_ReadsOnlyItsOwnSealedOffset(t *testing.T) {
	t.Parallel()

	params := map[string]any{"status": "Active"}
	binding, err := querytool.BindCall(cursorCall("worker_credentials", params))
	require.NoError(t, err)
	params[querytool.ParamAfter] = querytool.OffsetWindow{Binding: binding}.At(30)

	window, err := querytool.ReadOffsetWindow(cursorCall("worker_credentials", params), 10, 50)
	require.NoError(t, err)
	assert.Equal(t, 30, window.Offset)

	params[querytool.ParamAfter] = pagination.EncodeKeyCursor(binding, "30")
	_, err = querytool.ReadOffsetWindow(cursorCall("worker_credentials", params), 10, 50)
	require.ErrorIs(
		t,
		err,
		querytool.ErrStaleCursor,
		"an offset written into another kind is refused",
	)
}
