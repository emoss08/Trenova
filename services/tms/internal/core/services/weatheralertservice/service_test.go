package weatheralertservice

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/weatheralert"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type repoStub struct {
	tenants      []pagination.TenantInfo
	activeAlerts []*weatheralert.WeatherAlert
	alert        *weatheralert.WeatherAlert
	activities   []*weatheralert.Activity
	upserted     []*weatheralert.WeatherAlert
	expireCalls  int
}

func (s *repoStub) ListTenants(context.Context) ([]pagination.TenantInfo, error) {
	return s.tenants, nil
}

func (s *repoStub) GetActiveAlerts(context.Context, pagination.TenantInfo) ([]*weatheralert.WeatherAlert, error) {
	return s.activeAlerts, nil
}

func (s *repoStub) GetByID(context.Context, repositories.GetWeatherAlertByIDRequest) (*weatheralert.WeatherAlert, error) {
	return s.alert, nil
}

func (s *repoStub) GetActivities(context.Context, repositories.GetWeatherAlertByIDRequest) ([]*weatheralert.Activity, error) {
	return s.activities, nil
}

func (s *repoStub) UpsertAlert(_ context.Context, alert *weatheralert.WeatherAlert) (*repositories.UpsertWeatherAlertResult, error) {
	s.upserted = append(s.upserted, alert)
	return &repositories.UpsertWeatherAlertResult{Alert: alert}, nil
}

