package sim

import (
	"net/http"
	"slices"
	"testing"
	"time"
)

func TestDriverEfficiencyWindowAndFilters(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	tests := []struct {
		target   string
		fragment string
	}{
		{
			target:   "/beta/fleet/drivers/efficiency?driverIds=1&driverTagIds=2",
			fragment: "cannot be combined",
		},
		{
			target:   "/beta/fleet/drivers/efficiency?driverIds=1&driverActivationStatus=active",
			fragment: "cannot be combined",
		},
		{target: "/beta/fleet/drivers/efficiency?endTime=2026-03-05T00:00:00Z", fragment: "future"},
		{
			target:   "/beta/fleet/drivers/efficiency?startTime=2026-01-01T00:00:00Z&endTime=2026-03-04T00:00:00Z",
			fragment: "31 days",
		},
		{
			target:   "/beta/fleet/drivers/efficiency?startTime=2026-03-04T10:00:00Z&endTime=2026-03-03T00:00:00Z",
			fragment: "endTime",
		},
		{
			target:   "/beta/fleet/drivers/efficiency?driverActivationStatus=x",
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

	result := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/beta/fleet/drivers/efficiency",
		nil,
		http.StatusOK,
	)
	data := result.data(t)
	if data["summaryEndTime"] != fleetTestTime.Format(time.RFC3339) ||
		data["summaryStartTime"] != fleetTestTime.Add(-24*time.Hour).Format(time.RFC3339) {
		t.Fatalf(
			"expected a 24h window ending at the hour, got %v - %v",
			data["summaryStartTime"],
			data["summaryEndTime"],
		)
	}
	summaries := data["driverSummaries"].([]any)
	if len(summaries) == 0 {
		t.Fatal("expected driver summaries over the last day")
	}
	for _, raw := range summaries {
		summary := Record(raw.(map[string]any))
		driverID := nestedString(summary, "driver", "id")
		if nestedString(summary, "driver", "username") == "" {
			t.Fatalf("expected driver usernames in the summary, got %v", summary["driver"])
		}
		vehicles := summary["vehicleSummaries"].([]any)
		totalDrive := 0.0
		for _, rawVehicle := range vehicles {
			vehicle := Record(rawVehicle.(map[string]any))
			totalDrive += floatFromAny(vehicle["driveTimeDurationMs"])
			if nestedString(vehicle, "vehicle", "id") == "" {
				t.Fatalf("expected vehicle references, got %v", vehicle)
			}
			if _, ok := anyAsMap(nestedAny(vehicle, "vehicle", "ExternalIds")); !ok {
				t.Fatalf(
					"expected the spec's ExternalIds key on vehicleTinyResponse, got %v",
					vehicle["vehicle"],
				)
			}
		}
		if totalDrive != floatFromAny(summary["totalDriveTimeDurationMs"]) {
			t.Fatalf("expected vehicle summaries to add up for %s", driverID)
		}
		timeline := srv.live.driverTimelineSegments(
			driverID,
			fleetTestTime.Add(-24*time.Hour),
			fleetTestTime,
			fleetTestTime,
		)
		expected := durationForStatuses(
			timeline,
			fleetTestTime.Add(-24*time.Hour),
			fleetTestTime,
			hosStatusDriving,
		)
		if diff := time.Duration(
			totalDrive,
		)*time.Millisecond - expected; diff > 2*time.Millisecond ||
			diff < -2*time.Millisecond {
			t.Fatalf("expected drive time %s from the HOS timeline for %s, got %s",
				expected, driverID, time.Duration(totalDrive)*time.Millisecond)
		}
		if floatFromAny(summary["totalFuelConsumedMl"]) <= 0 ||
			floatFromAny(summary["totalDistanceDrivenMeters"]) <= 0 {
			t.Fatalf("expected distance and fuel for %s", driverID)
		}
	}

	recent := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/beta/fleet/drivers/efficiency", map[string]string{
			"startTime": fleetTestTime.Add(-30 * time.Minute).Format(time.RFC3339),
		}),
		nil,
		http.StatusOK,
	).data(t)
	if len(recent["driverSummaries"].([]any)) != 0 {
		t.Fatal("expected no summaries when the window starts within the last hour")
	}
	single := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/beta/fleet/drivers/efficiency?driverIds="+fixtureDriverAlex,
		nil,
		http.StatusOK,
	).data(t)
	if len(single["driverSummaries"].([]any)) > 1 {
		t.Fatal("expected driverIds to narrow the summaries")
	}
}

