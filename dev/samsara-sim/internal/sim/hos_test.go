package sim

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	fixtureDriverRiley = "1655101"
	fixtureDriverSam   = "1655230"
	tagElPasoTerminal  = "342511"
)

func driverIDsFromRecords(t *testing.T, records []Record) []string {
	t.Helper()

	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, nestedString(record, "driver", "id"))
	}
	return out
}

func taggedDriverIDs(t *testing.T, srv *Server, tagID string) []string {
	t.Helper()

	drivers := callAPI(t, srv, http.MethodGet, "/fleet/drivers?tagIds="+tagID, nil).
		expect(t, http.StatusOK).
		list(t)
	if len(drivers) == 0 {
		t.Fatalf("expected fixture drivers tagged %s", tagID)
	}
	return listIDs(drivers)
}

func TestHOSLogsDefaultWindowIsNowAndAnswersQuickly(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})

	started := time.Now()
	records := callAPI(t, srv, http.MethodGet, "/fleet/hos/logs", nil).expect(t, http.StatusOK).list(t)
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("expected /fleet/hos/logs without a window to answer quickly, took %s", elapsed)
	}
	if len(records) == 0 {
		t.Fatal("expected HOS logs for the active drivers")
	}
	now := srv.simNow()
	for _, record := range records {
		logs := listOf(record["hosLogs"])
		if len(logs) != 1 {
			t.Fatalf("expected exactly the log in effect now, got %d entries for %v", len(logs), record["driver"])
		}
		entry := Record(mapOf(logs[0]))
		start := mustParseRecordTime(t, entry, "logStartTime")
		if start.After(now) {
			t.Fatalf("expected log in effect at %s, got start %s", now, start)
		}
		if end := stringValue(entry, "logEndTime"); end != "" {
			t.Fatalf("expected the current log to be open-ended, got logEndTime %s", end)
		}
		if vehicle := mapOf(entry["vehicle"]); vehicle != nil {
			if _, ok := vehicle["ExternalIds"]; !ok {
				t.Fatalf("expected vehicleTinyResponse ExternalIds, got %v", vehicle)
			}
		}
	}
}

func TestHOSLogsLongWindowStaysFast(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	target := query("/fleet/hos/logs", map[string]string{
		"startTime": fleetTestTime.Add(-60 * 24 * time.Hour).Format(time.RFC3339),
		"endTime":   fleetTestTime.Format(time.RFC3339),
		"driverIds": fixtureDriverAlex,
	})
	started := time.Now()
	records := callAPI(t, srv, http.MethodGet, target, nil).expect(t, http.StatusOK).list(t)
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("expected a 60-day HOS log window to stay fast, took %s", elapsed)
	}
	if len(records) != 1 || len(listOf(records[0]["hosLogs"])) < 100 {
		t.Fatalf("expected one driver with 60 days of logs, got %d records", len(records))
	}
}

func TestHOSLogsValidationAndFilters(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})

	callAPI(t, srv, http.MethodGet, "/fleet/hos/logs?startTime=yesterday", nil).
		expectError(t, http.StatusBadRequest, "Invalid value for parameter startTime")
	callAPI(t, srv, http.MethodGet, "/fleet/hos/logs?endTime=2026-03-04T15:00:00Z&startTime=2026-03-04T16:00:00Z", nil).
		expectError(t, http.StatusBadRequest, "endTime")
	callAPI(t, srv, http.MethodGet, "/fleet/hos/logs?endTime=2026-03-01T00:00:00Z", nil).
		expectError(t, http.StatusBadRequest, "must not be before startTime")
	callAPI(t, srv, http.MethodGet, "/fleet/hos/logs?after=bogus", nil).
		expectError(t, http.StatusBadRequest, "after")

	window := map[string]string{
		"startTime": "2026-03-04T12:00:00.000Z",
		"endTime":   "2026-03-04T09:00:00-06:00",
	}
	tagged := taggedDriverIDs(t, srv, tagAustinTerminal)
	filtered := callAPI(t, srv, http.MethodGet, query("/fleet/hos/logs", mergeParams(window, map[string]string{
		"tagIds": tagAustinTerminal,
	})), nil).expect(t, http.StatusOK).list(t)
	if got := strings.Join(driverIDsFromRecords(t, filtered), ","); got != strings.Join(tagged, ",") {
		t.Fatalf("expected tagIds to select %v, got %s", tagged, got)
	}
	parent := callAPI(t, srv, http.MethodGet, query("/fleet/hos/logs", mergeParams(window, map[string]string{
		"parentTagIds": tagCentralTexas,
	})), nil).expect(t, http.StatusOK).list(t)
	if len(parent) < len(filtered) {
		t.Fatalf("expected parentTagIds to include descendant terminals, got %d < %d", len(parent), len(filtered))
	}
	named := callAPI(t, srv, http.MethodGet, query("/fleet/hos/logs", mergeParams(window, map[string]string{
		"driverIds": fixtureDriverJordan + ",9999999",
	})), nil).expect(t, http.StatusOK).list(t)
	if got := strings.Join(driverIDsFromRecords(t, named), ","); got != fixtureDriverJordan {
		t.Fatalf("expected only the known driver, got %s", got)
	}
	none := callAPI(t, srv, http.MethodGet, query("/fleet/hos/logs", mergeParams(window, map[string]string{
		"tagIds": "1",
	})), nil).expect(t, http.StatusOK).list(t)
	if len(none) != 0 {
		t.Fatalf("expected unknown tag to match nothing, got %d", len(none))
	}
}

