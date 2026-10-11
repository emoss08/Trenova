package sim

import (
	"net/http"
	"testing"
	"time"
)

func TestStoreTransactRollsBackOnError(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	before, err := srv.store.List(ResourceDrivers)
	if err != nil {
		t.Fatalf("list drivers: %v", err)
	}
	body := validDriverBody("rollback.driver")
	body["tagIds"] = []any{"404"}
	callAPI(
		t,
		srv,
		http.MethodPost,
		"/fleet/drivers",
		body,
	).expectError(t, http.StatusBadRequest, "tag")
	after, err := srv.store.List(ResourceDrivers)
	if err != nil {
		t.Fatalf("list drivers: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf(
			"expected the failed create to leave no driver behind, got %d -> %d",
			len(before),
			len(after),
		)
	}
	created := requireStatus(
		t,
		srv,
		http.MethodPost,
		"/fleet/drivers",
		validDriverBody("rollback.driver"),
		http.StatusOK,
	).data(t)
	if recordID(created) != "1655320" {
		t.Fatalf(
			"expected the rolled back id to be reused deterministically, got %s",
			recordID(created),
		)
	}
}

func TestDerivedAssignmentsFollowHOSWorkdays(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	view := srv.fleetView()
	start := fleetTestTime.Add(-72 * time.Hour)
	intervals := view.assignmentIntervals(&assignmentQuery{
		WindowStart: &start,
		WindowEnd:   &fleetTestTime,
		DriverIDs:   map[string]struct{}{fixtureDriverAlex: {}},
		DriverApp:   true,
	})
	if len(intervals) < 2 {
		t.Fatalf("expected several workday assignments, got %d", len(intervals))
	}
	timeline := srv.live.driverTimelineSegments(
		fixtureDriverAlex,
		start.Add(-24*time.Hour),
		fleetTestTime,
		fleetTestTime,
	)
	for _, interval := range intervals {
		if interval.VehicleID != fixtureTruck1001 || interval.Type != assignmentTypeDriverApp {
			t.Fatalf("expected driverApp assignments on Truck 1001, got %+v", interval)
		}
		index := timelineSegmentIndexAt(timeline, interval.Start)
		if index < 0 || isRestStatus(timeline[index].Status) {
			t.Fatalf("expected assignment start %s to be on duty", interval.Start)
		}
		if interval.End != nil {
			before := timelineSegmentIndexAt(timeline, interval.End.Add(-time.Second))
			if before < 0 || isRestStatus(timeline[before].Status) {
				t.Fatalf("expected assignment end %s to close a duty block", interval.End)
			}
		}
	}
}

func TestDriverVehicleAssignmentListValidation(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	tests := []struct {
		target   string
		fragment string
	}{
		{target: "/fleet/driver-vehicle-assignments", fragment: "filterBy"},
		{target: "/fleet/driver-vehicle-assignments?filterBy=trucks", fragment: "filterBy"},
		{
			target:   "/fleet/driver-vehicle-assignments?filterBy=drivers&vehicleIds=1",
			fragment: "filterBy=vehicles",
		},
		{
			target:   "/fleet/driver-vehicle-assignments?filterBy=vehicles&driverIds=1",
			fragment: "filterBy=drivers",
		},
		{
			target:   "/fleet/driver-vehicle-assignments?filterBy=vehicles&sourceName=x",
			fragment: "filterBy=drivers",
		},
		{
			target:   "/fleet/driver-vehicle-assignments?filterBy=drivers&sourceName=x&driverIds=1",
			fragment: "cannot be combined",
		},
		{
			target:   "/fleet/driver-vehicle-assignments?filterBy=drivers&assignmentType=magic",
			fragment: "assignmentType",
		},
		{
			target:   "/fleet/driver-vehicle-assignments?filterBy=drivers&startTime=nope",
			fragment: "startTime",
		},
		{
			target:   "/fleet/driver-vehicle-assignments?filterBy=drivers&startTime=2026-03-04T00:00:00Z&endTime=2026-03-03T00:00:00Z",
			fragment: "endTime",
		},
		{
			target:   "/fleet/vehicles/driver-assignments?startTime=2026-02-01T00:00:00Z&endTime=2026-03-01T00:00:00Z",
			fragment: "7 days",
		},
		{
			target:   "/fleet/drivers/vehicle-assignments?startTime=2026-02-01T00:00:00Z&endTime=2026-03-01T00:00:00Z",
			fragment: "7 days",
		},
		{
			target:   "/fleet/drivers/vehicle-assignments?driverActivationStatus=gone",
			fragment: "driverActivationStatus",
		},
	}
	for _, testCase := range tests {
		callAPI(
			t,
			srv,
			http.MethodGet,
			testCase.target,
			nil,
		).expectError(t, http.StatusBadRequest, testCase.fragment)
	}
}

