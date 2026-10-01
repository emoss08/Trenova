package pagination

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hostileTenantParams() url.Values {
	params := url.Values{}
	for _, key := range []string{"OrgID", "orgId", "TenantInfo.OrgID"} {
		params.Set(key, pulid.MustNew("org").String())
	}
	for _, key := range []string{"BuID", "buId", "TenantInfo.BuID"} {
		params.Set(key, pulid.MustNew("bu").String())
	}
	for _, key := range []string{"UserID", "userId", "TenantInfo.UserID"} {
		params.Set(key, pulid.MustNew("usr").String())
	}
	return params
}

func TestNewQueryOptions_IgnoresTenantFromQuery(t *testing.T) {
	t.Parallel()

	authCtx := newTestAuthContext()
	opts := NewQueryOptions(createTestContextWithParams(hostileTenantParams()), authCtx)

	assert.Equal(t, authCtx.OrganizationID, opts.TenantInfo.OrgID)
	assert.Equal(t, authCtx.BusinessUnitID, opts.TenantInfo.BuID)
	assert.Equal(t, authCtx.UserID, opts.TenantInfo.UserID)
}

func TestNewSelectQueryRequest_IgnoresTenantFromQuery(t *testing.T) {
	t.Parallel()

	authCtx := newTestAuthContext()
	req := NewSelectQueryRequest(createTestContextWithParams(hostileTenantParams()), authCtx)

	assert.Equal(t, authCtx.OrganizationID, req.TenantInfo.OrgID)
	assert.Equal(t, authCtx.BusinessUnitID, req.TenantInfo.BuID)
	assert.Equal(t, authCtx.UserID, req.TenantInfo.UserID)
}

func TestCursorList_IgnoresTenantFromQuery(t *testing.T) {
	t.Parallel()

	authCtx := newTestAuthContext()
	params := hostileTenantParams()
	c := createTestContextWithParams(params)
	opts := NewQueryOptions(c, authCtx)

	var seen TenantInfo
	CursorList(c, opts, newTestErrorHandler(),
		func(CursorInfo) (*CursorListResult[string], error) {
			seen = opts.TenantInfo
			return &CursorListResult[string]{}, nil
		})

	require.Equal(t, http.StatusOK, c.Writer.Status())
	assert.Equal(t, authCtx.OrganizationID, seen.OrgID)
	assert.Equal(t, authCtx.BusinessUnitID, seen.BuID)
	assert.Equal(t, authCtx.UserID, seen.UserID)
}