func mergeParams(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}

func TestHOSClocksFiltersLimitAndDeactivatedDrivers(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})

	all := callAPI(t, srv, http.MethodGet, "/fleet/hos/clocks", nil).expect(t, http.StatusOK).list(t)
	if len(all) != 12 {
		t.Fatalf("expected clocks for the 12 active fixture drivers, got %d", len(all))
	}
	for _, record := range all {
		if vehicle := mapOf(record["currentVehicle"]); vehicle != nil {
			if _, ok := vehicle["ExternalIds"]; !ok {
				t.Fatalf("expected currentVehicle ExternalIds, got %v", vehicle)
			}
		}
	}

	page := callAPI(t, srv, http.MethodGet, "/fleet/hos/clocks?limit=5", nil).expect(t, http.StatusOK)
	if len(page.list(t)) != 5 || page.pagination(t)["hasNextPage"] != true {
		t.Fatalf("expected a 5-record first page, got %s", page.Body)
	}
	callAPI(t, srv, http.MethodGet, "/fleet/hos/clocks?limit=513", nil).
		expectError(t, http.StatusBadRequest, "limit")
	callAPI(t, srv, http.MethodGet, "/fleet/hos/clocks?limit=0", nil).
		expectError(t, http.StatusBadRequest, "limit")

	tagged := taggedDriverIDs(t, srv, tagDFWTerminal)
	filtered := callAPI(t, srv, http.MethodGet, "/fleet/hos/clocks?tagIds="+tagDFWTerminal, nil).
		expect(t, http.StatusOK).list(t)
	if got := strings.Join(driverIDsFromRecords(t, filtered), ","); got != strings.Join(tagged, ",") {
		t.Fatalf("expected clocks for %v, got %s", tagged, got)
	}

	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverCameron, map[string]any{
		"driverActivationStatus": "deactivated",
	}).expect(t, http.StatusOK)
	active := callAPI(t, srv, http.MethodGet, "/fleet/hos/clocks", nil).expect(t, http.StatusOK).list(t)
	for _, id := range driverIDsFromRecords(t, active) {
		if id == fixtureDriverCameron {
			t.Fatal("expected a deactivated driver to drop out of the default clocks listing")
		}
	}
	named := callAPI(t, srv, http.MethodGet, "/fleet/hos/clocks?driverIds="+fixtureDriverCameron, nil).
		expect(t, http.StatusOK).list(t)
	if len(named) != 1 {
		t.Fatalf("expected a deactivated driver named in driverIds to be returned, got %d", len(named))
	}
}

