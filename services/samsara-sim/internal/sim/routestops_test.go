package sim

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

func newRouteStopFixture() *Fixture {
	fixture := newGeofenceFixture()
	fixture.Drivers = []Record{
		{
			"id":   "drv-geo",
			"name": "Casey Rivera",
			"externalIds": map[string]any{
				"tmsWorkerId": "worker-geo",
			},
		},
	}
	fixture.Routes = []Record{
		{
			"id":     "route-geo",
			"name":   "Geo Corridor Route",
			"driver": map[string]any{"id": "drv-geo", "name": "Casey Rivera"},
			"vehicle": map[string]any{
				"id":   "veh-geo",
				"name": "Truck Geo",
			},
			"externalIds": map[string]any{
				"tmsShipmentId": "shp-geo",
			},
		},
	}
	return fixture
}

func newRouteStopLiveSimulator() *LiveSimulator {
	return NewLiveSimulator(NewStore(newRouteStopFixture()), "route-stop-seed", LiveSimulationOptions{
		FleetSize:    1,
		TripHoursMin: 1,
		TripHoursMax: 1,
	})
}

func TestRouteStopWebhookEmissionsArrivalAndDeparture(t *testing.T) {
	t.Parallel()

	live := newRouteStopLiveSimulator()
	now := time.Now().UTC()
	emissions := live.RouteStopWebhookEmissions(now, now.Add(-70*time.Minute), now, nil)
	if len(emissions) == 0 {
		t.Fatal("expected route-stop transitions across a full route loop")
	}

	sawArrival := false
	sawDeparture := false
	for idx := range emissions {
		emission := &emissions[idx]
		arrival := emission.EventType == routeStopEventArrival
		switch emission.EventType {
		case routeStopEventArrival:
			sawArrival = true
		case routeStopEventDeparture:
			sawDeparture = true
		default:
			t.Fatalf("unexpected route-stop event type %q", emission.EventType)
		}

		if got := stringValue(Record(emission.Data), "type"); got != routeStopUpdateType {
			t.Fatalf("expected type %q, got %q", routeStopUpdateType, got)
		}
		wantOperation := routeStopOperationDeparted
		if arrival {
			wantOperation = routeStopOperationArrived
		}
		if got := stringValue(Record(emission.Data), "operation"); got != wantOperation {
			t.Fatalf("expected operation %q, got %q", wantOperation, got)
		}
		if got := stringValue(Record(emission.Data), "assignedToRoute"); got != "route-geo" {
			t.Fatalf("expected assignedToRoute route-geo, got %q", got)
		}

		route, ok := anyAsMap(emission.Data["route"])
		if !ok || stringValue(Record(route), "id") != "route-geo" {
			t.Fatalf("expected route payload with id route-geo, got %v", emission.Data["route"])
		}
		driver, ok := anyAsMap(emission.Data["driver"])
		if !ok || stringValue(Record(driver), "id") != "drv-geo" {
			t.Fatalf("expected driver payload with id drv-geo, got %v", emission.Data["driver"])
		}
		vehicle, ok := anyAsMap(emission.Data["vehicle"])
		if !ok || stringValue(Record(vehicle), "id") != "veh-geo" {
			t.Fatalf("expected vehicle payload with id veh-geo, got %v", emission.Data["vehicle"])
		}
		if got := stringValue(Record(vehicle), "vin"); got != "1FUJGLDR5CLBP9999" {
			t.Fatalf("expected fixture vin, got %q", got)
		}

		details, ok := anyAsMap(emission.Data["routeStopDetails"])
		if !ok {
			t.Fatalf("expected routeStopDetails payload, got %T", emission.Data["routeStopDetails"])
		}
		if stringValue(Record(details), "id") == "" {
			t.Fatal("expected route stop id")
		}
		wantState := routeStopStateDeparted
		if arrival {
			wantState = routeStopStateArrived
		}
		if got := stringValue(Record(details), "state"); got != wantState {
			t.Fatalf("expected state %q, got %q", wantState, got)
		}
		if stringValue(Record(details), "actualArrivalTime") == "" {
			t.Fatal("expected actualArrivalTime on route stop details")
		}
		if _, hasOrders := details["orders"].([]any); !hasOrders {
			t.Fatal("expected orders array on route stop details")
		}
		if arrival {
			if _, hasDeparture := details["actualDepartureTime"]; hasDeparture {
				t.Fatal("did not expect actualDepartureTime on arrival")
			}
		} else if stringValue(Record(details), "actualDepartureTime") == "" {
			t.Fatal("expected actualDepartureTime on departure")
		}
	}
	if !sawArrival || !sawDeparture {
		t.Fatalf("expected both arrival and departure, arrival=%v departure=%v", sawArrival, sawDeparture)
	}
}