func (s *repoStub) ExpireStaleAlerts(context.Context) (*repositories.ExpireWeatherAlertsResult, error) {
	s.expireCalls++
	return &repositories.ExpireWeatherAlertsResult{}, nil
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestMapAlertCategory(t *testing.T) {
	t.Parallel()

	service := New(Params{Logger: zap.NewNop(), Repo: &repoStub{}})

	assert.Equal(t, weatheralert.AlertCategoryWinterWeather, service.mapAlertCategory("Winter Storm Warning"))
	assert.Equal(t, weatheralert.AlertCategoryWindStorm, service.mapAlertCategory("High Wind Watch"))
	assert.Equal(t, weatheralert.AlertCategoryFloodWater, service.mapAlertCategory("Flash Flood Warning"))
	assert.Equal(t, weatheralert.AlertCategoryFire, service.mapAlertCategory("Red Flag Warning"))
	assert.Equal(t, weatheralert.AlertCategoryHeat, service.mapAlertCategory("Excessive Heat Warning"))
	assert.Equal(t, weatheralert.AlertCategoryTornadoSevereStorm, service.mapAlertCategory("Severe Thunderstorm Warning"))
	assert.Equal(t, weatheralert.AlertCategoryTropicalStormHurricane, service.mapAlertCategory("Tropical Storm Warning"))
	assert.Equal(t, weatheralert.AlertCategoryWindStorm, service.mapAlertCategory("Marine Weather Statement"))
	assert.Equal(t, weatheralert.AlertCategoryOther, service.mapAlertCategory("Dense Fog Advisory"))
}

func TestPollNWSAlertsFansOutAcrossTenantsAndSkipsNullGeometry(t *testing.T) {
	t.Parallel()

	repo := &repoStub{
		tenants: []pagination.TenantInfo{
			{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
			{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		},
	}
	service := New(Params{Logger: zap.NewNop(), Repo: repo})
	service.httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, nwsUserAgent, req.Header.Get("User-Agent"))
			body := `{"features":[{"geometry":{"type":"Polygon","coordinates":[[[-97,32],[-96,32],[-96,33],[-97,33],[-97,32]]]},"properties":{"id":"urn:oid:1","event":"Winter Storm Warning","severity":"Severe","messageType":"Alert","effective":"2026-04-15T12:00:00Z","expires":"2026-04-15T18:00:00Z"}},{"geometry":null,"properties":{"id":"urn:oid:2","event":"Flood Warning","messageType":"Alert"}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	require.NoError(t, service.PollNWSAlerts(t.Context()))
	require.Len(t, repo.upserted, 2)
	assert.Equal(t, weatheralert.AlertCategoryWinterWeather, repo.upserted[0].AlertCategory)
	assert.Equal(t, 1, repo.expireCalls)
}

func TestGetActiveAlertsReturnsFeatureCollection(t *testing.T) {
	t.Parallel()

	geometry, err := parseGeometry(map[string]any{
		"type": "Polygon",
		"coordinates": []any{
			[]any{
				[]any{-97.0, 32.0},
				[]any{-96.0, 32.0},
				[]any{-96.0, 33.0},
				[]any{-97.0, 33.0},
				[]any{-97.0, 32.0},
			},
		},
	})
	require.NoError(t, err)

	repo := &repoStub{
		activeAlerts: []*weatheralert.WeatherAlert{
			{
				ID:            pulid.MustNew("walt_"),
				NWSID:         "urn:oid:1",
				Event:         "Flood Warning",
				AlertCategory: weatheralert.AlertCategoryFloodWater,
				Geometry:      geometry,
				FirstSeenAt:   1,
				LastUpdatedAt: 2,
			},
		},
	}
	service := New(Params{Logger: zap.NewNop(), Repo: repo})

	result, err := service.GetActiveAlerts(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	require.Len(t, result.Features, 1)
	assert.Equal(t, "FeatureCollection", result.Type)
	assert.Equal(t, "Flood Warning", result.Features[0].Properties.Event)
}

func TestGetActiveAlertsSkipsExpiredAlerts(t *testing.T) {
	t.Parallel()

	geometry, err := parseGeometry(map[string]any{
		"type": "Polygon",
		"coordinates": []any{
			[]any{
				[]any{-97.0, 32.0},
				[]any{-96.0, 32.0},
				[]any{-96.0, 33.0},
				[]any{-97.0, 33.0},
				[]any{-97.0, 32.0},
			},
		},
	})
	require.NoError(t, err)

	expiredAt := time.Now().UTC().Add(-1 * time.Minute).Unix()
	expires := time.Now().UTC().Add(-1 * time.Minute).Unix()
	repo := &repoStub{
		activeAlerts: []*weatheralert.WeatherAlert{
			{
				ID:            pulid.MustNew("walt_"),
				NWSID:         "urn:oid:1",
				Event:         "Flood Warning",
				AlertCategory: weatheralert.AlertCategoryFloodWater,
				Geometry:      geometry,
				FirstSeenAt:   1,
				LastUpdatedAt: 2,
				ExpiredAt:     &expiredAt,
				Expires:       &expires,
			},
		},
	}
	service := New(Params{Logger: zap.NewNop(), Repo: repo})

	result, err := service.GetActiveAlerts(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	require.Empty(t, result.Features)
}

func TestGetAlertDetailReturnsActivities(t *testing.T) {
	t.Parallel()

	geometry, err := parseGeometry(map[string]any{
		"type": "Polygon",
		"coordinates": []any{
			[]any{
				[]any{-97.0, 32.0},
				[]any{-96.0, 32.0},
				[]any{-96.0, 33.0},
				[]any{-97.0, 33.0},
				[]any{-97.0, 32.0},
			},
		},
	})
	require.NoError(t, err)

	alertID := pulid.MustNew("walt_")
	repo := &repoStub{
		alert: &weatheralert.WeatherAlert{
			ID:            alertID,
			NWSID:         "urn:oid:1",
			Event:         "Flood Warning",
			AlertCategory: weatheralert.AlertCategoryFloodWater,
			Geometry:      geometry,
			FirstSeenAt:   1,
			LastUpdatedAt: 2,
		},
		activities: []*weatheralert.Activity{
			{WeatherAlertID: alertID, ActivityType: weatheralert.ActivityTypeIssued, Timestamp: 1},
		},
	}
	service := New(Params{Logger: zap.NewNop(), Repo: repo})

	result, err := service.GetAlertDetail(t.Context(), &serviceports.GetWeatherAlertDetailRequest{ID: alertID})
	require.NoError(t, err)
	require.NotNil(t, result.Feature)
	require.Len(t, result.Activities, 1)
}

func TestGetActiveAlertsRejectsLegacyAlertCategory(t *testing.T) {
	t.Parallel()

	geometry, err := parseGeometry(map[string]any{
		"type": "Polygon",
		"coordinates": []any{
			[]any{
				[]any{-97.0, 32.0},
				[]any{-96.0, 32.0},
				[]any{-96.0, 33.0},
				[]any{-97.0, 33.0},
				[]any{-97.0, 32.0},
			},
		},
	})
	require.NoError(t, err)

	repo := &repoStub{
		activeAlerts: []*weatheralert.WeatherAlert{
			{
				ID:            pulid.MustNew("walt_"),
				NWSID:         "urn:oid:1",
				Event:         "Coastal Flood Warning",
				AlertCategory: weatheralert.AlertCategory("coastal_marine_tsunami"),
				Geometry:      geometry,
				FirstSeenAt:   1,
				LastUpdatedAt: 2,
			},
		},
	}
	service := New(Params{Logger: zap.NewNop(), Repo: repo})

	_, err = service.GetActiveAlerts(t.Context(), pagination.TenantInfo{})
	require.Error(t, err)
}
