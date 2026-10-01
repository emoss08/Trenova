package authctx

import (
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newScopeTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	return c
}

func TestSetSessionAuthContext_BindsTheTenantDatabaseScope(t *testing.T) {
	t.Parallel()

	c := newScopeTestContext()
	userID, buID, orgID := pulid.MustNew("usr_"), pulid.MustNew("bu_"), pulid.MustNew("org_")

	SetSessionAuthContext(c, SessionAuthContextParams{
		SessionID:      pulid.MustNew("ses_"),
		UserID:         userID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
	})

	want := dbscope.Tenant{OrganizationID: orgID, BusinessUnitID: buID, UserID: userID}
	for name, scope := range map[string]dbscope.Scope{
		"gin context":     dbscope.From(c),
		"request context": dbscope.From(c.Request.Context()),
	} {
		got, ok := scope.Tenant()
		require.True(t, ok, name)
		assert.Equal(t, want, got, name)
	}
}

func TestSetAPIKeyContext_BindsAScopeWithoutAUser(t *testing.T) {
	t.Parallel()

	c := newScopeTestContext()
	buID, orgID := pulid.MustNew("bu_"), pulid.MustNew("org_")

	SetAPIKeyContext(c, pulid.MustNew("ak_"), buID, orgID)

	got, ok := dbscope.TenantFrom(c)
	require.True(t, ok)
	assert.Equal(t, dbscope.Tenant{OrganizationID: orgID, BusinessUnitID: buID}, got)
}

func TestSetCaptureDeviceContext_BindsTheDeviceOwnersScope(t *testing.T) {
	t.Parallel()

	c := newScopeTestContext()
	userID, buID, orgID := pulid.MustNew("usr_"), pulid.MustNew("bu_"), pulid.MustNew("org_")

	SetCaptureDeviceContext(c, pulid.MustNew("cdev_"), userID, buID, orgID)

	got, ok := dbscope.TenantFrom(c.Request.Context())
	require.True(t, ok)
	assert.Equal(t, dbscope.Tenant{OrganizationID: orgID, BusinessUnitID: buID, UserID: userID}, got)
}

func TestSystemScopeDeclaredOnAGinContextWins(t *testing.T) {
	t.Parallel()

	c := newScopeTestContext()
	SetAuthContext(c, pulid.MustNew("usr_"), pulid.MustNew("bu_"), pulid.MustNew("org_"))

	ctx := dbscope.WithSystem(c, "resolve share token")
	assert.True(t, dbscope.IsSystem(ctx))
}