func TestHOSViolationsSpecFilters(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	window := map[string]string{
		"startTime": fleetTestTime.Add(-14 * 24 * time.Hour).Format(time.RFC3339),
		"endTime":   fleetTestTime.Format(time.RFC3339),
	}

	callAPI(t, srv, http.MethodGet, query("/fleet/hos/violations", mergeParams(window, map[string]string{
		"types": "shiftHours,speeding",
	})), nil).expectError(t, http.StatusBadRequest, "types")
	callAPI(t, srv, http.MethodGet, query("/fleet/hos/violations", map[string]string{
		"startTime": "2026-03-04T10:00:00Z",
		"endTime":   "2026-03-04T09:00:00Z",
	}), nil).expectError(t, http.StatusBadRequest, "endTime")
	callAPI(t, srv, http.MethodGet, query("/fleet/hos/violations", mergeParams(window, map[string]string{
		"types": "NONE,unsubmittedLogs,shiftHours",
	})), nil).expect(t, http.StatusOK)

	all := mustReadHOSViolations(t, callAPI(t, srv, http.MethodGet, query("/fleet/hos/violations", window), nil).
		expect(t, http.StatusOK).Body)
	if len(all) == 0 {
		t.Fatal("expected violations across a two-week window")
	}
	for _, violation := range all {
		description := stringValue(violation, "description")
		switch stringValue(violation, "type") {
		case hosViolationTypeRestBreak:
			if description != "Rest Break Missed (8 hours)" {
				t.Fatalf("unexpected rest break description %q", description)
			}
		default:
			if !strings.Contains(description, "hours)") || !strings.Contains(description, "-") {
				t.Fatalf("expected [description] ([region]-[hours] hours), got %q", description)
			}
		}
		if _, ok := mapOf(violation["driver"])["externalIds"]; !ok {
			t.Fatalf("expected driver externalIds, got %v", violation["driver"])
		}
	}

	byExternal := mustReadHOSViolations(t, callAPI(t, srv, http.MethodGet, query("/fleet/hos/violations", mergeParams(window, map[string]string{
		"driverIds": "workerId:worker-1001",
	})), nil).expect(t, http.StatusOK).Body)
	for _, violation := range byExternal {
		if nestedString(violation, "driver", "id") != fixtureDriverAlex {
			t.Fatalf("expected external ID filter to select Alex, got %v", violation["driver"])
		}
	}

	tagged := toStringSet(taggedDriverIDs(t, srv, tagDFWTerminal))
	byTag := mustReadHOSViolations(t, callAPI(t, srv, http.MethodGet, query("/fleet/hos/violations", mergeParams(window, map[string]string{
		"tagIds": tagDFWTerminal,
	})), nil).expect(t, http.StatusOK).Body)
	for _, violation := range byTag {
		if _, ok := tagged[nestedString(violation, "driver", "id")]; !ok {
			t.Fatalf("expected only DFW drivers, got %v", violation["driver"])
		}
	}
}

func TestHOSViolationDescriptionsFollowDriverRuleset(t *testing.T) {
	limits := hosRulesetLimitsFor(Record{
		fieldEldSettings: map[string]any{"rulesets": []any{map[string]any{
			"cycle": "USA 70 hour / 8 day",
			"shift": "Texas Intrastate",
		}}},
	})
	if got := hosViolationDescription(hosViolationTypeShiftDriving, limits); got != "Shift Driving Hours (Texas Intrastate-12 hours)" {
		t.Fatalf("unexpected Texas description %q", got)
	}
	sixty := hosRulesetLimitsFor(Record{
		fieldEldSettings: map[string]any{"rulesets": []any{map[string]any{
			"cycle": "USA 60 hour / 7 day",
			"shift": "US Interstate Property",
		}}},
	})
	if got := hosViolationDescription(hosViolationTypeCycle, sixty); got != "Cycle Hours On (USA Property-60 hours)" {
		t.Fatalf("unexpected 60-hour description %q", got)
	}
}

func TestDriverLogDayUsesTimezoneAndStartHour(t *testing.T) {
	denver := Record{keyTimezone: "America/Denver"}
	start, end := driverLogDayAt(denver, time.Date(2026, 3, 4, 5, 0, 0, 0, time.UTC))
	if start.Format(time.RFC3339) != "2026-03-03T07:00:00Z" || end.Format(time.RFC3339) != "2026-03-04T07:00:00Z" {
		t.Fatalf("unexpected Denver day %s-%s", start, end)
	}
	noon := Record{keyTimezone: "America/Chicago", fieldEldDayStartHour: float64(12)}
	start, end = driverLogDayAt(noon, time.Date(2026, 3, 4, 15, 0, 0, 0, time.UTC))
	if start.Format(time.RFC3339) != "2026-03-03T18:00:00Z" || end.Format(time.RFC3339) != "2026-03-04T18:00:00Z" {
		t.Fatalf("unexpected noon-start day %s-%s", start, end)
	}
	spring, springEnd := driverLogDay(Record{keyTimezone: "America/Chicago"}, 2026, time.March, 8)
	if springEnd.Sub(spring) != 23*time.Hour {
		t.Fatalf("expected the DST day to last 23 hours, got %s", springEnd.Sub(spring))
	}
}