func TestDriverTachographActivity(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	callAPI(t, srv, http.MethodGet, "/fleet/drivers/tachograph-activity/history", nil).
		expectError(t, http.StatusBadRequest, "startTime")
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers/tachograph-activity/history?startTime=2026-01-01T00:00:00Z&endTime=2026-03-01T00:00:00Z",
		nil,
	).
		expectError(t, http.StatusBadRequest, "30 days")
	start := fleetTestTime.Add(-24 * time.Hour)
	records := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/fleet/drivers/tachograph-activity/history", map[string]string{
			"startTime": start.Format(time.RFC3339),
			"endTime":   fleetTestTime.Format(time.RFC3339),
		}),
		nil,
		http.StatusOK,
	).list(t)
	if len(records) != 3 {
		t.Fatalf(
			"expected activity only for the three tachograph card holders, got %d",
			len(records),
		)
	}
	for _, record := range records {
		activity := record["activity"].([]any)
		if len(activity) == 0 {
			t.Fatalf("expected activity for %v", record["driver"])
		}
		previousEnd := start
		for _, raw := range activity {
			entry := Record(raw.(map[string]any))
			state := stringValue(entry, "state")
			if !slices.Contains(
				[]string{tachographStateDriving, tachographStateWork, tachographStateBreakRest},
				state,
			) {
				t.Fatalf("unexpected tachograph state %q", state)
			}
			entryStart := mustRFC3339(t, stringValue(entry, "startTime"))
			if !entryStart.Equal(previousEnd) {
				t.Fatalf("expected contiguous activity, got gap at %s", entryStart)
			}
			previousEnd = mustRFC3339(t, stringValue(entry, "endTime"))
		}
		if !previousEnd.Equal(fleetTestTime) {
			t.Fatalf("expected activity to cover the window, ended at %s", previousEnd)
		}
	}
	filtered := requireStatus(
		t,
		srv,
		http.MethodGet,
		query("/fleet/drivers/tachograph-activity/history", map[string]string{
			"startTime": start.Format(time.RFC3339),
			"endTime":   fleetTestTime.Format(time.RFC3339),
			"tagIds":    tagHazmatCertified,
		}),
		nil,
		http.StatusOK,
	).list(t)
	if len(filtered) != 2 {
		t.Fatalf("expected the two hazmat card holders, got %d", len(filtered))
	}
}

func TestDriverAuthToken(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	tests := []struct {
		body     map[string]any
		status   int
		fragment string
	}{
		{
			body:     map[string]any{"username": "arivera"},
			status:   http.StatusBadRequest,
			fragment: "code",
		},
		{
			body:     map[string]any{"code": "short", "username": "arivera"},
			status:   http.StatusBadRequest,
			fragment: "12",
		},
		{
			body:     map[string]any{"code": "abcdefghijklmn"},
			status:   http.StatusBadRequest,
			fragment: "one is required",
		},
		{
			body:     map[string]any{"code": "abcdefghijklmn", "externalId": "noseparator"},
			status:   http.StatusBadRequest,
			fragment: "key:value",
		},
		{
			body:     map[string]any{"code": "abcdefghijklmn", "driverId": float64(1)},
			status:   http.StatusNotFound,
			fragment: "driver",
		},
		{
			body:     map[string]any{"code": "abcdefghijklmn", "username": "ghost"},
			status:   http.StatusNotFound,
			fragment: "username",
		},
		{
			body: map[string]any{
				"code":     "abcdefghijklmn",
				"driverId": float64(1654973),
				"username": "jlee",
			},
			status:   http.StatusBadRequest,
			fragment: "same driver",
		},
	}
	for _, testCase := range tests {
		callAPI(t, srv, http.MethodPost, "/fleet/drivers/auth-token", testCase.body).
			expectError(t, testCase.status, testCase.fragment)
	}
	for _, body := range []map[string]any{
		{"code": "abcdefghijklmn", "driverId": float64(1654973)},
		{"code": "abcdefghijklmn", "externalId": "workerId:worker-1001"},
		{"code": "abcdefghijklmn", "username": "ARivera"},
	} {
		data := requireStatus(
			t,
			srv,
			http.MethodPost,
			"/fleet/drivers/auth-token",
			body,
			http.StatusOK,
		).data(t)
		if len(stringValue(data, "token")) < 32 ||
			floatFromAny(
				data["expirationTime"],
			) != float64(
				fleetTestTime.Add(driverAuthTokenTTL).UnixMilli(),
			) {
			t.Fatalf("unexpected token response %v", data)
		}
	}
	requireStatus(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverAlex, map[string]any{
		"driverActivationStatus": "deactivated",
	}, http.StatusOK)
	callAPI(t, srv, http.MethodPost, "/fleet/drivers/auth-token", map[string]any{
		"code": "abcdefghijklmn", "username": "arivera",
	}).expectError(t, http.StatusBadRequest, "deactivated")
}

