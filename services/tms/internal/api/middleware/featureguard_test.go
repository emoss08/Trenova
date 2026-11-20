package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type failingGate struct{}

func (failingGate) Enabled(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, errors.New("plan store unreachable")
}

func (r failingGate) EnabledAnywhere(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	return r.Enabled(ctx, feature)
}

func serveThroughFeatureGuard(
	t *testing.T,
	gate platformcatalogports.FeatureGate,
	off OffAnswer,
) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(recorder)

	guard := NewFeatureGuard(FeatureGuardParams{
		Features: gate,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger:    zap.NewNop(),
			AppConfig: &appconfig.Config{},
		}),
		Logger: zap.NewNop(),
	})

	reached := false
	engine.GET(
		"/links/:token/",
		guard.RequireFeature(platformcatalog.FeatureCustomerTracking, off),
		func(c *gin.Context) {
			reached = true
			c.Status(http.StatusOK)
		},
	)

	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/links/abc/", http.NoBody))
	return recorder, reached
}

func TestFeatureGuardAnswersAsTheRouteDoesForAnUnknownTokenWhenTheFeatureIsOff(t *testing.T) {
	t.Parallel()

	recorder, reached := serveThroughFeatureGuard(
		t,
		plantest.Without(platformcatalog.FeatureCustomerTracking),
		AnswerError(func() error {
			return errortypes.NewValidationError(
				"token",
				errortypes.ErrInvalid,
				"This tracking link is no longer valid",
			)
		}),
	)

	assert.False(t, reached, "the handler is not reached")
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "This tracking link is no longer valid")
	assert.NotContains(t, recorder.Body.String(), "Your Plan", "the plan is not named")
}

func TestFeatureGuardAnswersAStatusAloneForAWebhook(t *testing.T) {
	t.Parallel()

	recorder, reached := serveThroughFeatureGuard(
		t,
		plantest.BaseOnly(),
		AnswerStatus(http.StatusNotFound),
	)

	assert.False(t, reached)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Empty(t, recorder.Body.String())
}

func TestFeatureGuardCanAnswerPlanRestricted(t *testing.T) {
	t.Parallel()

	recorder, reached := serveThroughFeatureGuard(
		t,
		plantest.BaseOnly(),
		AnswerPlanRestricted(platformcatalog.FeatureCustomerTracking),
	)

	assert.False(t, reached)
	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Not Available on Your Plan")
}

func TestFeatureGuardPassesWhenTheFeatureIsOn(t *testing.T) {
	t.Parallel()

	recorder, reached := serveThroughFeatureGuard(
		t,
		plantest.AllFeatures(),
		AnswerStatus(http.StatusNotFound),
	)

	assert.True(t, reached)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestFeatureGuardNeverPassesOnAGateError(t *testing.T) {
	t.Parallel()

	recorder, reached := serveThroughFeatureGuard(
		t,
		failingGate{},
		AnswerStatus(http.StatusNotFound),
	)

	require.False(t, reached, "a gate error never lets the request through")
	assert.GreaterOrEqual(t, recorder.Code, http.StatusInternalServerError)
}
