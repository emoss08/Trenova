package publicconfighandler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/handlers/publicconfighandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/platformconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const selfHostedBody = `{"platformMode":"self_hosted","signupEnabled":false,` +
	`"turnstileSiteKey":"","termsUrl":"","privacyUrl":"","freePlan":{"limits":{}},` +
	`"session":{"idleTimeoutSeconds":0,"idleWarningSeconds":0,"keepAliveSeconds":0}}`

type contributorFunc func(ctx context.Context, cfg *kernelports.PublicConfig) error

func (f contributorFunc) ContributePublicConfig(
	ctx context.Context,
	cfg *kernelports.PublicConfig,
) error {
	return f(ctx, cfg)
}

func serve(
	t *testing.T,
	cfg *config.Config,
	contributors ...kernelports.PublicConfigContributor,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	handler := publicconfighandler.New(publicconfighandler.Params{
		PlatformConfig: &cfg.Platform,
		SecurityConfig: &cfg.Security,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger:    zap.NewNop(),
			AppConfig: &cfg.App,
		}),
		Contributors: contributors,
	})

	router := gin.New()
	handler.RegisterPublicRoutes(&router.RouterGroup)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/system/public-config", nil))

	return w
}

func TestSelfHostedShape(t *testing.T) {
	t.Parallel()

	for _, mode := range []platformconfig.Mode{
		"",
		platformconfig.ModeSelfHosted,
		platformconfig.ModeCommunity,
		platformconfig.ModeEnterprise,
	} {
		w := serve(t, &config.Config{Platform: platformconfig.Config{Mode: mode}})

		require.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, selfHostedBody, w.Body.String())
		assert.Equal(t, "public, max-age=60", w.Header().Get("Cache-Control"))
	}
}

func TestDevelopmentModeIsReported(t *testing.T) {
	t.Parallel()

	w := serve(
		t,
		&config.Config{Platform: platformconfig.Config{Mode: platformconfig.ModeDevelopment}},
	)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"platformMode":"development"`)
}

func TestContributorsShapeTheResponse(t *testing.T) {
	t.Parallel()

	signup := contributorFunc(func(_ context.Context, cfg *kernelports.PublicConfig) error {
		cfg.SignupEnabled = true
		cfg.TurnstileSiteKey = "site-key"
		cfg.TermsURL = "https://example.test/terms"
		cfg.PrivacyURL = "https://example.test/privacy"
		return nil
	})
	plan := contributorFunc(func(_ context.Context, cfg *kernelports.PublicConfig) error {
		cfg.FreePlan.Limits["shipments.total"] = 12
		return nil
	})

	w := serve(
		t,
		&config.Config{Platform: platformconfig.Config{Mode: platformconfig.ModeCloud}},
		signup,
		plan,
	)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"platformMode":"cloud","signupEnabled":true,"turnstileSiteKey":"site-key",`+
		`"termsUrl":"https://example.test/terms","privacyUrl":"https://example.test/privacy",`+
		`"freePlan":{"limits":{"shipments.total":12}},`+
		`"session":{"idleTimeoutSeconds":0,"idleWarningSeconds":0,"keepAliveSeconds":0}}`,
		w.Body.String(),
	)
}

func TestAContributorThatClearsTheLimitsStillAnswersAnObject(t *testing.T) {
	t.Parallel()

	clear := contributorFunc(func(_ context.Context, cfg *kernelports.PublicConfig) error {
		cfg.FreePlan.Limits = nil
		return nil
	})

	w := serve(t, &config.Config{}, clear)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, selfHostedBody, w.Body.String())
}

func TestAContributorErrorIsReported(t *testing.T) {
	t.Parallel()

	failing := contributorFunc(func(context.Context, *kernelports.PublicConfig) error {
		return errors.New("catalog unavailable")
	})

	w := serve(t, &config.Config{}, failing)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, w.Header().Get("Cache-Control"))
}

func TestAsContributorJoinsTheGroup(t *testing.T) {
	t.Parallel()

	contributor := contributorFunc(
		func(context.Context, *kernelports.PublicConfig) error { return nil },
	)

	var got []kernelports.PublicConfigContributor
	app := fx.New(
		fx.NopLogger,
		fx.Provide(
			publicconfighandler.AsContributor(func() contributorFunc { return contributor }),
		),
		fx.Invoke(fx.Annotate(
			func(contributors []kernelports.PublicConfigContributor) { got = contributors },
			fx.ParamTags(`group:"`+publicconfighandler.ContributorsGroup+`"`),
		)),
	)
	require.NoError(t, app.Err())
	assert.Len(t, got, 1)
}

func TestSessionIdlePolicyIsPublished(t *testing.T) {
	t.Parallel()

	w := serve(t, &config.Config{
		Security: securityconfig.Config{
			Session: securityconfig.SessionConfig{
				MaxAge:      12 * time.Hour,
				IdleTimeout: 30 * time.Minute,
				IdleWarning: 2 * time.Minute,
			},
		},
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(
		t,
		w.Body.String(),
		`"session":{"idleTimeoutSeconds":1800,"idleWarningSeconds":120,"keepAliveSeconds":60}`,
	)
}
