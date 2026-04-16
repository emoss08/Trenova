package weatheralerthandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/weatheralerthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type weatherAlertServiceStub struct {
	collection *serviceports.WeatherAlertFeatureCollection
	detail     *serviceports.WeatherAlertDetail
}

func (s *weatherAlertServiceStub) PollNWSAlerts(context.Context) error {
	return nil
}

func (s *weatherAlertServiceStub) GetActiveAlerts(context.Context, pagination.TenantInfo) (*serviceports.WeatherAlertFeatureCollection, error) {
	return s.collection, nil
}

func (s *weatherAlertServiceStub) GetAlertDetail(context.Context, *serviceports.GetWeatherAlertDetailRequest) (*serviceports.WeatherAlertDetail, error) {
	return s.detail, nil
}

func setupHandler(t *testing.T, service serviceports.WeatherAlertService) *weatheralerthandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: cfg})
	return weatheralerthandler.New(weatheralerthandler.Params{Service: service, ErrorHandler: errorHandler})
}

func TestListWeatherAlerts(t *testing.T) {
	t.Parallel()

	alertID := pulid.MustNew("walt_")
	handler := setupHandler(t, &weatherAlertServiceStub{
		collection: &serviceports.WeatherAlertFeatureCollection{
			Type: "FeatureCollection",
			Features: []*serviceports.WeatherAlertFeature{
				{
					Type: "Feature",
					ID:   alertID,
					Geometry: serviceports.WeatherAlertGeometry{
						"type": "Polygon",
					},
					Properties: serviceports.WeatherAlertFeatureProperties{
						ID:    alertID,
						NWSID: "urn:oid:1",
						Event: "Flood Warning",
					},
				},
			},
		},
	})

	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodGet).WithPath("/api/v1/weather-alerts/").WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp serviceports.WeatherAlertFeatureCollection
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp.Features, 1)
	assert.Equal(t, alertID, resp.Features[0].ID)
}

func TestGetWeatherAlertDetail(t *testing.T) {
	t.Parallel()

	alertID := pulid.MustNew("walt_")
	handler := setupHandler(t, &weatherAlertServiceStub{
		detail: &serviceports.WeatherAlertDetail{
			Feature: &serviceports.WeatherAlertFeature{
				Type: "Feature",
				ID:   alertID,
				Geometry: serviceports.WeatherAlertGeometry{
					"type": "Polygon",
				},
				Properties: serviceports.WeatherAlertFeatureProperties{
					ID:    alertID,
					NWSID: "urn:oid:1",
					Event: "Flood Warning",
				},
			},
		},
	})

	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodGet).WithPath("/api/v1/weather-alerts/" + alertID.String() + "/").WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp serviceports.WeatherAlertDetail
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.NotNil(t, resp.Feature)
	assert.Equal(t, alertID, resp.Feature.ID)
}
