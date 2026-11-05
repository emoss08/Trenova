package publicconfighandler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/handlers/publicconfighandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/platformconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const selfHostedBody = `{"platformMode":"self_hosted",` +
	`"session":{"idleTimeoutSeconds":0,"idleWarningSeconds":0,"keepAliveSeconds":0}}`

func serve(t *testing.T, cfg *config.Config) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	handler := publicconfighandler.New(publicconfighandler.Params{
		PlatformConfig: &cfg.Platform,
		SecurityConfig: &cfg.Security,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger:    zap.NewNop(),
			AppConfig: &cfg.App,
		}),
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