func TestDriverVehicleAssignmentLifecycle(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	callAPI(t, srv, http.MethodPost, "/fleet/driver-vehicle-assignments", map[string]any{
		"vehicleId": fixtureTruck1001,
	}).expectError(t, http.StatusBadRequest, "driverId")
	callAPI(t, srv, http.MethodPost, "/fleet/driver-vehicle-assignments", map[string]any{
		"driverId": "workerId:none", "vehicleId": fixtureTruck1001,
	}).expectError(t, http.StatusNotFound, "driver")
	callAPI(t, srv, http.MethodPost, "/fleet/driver-vehicle-assignments", map[string]any{
		"driverId": fixtureDriverCameron, "vehicleId": fixtureTrailer2042,
	}).expectError(t, http.StatusNotFound, "vehicle")
	callAPI(t, srv, http.MethodPost, "/fleet/driver-vehicle-assignments", map[string]any{
		"driverId": fixtureDriverCameron, "vehicleId": fixtureTruck1001,
		"startTime": "2026-03-04T10:00:00Z", "endTime": "2026-03-04T09:00:00Z",
	}).expectError(t, http.StatusBadRequest, "endTime")
	callAPI(t, srv, http.MethodPost, "/fleet/driver-vehicle-assignments", map[string]any{
		"driverId": fixtureDriverCameron, "vehicleId": fixtureTruck1001,
		"metadata": map[string]any{"sourceName": string(make([]byte, 101))},
	}).expectError(t, http.StatusBadRequest, "100")

	start := fleetTestTime.Add(-30 * time.Minute)
	created := requireStatus(
		t,
		srv,
		http.MethodPost,
		"/fleet/driver-vehicle-assignments",
		map[string]any{
			"driverId":  "workerId:worker-1012",
			"vehicleId": "samsara.vin:1FUJGLDR5CLBP1001",
			"startTime": start.Format(time.RFC3339),
			"metadata":  map[string]any{"sourceName": "TMS dispatch"},
		},
		http.StatusCreated,
	)
	if nestedString(created.Payload, "data", "message") != assignmentSubmittedMessage {
		t.Fatalf("unexpected create response %s", created.Body)
	}
	callAPI(t, srv, http.MethodPost, "/fleet/driver-vehicle-assignments", map[string]any{
		"driverId":  fixtureDriverCameron,
		"vehicleId": fixtureTruck1001,
		"startTime": start.Format(time.RFC3339),
	}).expectError(t, http.StatusBadRequest, "already exists")

	listed := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/driver-vehicle-assignments?filterBy=drivers&driverIds=workerId:worker-1012",
		nil,
		http.StatusOK,
	).list(t)
	var external Record
	for _, record := range listed {
		if stringValue(record, "assignmentType") == assignmentTypeExternal {
			external = record
		}
	}
	if external == nil || nestedString(external, "vehicle", "id") != fixtureTruck1001 ||
		nestedString(external, "metadata", "sourceName") != "TMS dispatch" ||
		nestedString(external, "driver", fieldExternalIDs, "workerId") != "worker-1012" {
		t.Fatalf("expected the external assignment in the listing, got %v", listed)
	}
	if _, ongoing := external["endTime"]; ongoing {
		t.Fatal("expected an ongoing assignment without endTime")
	}
	unknown := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/driver-vehicle-assignments?filterBy=drivers&driverIds=workerId:nobody",
		nil,
		http.StatusOK,
	).list(t)
	if len(unknown) != 0 {
		t.Fatalf("expected unknown driver references to match nothing, got %d", len(unknown))
	}
	bySource := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/driver-vehicle-assignments?filterBy=drivers&sourceName=TMS%20dispatch",
		nil,
		http.StatusOK,
	).list(t)
	if len(bySource) != 1 {
		t.Fatalf("expected the source-name lookup to find one assignment, got %d", len(bySource))
	}

	view := srv.fleetView()
	if view.currentDriverOf(fixtureTruck1001) != fixtureDriverCameron {
		t.Fatalf("expected the active assignment to make Cameron the current driver, got %q",
			view.currentDriverOf(fixtureTruck1001))
	}
	if roster := srv.live.loadDriverRoster(); roster[fixtureDriverCameron].VehicleID != fixtureTruck1001 {
		t.Fatalf(
			"expected the live roster to follow the assignment, got %v",
			roster[fixtureDriverCameron],
		)
	}
	clocks := srv.live.HOSClocks(fleetTestTime, []string{fixtureDriverCameron})
	if nestedString(clocks[0], "currentVehicle", "id") != fixtureTruck1001 {
		t.Fatalf(
			"expected HOS currentVehicle to follow the assignment, got %v",
			clocks[0]["currentVehicle"],
		)
	}
	vehicle := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/"+fixtureTruck1001,
		nil,
		http.StatusOK,
	).data(t)
	if nestedString(vehicle, "staticAssignedDriver", "id") != fixtureDriverCameron {
		t.Fatalf(
			"expected the vehicle's current driver to follow the assignment, got %v",
			vehicle["staticAssignedDriver"],
		)
	}
	alexIntervals := view.assignmentIntervals(&assignmentQuery{
		WindowStart: &start,
		WindowEnd:   &fleetTestTime,
		DriverIDs:   map[string]struct{}{fixtureDriverAlex: {}},
		DriverApp:   true,
	})
	for _, interval := range alexIntervals {
		if interval.End == nil || interval.End.After(start) {
			t.Fatalf(
				"expected Alex's assignment on Truck 1001 to end when Cameron took over, got %+v",
				interval,
			)
		}
	}

	callAPI(t, srv, http.MethodPatch, "/fleet/driver-vehicle-assignments", map[string]any{
		"isPassenger": true,
	}).expectError(t, http.StatusBadRequest, "sourceName")
	callAPI(t, srv, http.MethodPatch, "/fleet/driver-vehicle-assignments", map[string]any{
		"driverId": fixtureDriverCameron, "vehicleId": fixtureTruck1001,
	}).expectError(t, http.StatusBadRequest, "required together")
	callAPI(t, srv, http.MethodPatch, "/fleet/driver-vehicle-assignments", map[string]any{
		"metadata": map[string]any{"sourceName": "unknown"},
	}).expectError(t, http.StatusNotFound, "source name")
	end := fleetTestTime.Add(30 * time.Minute)
	patched := requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/fleet/driver-vehicle-assignments",
		map[string]any{
			"driverId":  fixtureDriverCameron,
			"vehicleId": fixtureTruck1001,
			"startTime": start.Format(time.RFC3339),
			"endTime":   end.Format(time.RFC3339),
		},
		http.StatusAccepted,
	)
	if nestedString(patched.Payload, "data", "message") != assignmentUpdatedMessage {
		t.Fatalf("unexpected patch response %s", patched.Body)
	}
	requireStatus(t, srv, http.MethodPatch, "/fleet/driver-vehicle-assignments", map[string]any{
		"metadata": map[string]any{"sourceName": "TMS dispatch"},
		"endTime":  nil,
	}, http.StatusAccepted)
	stored, err := srv.store.List(ResourceDriverVehicleAssignments)
	if err != nil || len(stored) != 1 || stringValue(stored[0], fieldEndTime) != "" {
		t.Fatalf("expected endTime null to make the assignment ongoing, got %v (%v)", stored, err)
	}

	callAPI(t, srv, http.MethodDelete, "/fleet/driver-vehicle-assignments", map[string]any{}).
		expectError(t, http.StatusBadRequest, "vehicleId")
	callAPI(
		t,
		srv,
		http.MethodDelete,
		"/fleet/driver-vehicle-assignments",
		map[string]any{"vehicleId": "999"},
	).
		expectError(t, http.StatusNotFound, "vehicle")
	requireStatus(t, srv, http.MethodDelete, "/fleet/driver-vehicle-assignments", map[string]any{
		"vehicleId": fixtureTruck1001,
		"startTime": fleetTestTime.Add(-2 * time.Hour).Format(time.RFC3339),
		"endTime":   fleetTestTime.Format(time.RFC3339),
	}, http.StatusNoContent)
	stored, err = srv.store.List(ResourceDriverVehicleAssignments)
	if err != nil || len(stored) != 0 {
		t.Fatalf("expected the API assignment to be deleted, got %v (%v)", stored, err)
	}
	if srv.fleetView().currentDriverOf(fixtureTruck1001) != fixtureDriverAlex {
		t.Fatal("expected the base pairing to return after deletion")
	}
}

