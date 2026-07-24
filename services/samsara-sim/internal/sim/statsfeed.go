package sim

import (
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	statTypeGPS               = "gps"
	statTypeEngineStates      = "engineStates"
	statTypeFuelPercents      = "fuelPercents"
	statTypeObdOdometerMeters = "obdOdometerMeters"
	statTypeEcuSpeedMph       = "ecuSpeedMph"
	statTypeBatteryMilliVolts = "batteryMilliVolts"

	maxFeedSamplesPerVehicle  = maxAssetSamplesPerAsset
	maxFeedRecordsPerResponse = 512
)

func supportedStatTypes() map[string]struct{} {
	return map[string]struct{}{
		statTypeGPS:               {},
		statTypeEngineStates:      {},
		statTypeFuelPercents:      {},
		statTypeObdOdometerMeters: {},
		statTypeEcuSpeedMph:       {},
		statTypeBatteryMilliVolts: {},
	}
}

func parseStatTypes(request *http.Request) ([]string, error) {
	raw := queryValue(request, "types")
	if raw == "" {
		return nil, ErrStatTypesRequired
	}

	supported := supportedStatTypes()
	types := splitCSV(raw)
	if len(types) == 0 {
		return nil, ErrStatTypesRequired
	}
	out := make([]string, 0, len(types))
	for _, statType := range types {
		if _, ok := supported[statType]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrStatTypeInvalid, statType)
		}
		out = append(out, statType)
	}
	return uniqueStrings(out), nil
}

func encodeStatsFeedCursor(at time.Time, offset int) string {
	raw := fmt.Sprintf("t=%d;i=%d", at.UTC().UnixMilli(), offset)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func decodeStatsFeedCursor(cursor string) (time.Time, int, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return time.Time{}, 0, ErrCursorInvalid
	}

	unixMilli := int64(-1)
	offset := 0
	for _, part := range strings.Split(string(decoded), ";") {
		key, value, found := strings.Cut(part, "=")
		if !found {
			return time.Time{}, 0, ErrCursorInvalid
		}
		switch key {
		case "t":
			parsed, parseErr := strconv.ParseInt(value, 10, 64)
			if parseErr != nil || parsed < 0 {
				return time.Time{}, 0, ErrCursorInvalid
			}
			unixMilli = parsed
		case "i":
			parsed, parseErr := strconv.Atoi(value)
			if parseErr != nil || parsed < 0 {
				return time.Time{}, 0, ErrCursorInvalid
			}
			offset = parsed
		default:
			return time.Time{}, 0, ErrCursorInvalid
		}
	}
	if unixMilli < 0 {
		return time.Time{}, 0, ErrCursorInvalid
	}
	return time.UnixMilli(unixMilli).UTC(), offset, nil
}

func statsFeedSampleTimes(
	cursorTime time.Time,
	gridNow time.Time,
	vehicleCount int,
	totalCap int,
) (times []time.Time, hasNext bool) {
	if !gridNow.After(cursorTime) {
		return []time.Time{}, false
	}

	available := int(gridNow.Sub(cursorTime) / defaultAssetSampleStep)
	if available <= 0 {
		return []time.Time{}, false
	}
	perVehicle := available
	if perVehicle > maxFeedSamplesPerVehicle {
		perVehicle = maxFeedSamplesPerVehicle
	}
	if totalCap > 0 && vehicleCount > 0 && perVehicle*vehicleCount > totalCap {
		perVehicle = totalCap / vehicleCount
		if perVehicle < 1 {
			perVehicle = 1
		}
	}

	times = make([]time.Time, 0, perVehicle)
	for idx := 1; idx <= perVehicle; idx++ {
		times = append(times, cursorTime.Add(time.Duration(idx)*defaultAssetSampleStep).UTC())
	}
	return times, perVehicle < available
}

func (l *LiveSimulator) FeedVehicleIDs(vehicleIDs []string) []string {
	return selectVehicleIDs(
		vehicleIDs,
		l.loadVehicleStatsTemplates(),
		l.loadAssetWaypoints(),
		l.loadAssetMetadata(),
	)
}