func TestDriverWorkflows(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	all := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers/workflows",
		nil,
		http.StatusOK,
	).list(t)
	if len(all) != 6 {
		t.Fatalf("expected six fixture workflows, got %d", len(all))
	}
	for _, workflow := range all {
		if !uuidPattern.MatchString(recordID(workflow)) || len(workflow) != 3 {
			t.Fatalf("expected {id,name,workflowType} workflows, got %v", workflow)
		}
	}
	assetSelection := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers/workflows?workflowType=assetSelection",
		nil,
		http.StatusOK,
	).list(t)
	if len(assetSelection) != 2 {
		t.Fatalf("expected two assetSelection workflows, got %d", len(assetSelection))
	}
	callAPI(t, srv, http.MethodGet, "/fleet/drivers/workflows?workflowType=lunch", nil).
		expectError(t, http.StatusBadRequest, "workflowType")
	paged := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers/workflows?limit=4",
		nil,
		http.StatusOK,
	)
	if len(paged.list(t)) != 4 || paged.pagination(t)["hasNextPage"] != true {
		t.Fatalf("expected limit paging, got %s", paged.Body)
	}

	workflowID := recordID(assetSelection[1])
	tests := []struct {
		body     map[string]any
		status   int
		fragment string
	}{
		{
			body:     map[string]any{"driverIdsToPublish": []any{fixtureDriverAlex}},
			status:   http.StatusBadRequest,
			fragment: "workflowId",
		},
		{
			body:     map[string]any{"workflowId": workflowID},
			status:   http.StatusBadRequest,
			fragment: "at least one",
		},
		{
			body: map[string]any{
				"workflowId":         "00000000-0000-4000-8000-000000000000",
				"driverIdsToPublish": []any{fixtureDriverAlex},
			},
			status:   http.StatusNotFound,
			fragment: "workflow",
		},
		{
			body:     map[string]any{"workflowId": workflowID, "driverIdsToPublish": []any{"1"}},
			status:   http.StatusBadRequest,
			fragment: "driver",
		},
		{
			body: map[string]any{
				"workflowId":           workflowID,
				"driverIdsToPublish":   []any{fixtureDriverAlex},
				"driverIdsToUnpublish": []any{fixtureDriverAlex},
			},
			status:   http.StatusBadRequest,
			fragment: "cannot also contain",
		},
	}
	for _, testCase := range tests {
		callAPI(t, srv, http.MethodPost, "/fleet/drivers/workflow-assignments", testCase.body).
			expectError(t, testCase.status, testCase.fragment)
	}
	data := requireStatus(
		t,
		srv,
		http.MethodPost,
		"/fleet/drivers/workflow-assignments",
		map[string]any{
			"workflowId":           workflowID,
			"driverIdsToPublish":   []any{fixtureDriverAlex},
			"driverIdsToUnpublish": []any{fixtureDriverJordan},
		},
		http.StatusOK,
	).data(t)
	if stringValue(data, "workflowId") != workflowID {
		t.Fatalf("unexpected response %v", data)
	}
	stored, err := srv.store.Get(ResourceDriverWorkflows, workflowID)
	if err != nil {
		t.Fatalf("get workflow: %v", err)
	}
	published := stringListValues(stored["simPublishedDriverIds"])
	if !slices.Contains(published, fixtureDriverAlex) ||
		slices.Contains(published, fixtureDriverJordan) {
		t.Fatalf("expected publish/unpublish to apply, got %v", published)
	}
}
