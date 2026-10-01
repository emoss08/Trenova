package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/tenantboundary"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTenantBoundaryRouter(
	t *testing.T,
	recorder *securityaudittest.Recorder,
	userID, buID, orgID pulid.ID,
	handler gin.HandlerFunc,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	m := NewTenantBoundaryMiddleware(TenantBoundaryParams{Auditor: recorder, Logger: zap.NewNop()})
	router := gin.New()
	router.Use(m.Track())
	router.PATCH("/organizations/:id/", func(c *gin.Context) {
		if orgID.IsNotNil() {
			authctx.SetAuthContext(c, userID, buID, orgID)
		}
		handler(c)
	})
	return router
}

func TestTenantBoundaryRecordsARefusedCrossTenantRequest(t *testing.T) {
	recorder := &securityaudittest.Recorder{}
	userID, buID, orgID := pulid.MustNew("usr_"), pulid.MustNew("bu_"), pulid.MustNew("org_")
	foreign := pulid.MustNew("org_")

	router := newTenantBoundaryRouter(t, recorder, userID, buID, orgID, func(c *gin.Context) {
		tenantboundary.Report(c.Request.Context(), tenantboundary.Violation{
			Source:         tenantboundary.SourcePath,
			Field:          "id",
			OrganizationID: foreign,
		})
		c.Status(http.StatusNotFound)
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/organizations/"+foreign.String()+"/", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	change := recorder.Only(t)
	assert.Equal(t, permission.ResourceOrganization, change.Resource)
	assert.Equal(t, foreign.String(), change.ResourceID)
	assert.Equal(t, permission.OpUpdate, change.Operation)
	assert.Equal(t, orgID, change.OrganizationID)
	assert.Equal(t, buID, change.BusinessUnitID)
	assert.Equal(t, userID, change.Actor.UserID)
	assert.Equal(t, services.PrincipalTypeUser, change.Actor.PrincipalType)
	assert.Equal(t, "path", change.Metadata["source"])
	assert.Equal(t, "/organizations/:id/", change.Metadata["route"])
	assert.Equal(t, http.StatusNotFound, change.Metadata["status"])
}

func TestTenantBoundaryRecordsNothingForAnOrdinaryRequest(t *testing.T) {
	recorder := &securityaudittest.Recorder{}
	router := newTenantBoundaryRouter(t, recorder, pulid.MustNew("usr_"), pulid.MustNew("bu_"), pulid.MustNew("org_"),
		func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/organizations/x/", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	assert.Empty(t, recorder.Changes())
}

func TestTenantBoundaryWithoutASessionOnlyLogs(t *testing.T) {
	recorder := &securityaudittest.Recorder{}
	router := newTenantBoundaryRouter(t, recorder, pulid.Nil, pulid.Nil, pulid.Nil, func(c *gin.Context) {
		require.True(t, tenantboundary.Report(c, tenantboundary.Violation{Source: tenantboundary.SourceRequestBody}))
		c.Status(http.StatusForbidden)
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/organizations/x/", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	assert.Empty(t, recorder.Changes())
}

func TestOperationForMethod(t *testing.T) {
	t.Parallel()

	assert.Equal(t, permission.OpRead, operationForMethod(http.MethodGet))
	assert.Equal(t, permission.OpCreate, operationForMethod(http.MethodPost))
	assert.Equal(t, permission.OpUpdate, operationForMethod(http.MethodPut))
	assert.Equal(t, permission.OpDelete, operationForMethod(http.MethodDelete))
}
