package cloudsignuphandler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/csrf"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testConfig(mode config.PlatformMode) *config.Config {
	return &config.Config{
		App: config.AppConfig{Debug: true},
		Platform: config.PlatformConfig{
			Mode: mode,
			Cloud: config.PlatformCloudConfig{
				Signup: config.CloudSignupConfig{
					Enabled:      true,
					PerIPPerHour: 3,
					TermsURL:     "https://trenova.test/terms",
				},
				Turnstile: config.CloudTurnstileConfig{Enabled: true, SiteKey: "site-key"},
			},
		},
		Security: config.SecurityConfig{
			Session: config.SessionConfig{
				Name:     "session_id",
				Secret:   "test-session-secret",
				Path:     "/",
				HTTPOnly: true,
				SameSite: "lax",
			},
			CSRF: config.CSRFConfig{HeaderName: "X-CSRF-Token"},
		},
	}
}

type fakeStore struct {
	t         *testing.T
	decisions []repositories.RateLimitDecision
	requests  []repositories.RateLimitRequest
}

func (f *fakeStore) Check(
	_ context.Context,
	requests []repositories.RateLimitRequest,
) ([]repositories.RateLimitDecision, error) {
	f.requests = append(f.requests, requests...)
	if f.decisions == nil {
		f.t.Fatalf("unexpected rate limit check")
	}
	return f.decisions, nil
}

type harness struct {
	svc    *mocks.MockCloudSignupService
	store  *fakeStore
	router *gin.Engine
}

func newHarness(t *testing.T, mode config.PlatformMode) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := testConfig(mode)
	catalog, err := platformplan.NewCatalog(nil)
	require.NoError(t, err)

	h := &harness{
		svc:   mocks.NewMockCloudSignupService(t),
		store: &fakeStore{t: t},
	}
	handler := New(Params{
		Service:        h.svc,
		Catalog:        catalog,
		RateLimitStore: h.store,
		Config:         cfg,
		ErrorHandler:   helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: zap.NewNop(), Config: cfg}),
		Logger:         zap.NewNop(),
	})
	h.router = gin.New()
	handler.RegisterPublicRoutes(&h.router.RouterGroup)

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

func (h *harness) allow(t *testing.T) {
	t.Helper()
	h.store.decisions = []repositories.RateLimitDecision{{Allowed: true}}
	t.Cleanup(func() {
		require.Len(t, h.store.requests, 1)
		req := h.store.requests[0]
		assert.Equal(t, 3, req.Policy.Rate)
		assert.Equal(t, 3, req.Policy.Burst)
		assert.Equal(t, time.Hour, req.Policy.Period)
		assert.Equal(t, signupLimitPrefix+"192.0.2.1", req.Key)
	})
}

func TestPublicConfigInCloudMode(t *testing.T) {
	t.Parallel()

	h := newHarness(t, config.PlatformModeCloud)
	h.svc.EXPECT().Enabled().Return(true)

	w := h.do(http.MethodGet, "/system/public-config", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var body PublicConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "cloud", body.PlatformMode)
	assert.True(t, body.SignupEnabled)
	assert.Equal(t, "site-key", body.TurnstileSiteKey)
	assert.Equal(t, "https://trenova.test/terms", body.TermsURL)
	assert.Equal(t, config.DefaultCloudSignupPrivacyURL, body.PrivacyURL)
	assert.Equal(t, int64(12), body.FreePlan.Limits["shipments.total"])
	assert.Equal(t, int64(1), body.FreePlan.Limits["users.seats"])
}

func TestPublicConfigOutsideCloud(t *testing.T) {
	t.Parallel()

	h := newHarness(t, config.PlatformModeSelfHosted)
	h.svc.EXPECT().Enabled().Return(false)

	w := h.do(http.MethodGet, "/system/public-config", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "self_hosted", body["platformMode"])
	assert.Equal(t, false, body["signupEnabled"])
	assert.Empty(t, body["turnstileSiteKey"])
	assert.Equal(t, map[string]any{"limits": map[string]any{}}, body["freePlan"])
}

func TestSignupRoutesAreHiddenWhenDisabled(t *testing.T) {
	t.Parallel()

	h := newHarness(t, config.PlatformModeSelfHosted)
	h.svc.EXPECT().Enabled().Return(false)

	for _, path := range []string{"/cloud/signups", "/cloud/signups/resend", "/cloud/signups/verify"} {
		w := h.do(http.MethodPost, path, map[string]string{})
		assert.Equal(t, http.StatusNotFound, w.Code, path)
	}
}

