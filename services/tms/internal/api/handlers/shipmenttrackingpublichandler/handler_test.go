package shipmenttrackingpublichandler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	liveToken     = "live-token"
	throttleBurst = 5
	podBody       = "%PDF-1.7 signed"
)

type fakeService struct {
	customertrackingports.CustomerTrackingService
}

func invalid() error {
	return errortypes.NewValidationError(
		"token",
		errortypes.ErrInvalid,
		"This tracking link is no longer valid",
	)
}

func (f *fakeService) PreviewByToken(
	_ context.Context,
	token string,
) (*customertrackingports.PublicTrackingView, error) {
	if token != liveToken {
		return nil, invalid()
	}
	eta := int64(1_800_007_200)
	return &customertrackingports.PublicTrackingView{
		CompanyName:       "Trenova Logistics",
		ShipmentProNumber: "PRO-77",
		Fields: []customertracking.Field{
			customertracking.FieldStatus,
			customertracking.FieldEta,
		},
		Status: customertrackingports.PublicStatusDelayed,
		Stops: []customertrackingports.PublicStop{
			{Sequence: 1, Type: "Pickup", City: "Dallas", StateCode: "TX"},
			{
				Sequence:         2,
				Type:             "Delivery",
				City:             "Memphis",
				StateCode:        "TN",
				EstimatedArrival: &eta,
			},
		},
	}, nil
}

func (f *fakeService) DownloadPodByToken(
	_ context.Context,
	token string,
) (*customertrackingports.DownloadResult, error) {
	if token != liveToken {
		return nil, invalid()
	}
	return &customertrackingports.DownloadResult{
		FileName:           "pod.pdf",
		ContentType:        "application/pdf",
		ContentLength:      int64(len(podBody)),
		ContentDisposition: `attachment; filename="pod.pdf"`,
		Body:               io.NopCloser(strings.NewReader(podBody)),
	}, nil
}

func newRouter() *gin.Engine {
	cfg := &config.Config{
		Security: securityconfig.Config{
			RateLimit: securityconfig.RateLimitConfig{
				Enabled: true,
				PublicToken: securityconfig.RateLimitScopeConfig{
					RequestsPerMinute: 10,
					BurstSize:         throttleBurst,
				},
			},
		},
	}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    zap.NewNop(),
		AppConfig: &cfg.App,
	})
	handler := New(Params{
		Service:      &fakeService{},
		ErrorHandler: errorHandler,
		RateLimiter: middleware.NewRateLimiter(middleware.RateLimiterParams{
			SecurityConfig: &cfg.Security,
			Store:          ratelimit.NewMemoryStore(ratelimit.MemoryStoreOptions{}),
			ErrorHandler:   errorHandler,
			Logger:         zap.NewNop(),
		}),
	})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterPublicRoutes(router.Group("/api/v1"))
	return router
}

func get(router *gin.Engine, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestPreviewServesOnlyWhatTheServiceReturns(t *testing.T) {
	t.Parallel()

	rec := get(newRouter(), "/api/v1/shipment-tracking-links/"+liveToken+"/")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	var view customertrackingports.PublicTrackingView
	require.NoError(t, sonic.Unmarshal(rec.Body.Bytes(), &view))
	assert.Equal(t, "PRO-77", view.ShipmentProNumber)
	assert.Equal(t, customertrackingports.PublicStatusDelayed, view.Status)
	require.Len(t, view.Stops, 2)
	assert.Equal(t, int64(1_800_007_200), *view.Stops[1].EstimatedArrival)
	assert.NotContains(t, rec.Body.String(), "carrierName")
	assert.NotContains(t, rec.Body.String(), "proofOfDelivery")
}

func TestUnknownTokenGetsTheVagueAnswer(t *testing.T) {
	t.Parallel()

	rec := get(newRouter(), "/api/v1/shipment-tracking-links/dead/")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "This tracking link is no longer valid")
}

func TestProofOfDeliveryDownloadsAsAnAttachment(t *testing.T) {
	t.Parallel()

	rec := get(newRouter(), "/api/v1/shipment-tracking-links/"+liveToken+"/proof-of-delivery/")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.Equal(t, `attachment; filename="pod.pdf"`, rec.Header().Get("Content-Disposition"))
	assert.Equal(t, podBody, rec.Body.String())
}

func TestPerTokenThrottleCapsBursts(t *testing.T) {
	t.Parallel()

	router := newRouter()
	var last *httptest.ResponseRecorder
	for range throttleBurst + 1 {
		last = get(router, "/api/v1/shipment-tracking-links/"+liveToken+"/")
	}

	require.Equal(t, http.StatusTooManyRequests, last.Code)
	assert.Equal(t, "publicToken", last.Header().Get("X-RateLimit-Scope"))
}
