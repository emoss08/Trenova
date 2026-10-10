package weatheralerthandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/weatheralerthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/operations/ports/weatheralertports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type weatherAlertServiceStub struct {
	collection *weatheralertports.WeatherAlertFeatureCollection
	detail     *weatheralertports.WeatherAlertDetail
}

func (s *weatherAlertServiceStub) PollNWSAlerts(
	context.Context,
	weatheralertports.WeatherAlertPollHeartbeat,
) (*weatheralertports.PollNWSAlertsResult, error) {
	return &weatheralertports.PollNWSAlertsResult{}, nil
}

func (s *weatherAlertServiceStub) ListWeatherAlertTenants(
	context.Context,
	int,
) ([]pagination.TenantInfo, error) {
	return nil, nil
}

func (s *weatherAlertServiceStub) PollNWSAlertsForTenant(
	context.Context,
	pagination.TenantInfo,
) error {
	return nil
}

func (s *weatherAlertServiceStub) ExpireStaleWeatherAlerts(context.Context) error {
	return nil
}

func (s *weatherAlertServiceStub) GetActiveAlerts(
	context.Context,
	pagination.TenantInfo,
) (*weatheralertports.WeatherAlertFeatureCollection, error) {
	return s.collection, nil
}

func (s *weatherAlertServiceStub) GetAlertDetail(
	context.Context,
	*weatheralertports.GetWeatherAlertDetailRequest,
) (*weatheralertports.WeatherAlertDetail, error) {
	return s.detail, nil
}

func setupHandler(
	t *testing.T,
	service weatheralertports.WeatherAlertService,
) *weatheralerthandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: cfg})
	return weatheralerthandler.New(
		weatheralerthandler.Params{Service: service, ErrorHandler: errorHandler},
	)
}

func TestListWeatherAlerts(t *testing.T) {
	t.Parallel()

	alertID := pulid.MustNew("walt_")
	handler := setupHandler(t, &weatherAlertServiceStub{
		collection: &weatheralertports.WeatherAlertFeatureCollection{
			Type: "FeatureCollection",
			Features: []*weatheralertports.WeatherAlertFeature{
				{
					Type: "Feature",
					ID:   alertID,
					Geometry: weatheralertports.WeatherAlertGeometry{
						"type": "Polygon",
					},
					Properties: weatheralertports.WeatherAlertFeatureProperties{
						ID:    alertID,
						NWSID: "urn:oid:1",
						Event: "Flood Warning",
					},
				},
			},
		},
	})

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/weather-alerts/").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp weatheralertports.WeatherAlertFeatureCollection
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp.Features, 1)
	assert.Equal(t, alertID, resp.Features[0].ID)
}

func TestGetWeatherAlertDetail(t *testing.T) {
	t.Parallel()

	alertID := pulid.MustNew("walt_")
	handler := setupHandler(t, &weatherAlertServiceStub{
		detail: &weatheralertports.WeatherAlertDetail{
			Feature: &weatheralertports.WeatherAlertFeature{
				Type: "Feature",
				ID:   alertID,
				Geometry: weatheralertports.WeatherAlertGeometry{
					"type": "Polygon",
				},
				Properties: weatheralertports.WeatherAlertFeatureProperties{
					ID:    alertID,
					NWSID: "urn:oid:1",
					Event: "Flood Warning",
				},
			},
		},
	})

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/weather-alerts/" + alertID.String() + "/").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp weatheralertports.WeatherAlertDetail
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.NotNil(t, resp.Feature)
	assert.Equal(t, alertID, resp.Feature.ID)
}