func TestLegacyAssignmentEndpointsShape(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	start := fleetTestTime.Add(-6 * 24 * time.Hour).Format(time.RFC3339)
	end := fleetTestTime.Format(time.RFC3339)
	vehicles := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/fleet/vehicles/driver-assignments", map[string]string{
			"vehicleIds": fixtureTruck1001 + ",tmsVehicleId:unit-1002",
			"startTime":  start,
			"endTime":    end,
		}),
		nil,
		http.StatusOK,
	).list(t)
	if len(vehicles) != 2 {
		t.Fatalf("expected both requested vehicles, got %v", listIDs(vehicles))
	}
	for _, vehicle := range vehicles {
		assignments, ok := vehicle["driverAssignments"].([]any)
		if !ok || len(assignments) == 0 {
			t.Fatalf("expected driverApp assignments for %s, got %v", recordID(vehicle), vehicle)
		}
		first := Record(assignments[0].(map[string]any))
		if stringValue(first, "assignmentType") != assignmentTypeDriverApp ||
			nestedString(first, "driver", "id") == "" {
			t.Fatalf("unexpected assignment shape %v", first)
		}
	}
	drivers := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/fleet/drivers/vehicle-assignments", map[string]string{
			"driverIds": "workerId:worker-1001",
			"startTime": start,
			"endTime":   end,
		}),
		nil,
		http.StatusOK,
	).list(t)
	if len(drivers) != 1 || drivers[0]["driverActivationStatus"] != "active" {
		t.Fatalf("expected one active driver, got %v", drivers)
	}
	assignments := drivers[0]["vehicleAssignments"].([]any)
	if len(assignments) == 0 ||
		nestedString(Record(assignments[0].(map[string]any)), "vehicle", "id") != fixtureTruck1001 {
		t.Fatalf("expected Truck 1001 assignments, got %v", assignments)
	}
}