func (l *LiveSimulator) VehicleStatsFeed(
	now time.Time,
	vehicleIDs []string,
	statTypes []string,
	feedSampleTimes []time.Time,
) []Record {
	if len(vehicleIDs) == 0 || len(feedSampleTimes) == 0 {
		return []Record{}
	}

	templates := l.loadVehicleStatsTemplates()
	waypoints := l.loadAssetWaypoints()
	assets := l.loadAssetMetadata()
	driverByVehicle := l.driverByVehicleMap()
	requested := toStringSet(statTypes)
	eventsByVehicle := l.vehicleEventsForWindow(
		vehicleIDs,
		driverByVehicle,
		feedSampleTimes[0].Add(-2*time.Hour),
		feedSampleTimes[len(feedSampleTimes)-1].Add(2*time.Hour),
	)

	out := make([]Record, 0, len(vehicleIDs))
	for _, vehicleID := range vehicleIDs {
		base := templates[vehicleID]
		record := Record{
			"id":   vehicleID,
			"name": vehicleName(vehicleID, base, assets),
		}
		states := l.feedRouteStates(
			vehicleID,
			waypoints[vehicleID],
			eventsByVehicle[vehicleID],
			feedSampleTimes,
			now,
			requested,
		)
		l.applyFeedStatGroups(record, base, vehicleID, states, feedSampleTimes, requested)
		out = append(out, record)
	}

	sort.Slice(out, func(i, j int) bool {
		return recordID(out[i]) < recordID(out[j])
	})
	return out
}

func (l *LiveSimulator) feedRouteStates(
	vehicleID string,
	points []routePoint,
	events []SimEvent,
	feedSampleTimes []time.Time,
	now time.Time,
	requested map[string]struct{},
) []routeState {
	if len(points) == 0 {
		return nil
	}
	if !statGroupRequiresRouteState(requested) {
		return nil
	}

	states := make([]routeState, 0, len(feedSampleTimes))
	for _, sampleTime := range feedSampleTimes {
		state := l.routeStateForSample(vehicleID, points, sampleTime, now, now)
		state = l.applyVehicleEventsToRouteState(
			vehicleID,
			points,
			events,
			sampleTime,
			now,
			now,
			state,
		)
		states = append(states, state)
	}
	return states
}

func statGroupRequiresRouteState(requested map[string]struct{}) bool {
	for _, statType := range []string{statTypeGPS, statTypeEngineStates, statTypeEcuSpeedMph} {
		if _, ok := requested[statType]; ok {
			return true
		}
	}
	return false
}

func (l *LiveSimulator) applyFeedStatGroups(
	record Record,
	base Record,
	vehicleID string,
	states []routeState,
	feedSampleTimes []time.Time,
	requested map[string]struct{},
) {
	hashFactor := l.hashFraction(vehicleID)
	if _, ok := requested[statTypeGPS]; ok {
		record[statTypeGPS] = feedGPSSamples(states, feedSampleTimes)
	}
	if _, ok := requested[statTypeEngineStates]; ok {
		record[statTypeEngineStates] = feedEngineStateSamples(states, feedSampleTimes)
	}
	if _, ok := requested[statTypeEcuSpeedMph]; ok {
		record[statTypeEcuSpeedMph] = feedEcuSpeedSamples(states, feedSampleTimes)
	}
	if _, ok := requested[statTypeFuelPercents]; ok {
		record[statTypeFuelPercents] = feedScalarSamples(
			feedSampleTimes,
			func(at time.Time) any {
				return dynamicFuelPercent(base, at, l.startedAt, hashFactor)
			},
		)
	}
	if _, ok := requested[statTypeObdOdometerMeters]; ok {
		record[statTypeObdOdometerMeters] = feedScalarSamples(
			feedSampleTimes,
			func(at time.Time) any {
				return dynamicOdometerMeters(base, at, l.startedAt, hashFactor)
			},
		)
	}
	if _, ok := requested[statTypeBatteryMilliVolts]; ok {
		record[statTypeBatteryMilliVolts] = feedScalarSamples(
			feedSampleTimes,
			func(at time.Time) any {
				return dynamicBatteryMilliVolts(base, at, hashFactor)
			},
		)
	}
}

func feedGPSSamples(states []routeState, feedSampleTimes []time.Time) []any {
	out := make([]any, 0, len(states))
	for idx := range states {
		state := &states[idx]
		sample := map[string]any{
			"time":              feedSampleTimes[idx].UTC().Format(time.RFC3339),
			"latitude":          round(state.Latitude, 6),
			"longitude":         round(state.Longitude, 6),
			"headingDegrees":    int64(math.Round(state.Heading)),
			"speedMilesPerHour": round(state.SpeedMPS*2.23694, 2),
			"isEcuSpeed":        false,
		}
		if formatted := formattedLocationFromAddress(state.Address); formatted != "" {
			sample["reverseGeo"] = map[string]any{
				"formattedLocation": formatted,
			}
		}
		out = append(out, sample)
	}
	return out
}

