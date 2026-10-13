package carriertrackingpublichandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	liveToken     = "live-token"
	throttleBurst = 5
)

type fakeService struct {
	checkcallports.CheckCallService
	posted []*checkcallports.PublicCheckCallInput
}

func (f *fakeService) PreviewByToken(
	_ context.Context,
	token string,
) (*checkcallports.PublicTrackingView, error) {
	if token != liveToken {
		return nil, errortypes.NewValidationError(
			"token",
			errortypes.ErrInvalid,
			"This tracking link is no longer valid",
		)
	}
	return &checkcallports.PublicTrackingView{
		CarrierName:       "Redline Freight",
		ShipmentProNumber: "PRO-77",
		Stops: []checkcallports.PublicTrackingStop{
			{ID: pulid.MustNew("stp_"), Type: "Pickup", City: "Dallas"},
		},
	}, nil
}

func (f *fakeService) PostByToken(
	_ context.Context,
	token string,
	input *checkcallports.PublicCheckCallInput,
) (*checkcallports.PublicCheckCallResult, error) {
	if token != liveToken {
		return nil, errortypes.NewValidationError(
			"token",
			errortypes.ErrInvalid,
			"This tracking link is no longer valid",
		)
	}
	f.posted = append(f.posted, input)
	return &checkcallports.PublicCheckCallResult{Outcome: trackingevent.OutcomeApplied}, nil
}

func newRouter(service *fakeService) *gin.Engine {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			RateLimit: config.RateLimitConfig{
				Enabled: true,
				PublicToken: config.RateLimitScopeConfig{
					RequestsPerMinute: 10,
					BurstSize:         throttleBurst,
				},
			},
		},
	}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: cfg,
	})
	handler := New(Params{
		Service:      service,
		ErrorHandler: errorHandler,
		RateLimiter: middleware.NewRateLimiter(middleware.RateLimiterParams{
			Config:       cfg,
			Store:        ratelimit.NewMemoryStore(ratelimit.MemoryStoreOptions{}),
			ErrorHandler: errorHandler,
			Logger:       zap.NewNop(),
		}),
	})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterPublicRoutes(router.Group("/api/v1"))
	return router
}

func TestPreviewServesTheLoadView(t *testing.T) {
	t.Parallel()

	router := newRouter(&fakeService{})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/api/v1/carrier-tracking-links/"+liveToken+"/", nil,
	))

	require.Equal(t, http.StatusOK, rec.Code)
	var view checkcallports.PublicTrackingView
	require.NoError(t, sonic.Unmarshal(rec.Body.Bytes(), &view))
	assert.Equal(t, "Redline Freight", view.CarrierName)
	assert.Equal(t, "PRO-77", view.ShipmentProNumber)
	require.Len(t, view.Stops, 1)
	assert.Equal(t, "Dallas", view.Stops[0].City)
}

func TestUnknownTokenGetsTheVagueAnswer(t *testing.T) {
	t.Parallel()

	router := newRouter(&fakeService{})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/api/v1/carrier-tracking-links/dead/", nil,
	))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "This tracking link is no longer valid")
}

func TestPostPassesTheCheckCallThrough(t *testing.T) {
	t.Parallel()

	service := &fakeService{}
	router := newRouter(service)
	stopID := pulid.MustNew("stp_")
	body := `{"kind":"Arrival","stopId":"` + stopID.String() + `","reportedAt":1800000000,` +
		`"latitude":32.78,"longitude":-96.8,"delayed":true,"delayReason":"Dock full",` +
		`"contactName":"Sam","clientEventId":"tap-9"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/carrier-tracking-links/"+liveToken+"/check-calls/",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"outcome":"Applied"`)
	require.Len(t, service.posted, 1)
	posted := service.posted[0]
	assert.Equal(t, checkcall.KindArrival, posted.Details.Kind)
	assert.Equal(t, stopID, posted.Details.StopID)
	assert.Equal(t, int64(1800000000), *posted.Details.ReportedAt)
	assert.InDelta(t, 32.78, *posted.Details.Latitude, 0.0001)
	assert.True(t, posted.Details.Delayed)
	assert.Equal(t, "Dock full", posted.Details.DelayReason)
	assert.Equal(t, "Sam", posted.Details.ContactName)
	assert.Equal(t, "tap-9", posted.ClientEventID)
}

func TestPostRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	service := &fakeService{}
	router := newRouter(service)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/carrier-tracking-links/"+liveToken+"/check-calls/",
		strings.NewReader("{"),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, service.posted)
}

func TestPerTokenThrottleCapsBursts(t *testing.T) {
	t.Parallel()

	router := newRouter(&fakeService{})
	var last *httptest.ResponseRecorder
	for range throttleBurst + 1 {
		last = httptest.NewRecorder()
		router.ServeHTTP(last, httptest.NewRequest(
			http.MethodGet, "/api/v1/carrier-tracking-links/"+liveToken+"/", nil,
		))
	}

	require.Equal(t, http.StatusTooManyRequests, last.Code)
	assert.Equal(t, "publicToken", last.Header().Get("X-RateLimit-Scope"))

	other := httptest.NewRecorder()
	router.ServeHTTP(other, httptest.NewRequest(
		http.MethodGet, "/api/v1/carrier-tracking-links/another/", nil,
	))
	assert.NotEqual(t, http.StatusTooManyRequests, other.Code)
}