func TestRouteStopWebhookEmissionsAlignWithGeofence(t *testing.T) {
	t.Parallel()

	live := newRouteStopLiveSimulator()
	now := time.Now().UTC().Truncate(time.Minute)
	windowStart := now.Add(-70 * time.Minute)

	geofence := live.GeofenceWebhookEmissions(now, windowStart, now, nil)
	routeStops := live.RouteStopWebhookEmissions(now, windowStart, now, nil)
	if len(routeStops) != len(geofence) {
		t.Fatalf(
			"expected route-stop emissions to align with geofence, got %d vs %d",
			len(routeStops),
			len(geofence),
		)
	}
	for idx := range geofence {
		wantEvent := routeStopEventDeparture
		if geofence[idx].EventType == geofenceEventEntry {
			wantEvent = routeStopEventArrival
		}
		if routeStops[idx].EventType != wantEvent {
			t.Fatalf(
				"expected aligned event %q, got %q",
				wantEvent,
				routeStops[idx].EventType,
			)
		}
	}
}

func TestRouteStopWebhookEmissionsDeterministic(t *testing.T) {
	t.Parallel()

	live := newRouteStopLiveSimulator()
	now := time.Now().UTC().Truncate(time.Minute)
	first := live.RouteStopWebhookEmissions(now, now.Add(-70*time.Minute), now, nil)
	second := live.RouteStopWebhookEmissions(now, now.Add(-70*time.Minute), now, nil)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("expected identical route-stop emissions for identical sim time")
	}
}

func TestRouteStopWebhookEmissionsRequireRoute(t *testing.T) {
	t.Parallel()

	fixture := newRouteStopFixture()
	fixture.Routes = []Record{}
	live := NewLiveSimulator(NewStore(fixture), "route-stop-seed", LiveSimulationOptions{
		FleetSize:    1,
		TripHoursMin: 1,
		TripHoursMax: 1,
	})
	now := time.Now().UTC()
	emissions := live.RouteStopWebhookEmissions(now, now.Add(-70*time.Minute), now, nil)
	if len(emissions) != 0 {
		t.Fatalf("expected no route-stop events without a route, got %d", len(emissions))
	}
}

func newRouteStopTestServer(t *testing.T, webhookURL string) *Server {
	t.Helper()

	cfg := config.Default()
	cfg.Auth.Tokens = []string{"dev-samsara-token"}
	cfg.Simulation.FleetSize = 1
	cfg.Simulation.TripHoursMin = 1
	cfg.Simulation.TripHoursMax = 1
	cfg.Webhooks.Enabled = true
	cfg.Webhooks.MaxAttempts = 1
	cfg.Webhooks.InitialBackoff = 10 * time.Millisecond

	fixture := newRouteStopFixture()
	fixture.Webhooks = []Record{
		{
			"id":   "wh-route-stop",
			"name": "route stop sink",
			"url":  webhookURL,
			"simDelivery": map[string]any{
				"allowDuplicates":    false,
				"allowReorder":       false,
				"allowTimestampSkew": false,
				"retryJitterMs":      0,
			},
		},
	}

	store := NewStore(fixture)
	scenarios, err := NewScenarioEngine("route-stop-seed", "default")
	if err != nil {
		t.Fatalf("failed to initialize scenario engine: %v", err)
	}
	dispatcher := NewDispatcher(cfg.Webhooks, store, nil)
	t.Cleanup(dispatcher.Shutdown)
	return NewServer(&cfg, store, scenarios, dispatcher, nil)
}

func TestServerDispatchRouteStopEventsDeduplicated(t *testing.T) {
	capture := &webhookEventCapture{}
	webhookReceiver := httptest.NewServer(capture.handler(t))
	defer webhookReceiver.Close()

	srv := newRouteStopTestServer(t, webhookReceiver.URL)

	request := httptest.NewRequest(http.MethodGet, "/fleet/vehicles/stats/feed?types=gps", nil)
	request.Header.Set("Authorization", "Bearer dev-samsara-token")

	srv.dispatchRouteStopEvents(request, srv.simNow())
	srv.clock.Step(70 * time.Minute)
	at := srv.simNow()
	srv.dispatchRouteStopEvents(request, at)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(capture.snapshot()) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond)

	first := capture.snapshot()
	if len(first) == 0 {
		t.Fatal("expected route-stop deliveries after full loop")
	}
	for _, event := range first {
		if event.EventType != routeStopEventArrival && event.EventType != routeStopEventDeparture {
			t.Fatalf("unexpected event type %q", event.EventType)
		}
	}

	srv.routeStopWindow.reset()
	srv.dispatchRouteStopEvents(request, at)
	time.Sleep(200 * time.Millisecond)

	if got := len(capture.snapshot()); got != len(first) {
		t.Fatalf("expected deduplicated route-stop deliveries %d, got %d", len(first), got)
	}
}
