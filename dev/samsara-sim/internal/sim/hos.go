package sim

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	paramDriverIDs            = "driverIds"
	paramExpand               = "expand"
	paramTypes                = "types"
	paramStartDate            = "startDate"
	paramEndDate              = "endDate"
	keyExternalIDsCapitalized = "ExternalIds"
	expandVehicle             = "vehicle"
	fieldEldDayStartHour      = "eldDayStartHour"
)

type hosDriverQuery struct {
	ExternalRefs    bool
	ActivationParam bool
}

var driverLocations sync.Map

func (v *fleetView) hosDrivers(values url.Values, query hosDriverQuery) ([]Record, error) {
	status := driverStatusActive
	if query.ActivationParam {
		parsed, err := parseActivationStatus(values)
		if err != nil {
			return nil, err
		}
		status = parsed
	}

	var named map[string]struct{}
	if refs := csvQueryValues(values, paramDriverIDs); len(refs) > 0 {
		if query.ExternalRefs {
			named = resolveRecordRefs(v.snap.drivers, refs, nil)
		} else {
			named = toStringSet(refs)
		}
	}
	tagIDs, parentTagIDs := standardTagParams(values)
	filter := v.snap.tags.filter(tagIDs, parentTagIDs)

	out := make([]Record, 0, len(v.snap.drivers))
	for _, driver := range v.snap.drivers {
		id := recordID(driver)
		if id == "" {
			continue
		}
		if named != nil {
			if _, ok := named[id]; !ok {
				continue
			}
		}
		if (named == nil || query.ActivationParam) && driverStatus(driver) != status {
			continue
		}
		if !filter.matches(v.snap.tags, tagMembersDrivers, id) {
			continue
		}
		out = append(out, driver)
	}
	sort.Slice(out, func(i, j int) bool { return recordID(out[i]) < recordID(out[j]) })
	return out, nil
}

func driverIDsOf(drivers []Record) []string {
	out := make([]string, 0, len(drivers))
	for _, driver := range drivers {
		out = append(out, recordID(driver))
	}
	return out
}

func vehicleTinyWithExternalIDs(vehicleID string, assets map[string]Record) map[string]any {
	return map[string]any{
		keyID:                     vehicleID,
		keyName:                   vehicleIDToName(vehicleID, assets),
		keyExternalIDsCapitalized: renderExternalIDs(assets[vehicleID], vehicleAutoExternalIDs),
	}
}

func driverLocation(driver Record) *time.Location {
	name := stringValue(driver, keyTimezone)
	if name == "" {
		name = defaultDriverTimezone
	}
	if cached, ok := driverLocations.Load(name); ok {
		if location, isLocation := cached.(*time.Location); isLocation {
			return location
		}
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		location = time.UTC
	}
	driverLocations.Store(name, location)
	return location
}

func driverTimezoneName(driver Record) string {
	if name := stringValue(driver, keyTimezone); name != "" {
		return name
	}
	return defaultDriverTimezone
}

func driverDayStartHour(driver Record) int {
	if hour, ok := int64Value(driver[fieldEldDayStartHour]); ok && hour == 12 {
		return 12
	}
	return 0
}

func driverLogDay(driver Record, year int, month time.Month, day int) (start, end time.Time) {
	location := driverLocation(driver)
	start = time.Date(year, month, day, driverDayStartHour(driver), 0, 0, 0, location)
	end = time.Date(year, month, day+1, driverDayStartHour(driver), 0, 0, 0, location)
	return start.UTC(), end.UTC()
}

func driverLogDayAt(driver Record, at time.Time) (start, end time.Time) {
	local := at.In(driverLocation(driver))
	shifted := local.Add(-time.Duration(driverDayStartHour(driver)) * time.Hour)
	return driverLogDay(driver, shifted.Year(), shifted.Month(), shifted.Day())
}

func (s *Server) handleHOSClockList(writer http.ResponseWriter, request *http.Request) {
	view := s.fleetView()
	drivers, err := view.hosDrivers(request.URL.Query(), hosDriverQuery{})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	records := []Record{}
	if len(drivers) > 0 {
		records = s.live.HOSClocks(view.now, driverIDsOf(drivers))
	}
	s.respondPage(writer, request, records, "|hos-clocks")
}

func (s *Server) handleHOSLogList(writer http.ResponseWriter, request *http.Request) {
	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	windowStart, windowEnd := view.now, view.now
	if startTime != nil {
		windowStart = *startTime
	}
	if endTime != nil {
		windowEnd = *endTime
	}
	if windowEnd.Before(windowStart) {
		s.writeError(writer, invalidParameter(fieldEndTime, "must not be before startTime"))
		return
	}
	drivers, err := view.hosDrivers(request.URL.Query(), hosDriverQuery{})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	records := []Record{}
	if len(drivers) > 0 {
		records = s.live.HOSLogs(view.now, driverIDsOf(drivers), &windowStart, &windowEnd)
	}
	s.respondPage(writer, request, records, "|hos-logs")
}

func parseExpand(values url.Values) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range csvQueryValues(values, paramExpand) {
		out[strings.TrimSpace(value)] = struct{}{}
	}
	return out
}
