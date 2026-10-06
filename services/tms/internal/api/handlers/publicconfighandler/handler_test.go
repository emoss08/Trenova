package publicconfighandler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/publicconfighandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const selfHostedBody = `{"platformMode":"self_hosted","signupEnabled":false,` +
	`"turnstileSiteKey":"","termsUrl":"","privacyUrl":"","freePlan":{"limits":{}}}`

type contributorFunc func(ctx context.Context, cfg *services.PublicConfig) error

func (f contributorFunc) ContributePublicConfig(ctx context.Context, cfg *services.PublicConfig) error {
	return f(ctx, cfg)
}

func serve(
	t *testing.T,
	cfg *config.Config,
	contributors ...services.PublicConfigContributor,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	handler := publicconfighandler.New(publicconfighandler.Params{
		Config: cfg,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: cfg,
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

	for _, mode := range []config.PlatformMode{
		"",
		config.PlatformModeSelfHosted,
		config.PlatformModeCommunity,
		config.PlatformModeEnterprise,
	} {
		w := serve(t, &config.Config{Platform: config.PlatformConfig{Mode: mode}})

		require.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, selfHostedBody, w.Body.String())
		assert.Equal(t, "public, max-age=60", w.Header().Get("Cache-Control"))
	}
}

func TestDevelopmentModeIsReported(t *testing.T) {
	t.Parallel()

	w := serve(t, &config.Config{Platform: config.PlatformConfig{Mode: config.PlatformModeDevelopment}})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"platformMode":"development"`)
}

func TestContributorsShapeTheResponse(t *testing.T) {
	t.Parallel()

	signup := contributorFunc(func(_ context.Context, cfg *services.PublicConfig) error {
		cfg.SignupEnabled = true
		cfg.TurnstileSiteKey = "site-key"
		cfg.TermsURL = "https://example.test/terms"
		cfg.PrivacyURL = "https://example.test/privacy"
		return nil
	})
	plan := contributorFunc(func(_ context.Context, cfg *services.PublicConfig) error {
		cfg.FreePlan.Limits["shipments.total"] = 12
		return nil
	})

	w := serve(t, &config.Config{Platform: config.PlatformConfig{Mode: config.PlatformModeCloud}}, signup, plan)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"platformMode":"cloud","signupEnabled":true,"turnstileSiteKey":"site-key",`+
		`"termsUrl":"https://example.test/terms","privacyUrl":"https://example.test/privacy",`+
		`"freePlan":{"limits":{"shipments.total":12}}}`, w.Body.String())
}

func TestAContributorThatClearsTheLimitsStillAnswersAnObject(t *testing.T) {
	t.Parallel()

	clear := contributorFunc(func(_ context.Context, cfg *services.PublicConfig) error {
		cfg.FreePlan.Limits = nil
		return nil
	})

	w := serve(t, &config.Config{}, clear)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, selfHostedBody, w.Body.String())
}

func TestAContributorErrorIsReported(t *testing.T) {
	t.Parallel()

	failing := contributorFunc(func(context.Context, *services.PublicConfig) error {
		return errors.New("catalog unavailable")
	})

	w := serve(t, &config.Config{}, failing)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, w.Header().Get("Cache-Control"))
}

func TestAsContributorJoinsTheGroup(t *testing.T) {
	t.Parallel()

	contributor := contributorFunc(func(context.Context, *services.PublicConfig) error { return nil })

	var got []services.PublicConfigContributor
	app := fx.New(
		fx.NopLogger,
		fx.Provide(publicconfighandler.AsContributor(func() contributorFunc { return contributor })),
		fx.Invoke(fx.Annotate(
			func(contributors []services.PublicConfigContributor) { got = contributors },
			fx.ParamTags(`group:"`+publicconfighandler.ContributorsGroup+`"`),
		)),
	)
	require.NoError(t, app.Err())
	assert.Len(t, got, 1)
}
