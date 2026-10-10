package samsaraprovider

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/services"
	sharedsamsara "github.com/emoss08/trenova/shared/samsara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	positionStatsBody = `{
		"data": [
			{
				"id": "281474977075805",
				"name": "Truck 1",
				"gps": {
					"time": "2026-03-01T14:00:00Z",
					"latitude": 37.77,
					"longitude": -122.41,
					"headingDegrees": 90,
					"speedMilesPerHour": 55.5,
					"reverseGeo": {"formattedLocation": "San Francisco, CA"}
				},
				"engineState": {"time": "2026-03-01T14:00:00Z", "value": "On"},
				"fuelPercent": {"time": "2026-03-01T14:00:00Z", "value": 64}
			},
			{
				"id": "281474977075806",
				"name": "Truck 2",
				"gps": {"time": "2026-03-01T14:01:00Z", "latitude": 34.05, "longitude": -118.24}
			},
			{
				"id": "281474977075807",
				"name": "Truck 3",
				"gps": {"time": "2026-03-01T14:02:00Z", "latitude": 40.71, "longitude": -74.0}
			},
			{"id": "281474977075808", "name": "No GPS"}
		],
		"pagination": {"endCursor": "", "hasNextPage": false}
	}`
	odometerStatsBody = `{
		"data": [
			{
				"id": "281474977075805",
				"name": "Truck 1",
				"obdOdometerMeters": {"time": "2026-03-01T13:59:00Z", "value": 120500300},
				"gpsOdometerMeters": {"time": "2026-03-01T13:59:00Z", "value": 99}
			},
			{
				"id": "281474977075806",
				"name": "Truck 2",
				"gpsOdometerMeters": {"time": "2026-03-01T13:58:00Z", "value": 88000000}
			},
			{"id": "281474977075808", "name": "No GPS", "obdOdometerMeters": {"time": "2026-03-01T13:58:00Z", "value": 5}}
		],
		"pagination": {"endCursor": "", "hasNextPage": false}
	}`
)

func newTestProvider(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := sharedsamsara.New(
		"test-token",
		sharedsamsara.WithBaseURL(server.URL),
		sharedsamsara.WithRateLimiting(false),
		sharedsamsara.WithRetry(sharedsamsara.RetryConfig{
			Enabled:        false,
			MaxAttempts:    1,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     time.Millisecond,
		}),
	)
	require.NoError(t, err)
	return New(client)
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func TestListPositionsMergesOdometerFromSecondStatsCall(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	requestedTypes := make([]string, 0, 2)
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fleet/vehicles/stats" {
			writeJSON(w, http.StatusNotFound, `{"message":"not found"}`)
			return
		}
		assert.False(t, r.URL.Query().Has("limit"))
		types := r.URL.Query().Get("types")
		assert.LessOrEqual(t, len(strings.Split(types, ",")), 3)

		mu.Lock()
		requestedTypes = append(requestedTypes, types)
		mu.Unlock()

		switch types {
		case "gps,engineStates,fuelPercents":
			writeJSON(w, http.StatusOK, positionStatsBody)
		case "obdOdometerMeters,gpsOdometerMeters":
			writeJSON(w, http.StatusOK, odometerStatsBody)
		default:
			writeJSON(w, http.StatusBadRequest, `{"message":"unexpected types"}`)
		}
	})

	positions, err := provider.ListPositions(t.Context())
	require.NoError(t, err)
	require.Len(t, positions, 3)

	assert.ElementsMatch(
		t,
		[]string{"gps,engineStates,fuelPercents", "obdOdometerMeters,gpsOdometerMeters"},
		requestedTypes,
	)

	byID := make(map[string]services.ProviderPosition, len(positions))
	for _, position := range positions {
		byID[position.VehicleID] = position
	}

	first := byID["281474977075805"]
	assert.InDelta(t, 37.77, first.Latitude, 1e-9)
	assert.InDelta(t, 90.0, first.HeadingDegrees, 1e-9)
	assert.InDelta(t, 55.5, first.SpeedMph, 1e-9)
	assert.Equal(t, "San Francisco, CA", first.FormattedLocation)
	assert.Equal(t, telematics.EngineState("On"), first.EngineState)
	require.NotNil(t, first.FuelPercent)
	assert.InDelta(t, 64.0, *first.FuelPercent, 1e-9)
	require.NotNil(t, first.OdometerMeters)
	assert.Equal(t, int64(120500300), *first.OdometerMeters)
	assert.Equal(
		t,
		time.Date(2026, 3, 1, 14, 0, 0, 0, time.UTC).Unix(),
		first.RecordedAt,
	)

	second := byID["281474977075806"]
	require.NotNil(t, second.OdometerMeters)
	assert.Equal(t, int64(88000000), *second.OdometerMeters)

	third := byID["281474977075807"]
	assert.Nil(t, third.OdometerMeters)

	_, hasNoGPS := byID["281474977075808"]
	assert.False(t, hasNoGPS)
}

func TestListPositionsFailsWhenOdometerCallFails(t *testing.T) {
	t.Parallel()

	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Query().Get("types"), "obdOdometerMeters") {
			writeJSON(w, http.StatusBadRequest, `{"message":"bad request","requestId":"r1"}`)
			return
		}
		writeJSON(w, http.StatusOK, positionStatsBody)
	})

	_, err := provider.ListPositions(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch samsara vehicle odometers")
	var apiErr *sharedsamsara.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestListVehiclesPaginatesAssetsWithoutLimit(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	cursors := make([]string, 0, 2)
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/assets", r.URL.Path)
		query := r.URL.Query()
		assert.False(t, query.Has("limit"))
		assert.Equal(t, "vehicle", query.Get("type"))

		mu.Lock()
		cursors = append(cursors, query.Get("after"))
		mu.Unlock()

		if query.Get("after") == "" {
			writeJSON(w, http.StatusOK, `{
				"data": [{"id": "281474977075805", "name": "Truck 1", "vin": "1FUJGLDR2LLLV0001"}],
				"pagination": {"endCursor": "cursor-2", "hasNextPage": true}
			}`)
			return
		}
		writeJSON(w, http.StatusOK, `{
			"data": [{"id": "281474977075806", "name": "Truck 2", "licensePlate": "ABC123"}],
			"pagination": {"endCursor": "", "hasNextPage": false}
		}`)
	})

	vehicles, err := provider.ListVehicles(t.Context())
	require.NoError(t, err)
	require.Len(t, vehicles, 2)
	assert.Equal(t, []string{"", "cursor-2"}, cursors)
	assert.Equal(t, "281474977075805", vehicles[0].ID)
	assert.Equal(t, "1FUJGLDR2LLLV0001", vehicles[0].VIN)
	assert.Equal(t, "ABC123", vehicles[1].LicensePlate)
}

func TestParseWebhookEventRouteStopUsesStopAddressExternalIDs(t *testing.T) {
	t.Parallel()

	provider := New(nil)
	body := []byte(`{
		"eventId": "8d7c2c1e-3c43-4a59-9a3f-51d7a0f2b001",
		"eventTime": "2026-03-01T14:00:00Z",
		"eventType": "RouteStopArrival",
		"orgId": 20936,
		"webhookId": "523918",
		"data": {
			"operation": "stop arrived",
			"type": "route tracking",
			"time": "2026-03-01T14:00:00Z",
			"driver": {"id": "1654973", "name": "Alex Driver"},
			"vehicle": {
				"id": "281474977075805",
				"name": "Truck 1",
				"vin": "1FUJGLDR2LLLV0001",
				"externalIds": {"trenovaTractorId": "trac_1"}
			},
			"route": {
				"id": "4291022",
				"name": "Load 1001",
				"stops": [
					{
						"id": "8818201",
						"address": {"id": "22410013", "externalIds": {"trenovaLocationId": "loc_a"}}
					},
					{
						"id": "8818202",
						"address": {"id": "22410014", "externalIds": {"trenovaLocationId": "loc_b"}}
					}
				]
			},
			"routeStopDetails": {
				"id": "8818202",
				"state": "arrived",
				"actualArrivalTime": "2026-03-01T13:59:30Z",
				"externalIds": {"trenovaStopId": "stp_2"}
			}
		}
	}`)

	event, err := provider.ParseWebhookEvent(body)
	require.NoError(t, err)
	assert.Equal(t, services.ProviderEventKindStopArrival, event.Kind)
	assert.Equal(t, "281474977075805", event.VehicleID)
	assert.Equal(t, "1654973", event.DriverID)
	require.NotNil(t, event.Stop)
	assert.Equal(t, map[string]string{"trenovaLocationId": "loc_b"}, event.Stop.AddressExternalIDs)
	assert.Equal(t, map[string]string{"trenovaStopId": "stp_2"}, event.Stop.StopExternalIDs)
	assert.Equal(t, "8818202", event.Stop.RouteStopID)
	assert.Equal(t, "1FUJGLDR2LLLV0001", event.Stop.VehicleVIN)
	assert.Equal(
		t,
		time.Date(2026, 3, 1, 13, 59, 30, 0, time.UTC).Unix(),
		event.Stop.OccurredAt,
	)
}

func TestParseWebhookEventRouteStopWithoutRouteStopsLeavesAddressEmpty(t *testing.T) {
	t.Parallel()

	provider := New(nil)
	body := []byte(`{
		"eventId": "8d7c2c1e-3c43-4a59-9a3f-51d7a0f2b002",
		"eventTime": "2026-03-01T15:00:00Z",
		"eventType": "RouteStopDeparture",
		"orgId": 20936,
		"webhookId": "523918",
		"data": {
			"operation": "stop departed",
			"type": "route tracking",
			"time": "2026-03-01T15:00:00Z",
			"vehicle": {"id": "281474977075805", "externalIds": {"trenovaTractorId": "trac_1"}},
			"route": {"id": "4291022"},
			"routeStopDetails": {
				"id": "8818202",
				"state": "departed",
				"actualDepartureTime": "2026-03-01T14:59:00Z",
				"externalIds": {"trenovaLocationId": "loc_b"}
			}
		}
	}`)

	event, err := provider.ParseWebhookEvent(body)
	require.NoError(t, err)
	assert.Equal(t, services.ProviderEventKindStopDeparture, event.Kind)
	require.NotNil(t, event.Stop)
	assert.Nil(t, event.Stop.AddressExternalIDs)
	assert.Equal(t, map[string]string{"trenovaLocationId": "loc_b"}, event.Stop.StopExternalIDs)
}