func TestRemoteSignOutEndsCurrentAssignment(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	view := srv.fleetView()
	windowStart := fleetTestTime.Add(-24 * time.Hour)
	var ongoing *assignmentInterval
	for _, driver := range view.snap.drivers {
		for _, interval := range view.assignmentIntervals(&assignmentQuery{
			WindowStart: &windowStart,
			WindowEnd:   &fleetTestTime,
			DriverIDs:   map[string]struct{}{recordID(driver): {}},
			DriverApp:   true,
		}) {
			if interval.End == nil {
				copied := interval
				ongoing = &copied
			}
		}
		if ongoing != nil {
			break
		}
	}
	if ongoing == nil {
		t.Fatal("expected at least one driver on duty at the test time")
	}
	callAPI(t, srv, http.MethodPost, "/fleet/drivers/remote-sign-out", map[string]any{}).
		expectError(t, http.StatusBadRequest, "driverId")
	callAPI(
		t,
		srv,
		http.MethodPost,
		"/fleet/drivers/remote-sign-out",
		map[string]any{"driverId": "1"},
	).
		expectError(t, http.StatusNotFound, "driver")
	result := requireStatus(
		t,
		srv,
		http.MethodPost,
		"/fleet/drivers/remote-sign-out",
		map[string]any{
			"driverId": ongoing.DriverID,
		},
		http.StatusOK,
	)
	if stringValue(Record(result.Payload), "driverName") == "" {
		t.Fatalf("expected driverName in the response, got %s", result.Body)
	}
	for _, interval := range srv.fleetView().assignmentIntervals(&assignmentQuery{
		WindowStart: &windowStart,
		WindowEnd:   &fleetTestTime,
		DriverIDs:   map[string]struct{}{ongoing.DriverID: {}},
		DriverApp:   true,
	}) {
		if interval.End == nil {
			t.Fatalf("expected the sign-out to end the ongoing assignment, got %+v", interval)
		}
	}
}

func TestVoiceSignInCreatesAssignment(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	callAPI(
		t,
		srv,
		http.MethodPost,
		"/fleet/drivers/voice-sign-in/resolve-assignment",
		map[string]any{
			"driverName": "Nobody Here", "vehicleId": fixtureTruck1012,
		},
	).expectError(t, http.StatusNotFound, "driver")
	callAPI(
		t,
		srv,
		http.MethodPost,
		"/fleet/drivers/voice-sign-in/resolve-assignment",
		map[string]any{
			"driverName": "Jamie Clark", "vehicleId": "nope",
		},
	).expectError(t, http.StatusNotFound, "vehicle")
	callAPI(
		t,
		srv,
		http.MethodPost,
		"/fleet/drivers/voice-sign-in/resolve-assignment",
		map[string]any{
			"driverName": "Jamie Clark",
		},
	).expectError(t, http.StatusBadRequest, "vehicleId")

	result := requireStatus(
		t,
		srv,
		http.MethodPost,
		"/fleet/drivers/voice-sign-in/resolve-assignment",
		map[string]any{
			"driverName": "  jamie   CLARK ",
			"vehicleId":  "tmsVehicleId:unit-1012",
		},
		http.StatusCreated,
	).data(t)
	if stringValue(result, "driverId") != "1655261" ||
		stringValue(result, "driverName") != "Jamie Clark" {
		t.Fatalf("unexpected voice sign-in response %v", result)
	}
	listed := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/driver-vehicle-assignments?filterBy=vehicles&vehicleIds="+fixtureTruck1012+"&assignmentType=voiceSignIn",
		nil,
		http.StatusOK,
	).list(t)
	if len(listed) != 1 || nestedString(listed[0], "driver", "id") != "1655261" {
		t.Fatalf("expected the voice sign-in assignment, got %v", listed)
	}
	if srv.fleetView().currentDriverOf(fixtureTruck1012) != "1655261" {
		t.Fatal("expected the voice sign-in to make Jamie the current driver")
	}
}