func TestSignupAnswersAccepted(t *testing.T) {
	t.Parallel()

	h := newHarness(t, config.PlatformModeCloud)
	h.svc.EXPECT().Enabled().Return(true)
	h.allow(t)
	h.svc.EXPECT().Signup(mock.Anything, &services.CloudSignupRequest{
		Name:           "Dana",
		EmailAddress:   "dana@example.com",
		Password:       "long-enough-password",
		CompanyName:    "Acme",
		AcceptTerms:    true,
		TurnstileToken: "ts",
	}).Return(&services.CloudSignupAccepted{Status: "pending"}, nil)

	w := h.do(http.MethodPost, "/cloud/signups", map[string]any{
		"name":           "Dana",
		"emailAddress":   "dana@example.com",
		"password":       "long-enough-password",
		"companyName":    "Acme",
		"acceptTerms":    true,
		"turnstileToken": "ts",
		"website":        "",
	})
	require.Equal(t, http.StatusAccepted, w.Code)
	assert.JSONEq(t, `{"status":"pending"}`, w.Body.String())
}

func TestSignupFieldErrorsCarryTheField(t *testing.T) {
	t.Parallel()

	h := newHarness(t, config.PlatformModeCloud)
	h.svc.EXPECT().Enabled().Return(true)
	h.allow(t)
	multiErr := errortypes.NewMultiError()
	multiErr.Add("password", errortypes.ErrInvalid, "Password must be at least 12 characters")
	h.svc.EXPECT().Signup(mock.Anything, mock.Anything).Return(nil, multiErr)

	w := h.do(http.MethodPost, "/cloud/signups", map[string]any{"password": "x"})
	require.Equal(t, http.StatusBadRequest, w.Code)

	var problem helpers.ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &problem))
	require.Len(t, problem.Errors, 1)
	assert.Equal(t, "password", problem.Errors[0].Field)
}

func TestSignupPerIPLimit(t *testing.T) {
	t.Parallel()

	h := newHarness(t, config.PlatformModeCloud)
	h.svc.EXPECT().Enabled().Return(true)
	h.store.decisions = []repositories.RateLimitDecision{{Allowed: false, RetryAfter: 20 * time.Minute}}

	w := h.do(http.MethodPost, "/cloud/signups/resend", map[string]any{"emailAddress": "a@b.co"})
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "1200", w.Header().Get("Retry-After"))
}

func TestResendAnswersAccepted(t *testing.T) {
	t.Parallel()

	h := newHarness(t, config.PlatformModeCloud)
	h.svc.EXPECT().Enabled().Return(true)
	h.allow(t)
	h.svc.EXPECT().Resend(mock.Anything, &services.CloudSignupResendRequest{
		EmailAddress:   "dana@example.com",
		TurnstileToken: "ts",
	}).Return(&services.CloudSignupAccepted{Status: "pending"}, nil)

	w := h.do(http.MethodPost, "/cloud/signups/resend", map[string]any{
		"emailAddress":   "dana@example.com",
		"turnstileToken": "ts",
	})
	require.Equal(t, http.StatusAccepted, w.Code)
}

func TestVerifySignsInLikeLogin(t *testing.T) {
	t.Parallel()

	h := newHarness(t, config.PlatformModeCloud)
	h.svc.EXPECT().Enabled().Return(true)
	sessionID := pulid.MustNew("ses_")
	h.svc.EXPECT().Verify(mock.Anything, &services.CloudSignupVerifyRequest{Token: "tok"}).Return(
		&services.LoginResponse{
			User: &tenant.User{
				ID:                    pulid.MustNew("usr_"),
				BusinessUnitID:        pulid.MustNew("bu_"),
				CurrentOrganizationID: pulid.MustNew("org_"),
			},
			SessionID:    sessionID.String(),
			SessionToken: "raw-session-token",
			ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		}, nil,
	)

	w := h.do(http.MethodPost, "/cloud/signups/verify", map[string]string{"token": "tok"})
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, csrf.Token(sessionID.String(), "test-session-secret"), body["csrfToken"])
	assert.Equal(t, sessionID.String(), body["sessionId"])
	assert.NotContains(t, w.Body.String(), "raw-session-token")

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "session_id", cookies[0].Name)
	assert.Equal(t, "raw-session-token", cookies[0].Value)
}

func TestVerifySignupsPaused(t *testing.T) {
	t.Parallel()

	h := newHarness(t, config.PlatformModeCloud)
	h.svc.EXPECT().Enabled().Return(true)
	h.svc.EXPECT().Verify(mock.Anything, mock.Anything).Return(nil, errortypes.NewPlanRestrictionError(
		"", errortypes.PlanRestrictionReasonSignupsPaused, "free_demo",
	))

	w := h.do(http.MethodPost, "/cloud/signups/verify", map[string]string{"token": "tok"})
	require.Equal(t, http.StatusForbidden, w.Code)

	var problem helpers.ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &problem))
	assert.Equal(t, errortypes.PlanRestrictionReasonSignupsPaused, problem.Params["reason"])
}