func feedEngineStateSamples(states []routeState, feedSampleTimes []time.Time) []any {
	out := make([]any, 0, len(states))
	for idx := range states {
		out = append(out, map[string]any{
			"time":  feedSampleTimes[idx].UTC().Format(time.RFC3339),
			"value": ternary(states[idx].SpeedMPS > movingSpeedThresholdMPS, "On", "Off"),
		})
	}
	return out
}

func feedEcuSpeedSamples(states []routeState, feedSampleTimes []time.Time) []any {
	out := make([]any, 0, len(states))
	for idx := range states {
		out = append(out, map[string]any{
			"time":  feedSampleTimes[idx].UTC().Format(time.RFC3339),
			"value": round(states[idx].SpeedMPS*2.23694*0.985, 2),
		})
	}
	return out
}

func feedScalarSamples(feedSampleTimes []time.Time, valueAt func(time.Time) any) []any {
	out := make([]any, 0, len(feedSampleTimes))
	for _, sampleTime := range feedSampleTimes {
		out = append(out, map[string]any{
			"time":  sampleTime.UTC().Format(time.RFC3339),
			"value": valueAt(sampleTime),
		})
	}
	return out
}

func formattedLocationFromAddress(address map[string]any) string {
	if len(address) == 0 {
		return ""
	}
	record := Record(address)
	if formatted := stringValue(record, "formattedAddress"); formatted != "" {
		return formatted
	}

	street := strings.TrimSpace(strings.Join([]string{
		stringValue(record, "streetNumber"),
		stringValue(record, "street"),
	}, " "))
	parts := make([]string, 0, 3)
	for _, part := range []string{street, stringValue(record, "city"), stringValue(record, "state")} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ", ")
}

func (s *Server) handleVehicleStatsFeed(writer http.ResponseWriter, request *http.Request) {
	statTypes, err := parseStatTypes(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	totalCap, err := parseLimitStrict(request.URL.Query(), maxFeedRecordsPerResponse)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	vehicleIDs := idsFromQuery(request.URL.Query(), "vehicleIds")
	now := s.simNow()
	gridNow := now.Truncate(defaultAssetSampleStep)
	s.dispatchLiveEvents(request, now, vehicleIDs)

	if s.live == nil {
		s.respondStatsFeed(writer, request, []Record{}, encodeStatsFeedCursor(gridNow, 0), false)
		return
	}

	after := queryValue(request, "after")
	if after == "" {
		records := s.live.VehicleStatsFeed(
			now,
			s.live.FeedVehicleIDs(vehicleIDs),
			statTypes,
			[]time.Time{gridNow},
		)
		s.respondStatsFeed(writer, request, records, encodeStatsFeedCursor(gridNow, 0), false)
		return
	}

	cursorTime, _, err := decodeStatsFeedCursor(after)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	feedIDs := s.live.FeedVehicleIDs(vehicleIDs)
	times, hasNext := statsFeedSampleTimes(cursorTime, gridNow, len(feedIDs), totalCap)
	if len(times) == 0 {
		s.respondStatsFeed(writer, request, []Record{}, after, false)
		return
	}

	records := s.live.VehicleStatsFeed(now, feedIDs, statTypes, times)
	endCursor := encodeStatsFeedCursor(times[len(times)-1], 0)
	s.respondStatsFeed(writer, request, records, endCursor, hasNext)
}

func (s *Server) respondStatsFeed(
	writer http.ResponseWriter,
	request *http.Request,
	records []Record,
	endCursor string,
	hasNextPage bool,
) {
	payload := map[string]any{
		"data": recordsAsAny(records),
		"pagination": map[string]any{
			"endCursor":   endCursor,
			"hasNextPage": hasNextPage,
		},
	}
	s.respondJSON(writer, request, requestSignature(request)+"|vehicle-stats-feed", payload)
}

func (s *Server) handleVehicleStatsHistory(writer http.ResponseWriter, request *http.Request) {
	statTypes, err := parseStatTypes(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	if startTime == nil || endTime == nil {
		s.writeAPIError(writer, http.StatusBadRequest, ErrTimeRangeRequired)
		return
	}

	vehicleIDs := idsFromQuery(request.URL.Query(), "vehicleIds")
	now := s.simNow()

	records := []Record{}
	if s.live != nil {
		times := sampleTimes(*startTime, *endTime, defaultAssetSampleStep, maxFeedSamplesPerVehicle)
		records = s.live.VehicleStatsFeed(now, s.live.FeedVehicleIDs(vehicleIDs), statTypes, times)
	}

	page, pagination, err := paginate(records, request.URL.Query(), maxFeedRecordsPerResponse)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|vehicle-stats-history", payload)
}
