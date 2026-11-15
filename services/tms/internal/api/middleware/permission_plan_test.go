package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func offPlanResult() *permissionports.PermissionCheckResult {
	denial := permissionports.NewPlanDenial(platformcatalog.FeatureAccounting)

	return &permissionports.PermissionCheckResult{
		Reason:     permissionports.ReasonNotOnPlan,
		PlanDenial: &denial,
	}
}

func serveGuarded(t *testing.T, guard gin.HandlerFunc) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	handlerCalled := false
	r := gin.New()
	r.GET(
		"/test",
		setAuthMiddleware(pulid.MustNew("usr_"), pulid.MustNew("bu_"), pulid.MustNew("org_")),
		guard,
		func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	return w, handlerCalled
}

func decodeProblem(t *testing.T, w *httptest.ResponseRecorder) helpers.ProblemDetail {
	t.Helper()

	var problem helpers.ProblemDetail
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &problem))

	return problem
}

func assertPlanRestricted(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	problem := decodeProblem(t, w)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, string(errortypes.ErrPlanRestricted), problem.Code)
	assert.Equal(t, "tms.accounting", problem.Params["feature"])
	assert.Equal(t, "Accounting & Cash", problem.Params["addOn"])
}

func TestRequirePermission_OffPlanResourceIsPlanRestricted(t *testing.T) {
	t.Parallel()

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.Anything).Return(offPlanResult(), nil)
	pm := newPermissionMiddleware(engine)

	w, handlerCalled := serveGuarded(
		t,
		pm.RequirePermission(permission.ResourceGeneralLedgerAccount.String(), permission.OpRead),
	)

	assert.False(t, handlerCalled)
	assertPlanRestricted(t, w)
	engine.AssertExpectations(t)
}

func TestRequirePermission_OrdinaryDenialStaysForbidden(t *testing.T) {
	t.Parallel()

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.Anything).
		Return(&permissionports.PermissionCheckResult{Reason: "no_permission"}, nil)
	pm := newPermissionMiddleware(engine)

	w, handlerCalled := serveGuarded(t, pm.RequirePermission("shipment", permission.OpRead))

	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, string(errortypes.ErrForbidden), decodeProblem(t, w).Code)
}

func TestRequireAllPermissions_OffPlanResourceIsPlanRestricted(t *testing.T) {
	t.Parallel()

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.MatchedBy(
		func(req *permissionports.PermissionCheckRequest) bool {
			return req.Resource == "shipment"
		},
	)).Return(&permissionports.PermissionCheckResult{Allowed: true}, nil)
	engine.On("Check", mock.Anything, mock.MatchedBy(
		func(req *permissionports.PermissionCheckRequest) bool {
			return req.Resource == permission.ResourceGeneralLedgerAccount.String()
		},
	)).Return(offPlanResult(), nil)
	pm := newPermissionMiddleware(engine)

	w, handlerCalled := serveGuarded(t, pm.RequireAllPermissions(
		PermissionCheck{Resource: "shipment", Operation: permission.OpRead},
		PermissionCheck{
			Resource:  permission.ResourceGeneralLedgerAccount.String(),
			Operation: permission.OpRead,
		},
	))

	assert.False(t, handlerCalled)
	assertPlanRestricted(t, w)
}

func TestRequireAnyPermission_EveryAlternativeOffPlanIsPlanRestricted(t *testing.T) {
	t.Parallel()

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.Anything).Return(offPlanResult(), nil)
	pm := newPermissionMiddleware(engine)

	w, handlerCalled := serveGuarded(t, pm.RequireAnyPermission(
		PermissionCheck{
			Resource:  permission.ResourceGeneralLedgerAccount.String(),
			Operation: permission.OpRead,
		},
		PermissionCheck{
			Resource:  permission.ResourceGeneralLedgerAccount.String(),
			Operation: permission.OpUpdate,
		},
	))

	assert.False(t, handlerCalled)
	assertPlanRestricted(t, w)
}

func TestRequireAnyPermission_APermissionDenialAmongThemStaysForbidden(t *testing.T) {
	t.Parallel()

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.MatchedBy(
		func(req *permissionports.PermissionCheckRequest) bool {
			return req.Resource == "shipment"
		},
	)).Return(&permissionports.PermissionCheckResult{Reason: "no_permission"}, nil)
	engine.On("Check", mock.Anything, mock.MatchedBy(
		func(req *permissionports.PermissionCheckRequest) bool {
			return req.Resource == permission.ResourceGeneralLedgerAccount.String()
		},
	)).Return(offPlanResult(), nil)
	pm := newPermissionMiddleware(engine)

	w, handlerCalled := serveGuarded(t, pm.RequireAnyPermission(
		PermissionCheck{
			Resource:  permission.ResourceGeneralLedgerAccount.String(),
			Operation: permission.OpRead,
		},
		PermissionCheck{Resource: "shipment", Operation: permission.OpRead},
	))

	assert.False(t, handlerCalled)
	assert.Equal(t, string(errortypes.ErrForbidden), decodeProblem(t, w).Code)
}
