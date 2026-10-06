package onboardinghandler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type grantEngine struct {
	*permtest.Engine
	allowed map[permission.Operation]bool
}

func (e grantEngine) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	return &services.PermissionCheckResult{Allowed: e.allowed[req.Operation]}, nil
}

type harness struct {
	svc    *mocks.MockOnboardingService
	router *gin.Engine
	userID pulid.ID
	orgID  pulid.ID
	buID   pulid.ID
}

func newHarness(t *testing.T, ops ...permission.Operation) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	eh := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: zap.NewNop(), Config: cfg})
	allowed := make(map[permission.Operation]bool, len(ops))
	for _, op := range ops {
		allowed[op] = true
	}
	engine := grantEngine{Engine: permtest.AllowAll(), allowed: allowed}

	h := &harness{
		svc:    mocks.NewMockOnboardingService(t),
		userID: pulid.MustNew("usr_"),
		orgID:  pulid.MustNew("org_"),
		buID:   pulid.MustNew("bu_"),
	}
	handler := New(Params{
		Service:      h.svc,
		ErrorHandler: eh,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: engine,
				ErrorHandler:     eh,
			},
		),
	})

	h.router = gin.New()
	h.router.Use(func(c *gin.Context) {
		authctx.SetAuthContext(c, h.userID, h.buID, h.orgID)
		c.Next()
	})
	handler.RegisterRoutes(&h.router.RouterGroup)

	return h
}

func (h *harness) do(method, path string, body any) *httptest.ResponseRecorder {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

func TestGetOnboarding(t *testing.T) {
	t.Parallel()

	h := newHarness(t, permission.OpRead)
	h.svc.EXPECT().Get(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, info pagination.TenantInfo) (*services.OnboardingState, error) {
			assert.Equal(t, h.orgID, info.OrgID)
			assert.Equal(t, h.buID, info.BuID)
			assert.Equal(t, h.userID, info.UserID)
			return &services.OnboardingState{
				Required: true,
				Status:   onboarding.StatusPending,
				Organization: &services.OnboardingOrganization{
					Name:     "Acme Freight",
					Timezone: "America/New_York",
				},
			}, nil
		},
	)

	w := h.do(http.MethodGet, "/onboarding/", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["required"])
	assert.Equal(t, "pending", body["status"])
	assert.Nil(t, body["completedAt"])
	assert.Equal(t, false, body["sampleDataLoaded"])
	org := body["organization"].(map[string]any)
	assert.Equal(t, "Acme Freight", org["name"])
	assert.Equal(t, "", org["scacCode"])
}

func TestCompleteOnboarding(t *testing.T) {
	t.Parallel()

	h := newHarness(t, permission.OpRead, permission.OpUpdate)
	stateID := pulid.MustNew("us_")
	completedAt := int64(1_790_000_000)
	h.svc.EXPECT().Complete(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, req *services.CompleteOnboardingRequest) (*services.OnboardingState, error) {
			assert.Equal(t, h.orgID, req.TenantInfo.OrgID)
			assert.Equal(t, h.userID, req.TenantInfo.UserID)
			assert.Equal(t, h.userID, req.Actor.UserID)
			assert.Equal(t, "Acme Freight", req.Organization.Name)
			assert.Equal(t, stateID, req.Organization.StateID)
			assert.Equal(t, "ACME", req.Organization.ScacCode)
			assert.Empty(t, req.Organization.DOTNumber)
			assert.Equal(t, tenant.OperationTypeBoth, req.OperationType)
			assert.True(t, req.LoadSampleData)
			return &services.OnboardingState{
				Status:           onboarding.StatusCompleted,
				OperationType:    tenant.OperationTypeBoth,
				SampleDataLoaded: true,
				CompletedAt:      &completedAt,
			}, nil
		},
	)

	w := h.do(http.MethodPost, "/onboarding/complete/", map[string]any{
		"organization": map[string]any{
			"name":         "Acme Freight",
			"timezone":     "America/Chicago",
			"addressLine1": "1 Main St",
			"city":         "Dallas",
			"stateId":      stateID.String(),
			"postalCode":   "75201",
			"scacCode":     "ACME",
		},
		"operationType":  "both",
		"loadSampleData": true,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "completed", body["status"])
	assert.Equal(t, "both", body["operationType"])
	assert.InDelta(t, completedAt, body["completedAt"], 0)
}

func TestCompleteOnboardingNeedsOrganizationUpdate(t *testing.T) {
	t.Parallel()

	h := newHarness(t, permission.OpRead)

	w := h.do(http.MethodPost, "/onboarding/complete/", map[string]any{"operationType": "asset"})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCompleteOnboardingReturnsNestedFieldErrors(t *testing.T) {
	t.Parallel()

	h := newHarness(t, permission.OpRead, permission.OpUpdate)
	multiErr := errortypes.NewMultiError()
	multiErr.WithPrefix("organization").
		Add("dotNumber", errortypes.ErrInvalid, "DOT number must be numeric")
	h.svc.EXPECT().Complete(mock.Anything, mock.Anything).Return(nil, multiErr)

	w := h.do(http.MethodPost, "/onboarding/complete/", map[string]any{"operationType": "asset"})
	require.Equal(t, http.StatusBadRequest, w.Code)

	var problem helpers.ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &problem))
	require.Len(t, problem.Errors, 1)
	assert.Equal(t, "organization.dotNumber", problem.Errors[0].Field)
}
