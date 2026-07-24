package sim

import (
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	dailyLogDateLayout         = "2006-01-02"
	dailyLogDefaultRangeDays   = 7
	dailyLogMaxRangeDays       = 30
	dailyLogCarrierName        = "Trenova Logistics"
	dailyLogCarrierUsDotNumber = int64(1234567)
	dailyLogDriverTimezone     = "America/Chicago"
	dailyLogCertifyGrace       = 24 * time.Hour
	dailyLogCertifyRate        = 0.85
)

var dailyLogHomeTerminals = []string{
	"Austin Terminal",
	"Dallas Terminal",
	"Houston Terminal",
}

type dailyLogDriverContext struct {
	DriverID   string
	DriverName string
	VehicleID  string
	Timeline   []timelineSegment
	Events     []SimEvent
	Geometry   *routeGeometry
	Assets     map[string]Record
}

func (s *Server) handleHOSDailyLogList(writer http.ResponseWriter, request *http.Request) {
	now := s.simNow()
	startDate, endDate, err := parseDailyLogDateRange(request, now)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	driverIDs := idsFromQuery(request.URL.Query(), "driverIds")
	records := []Record{}
	if s.live != nil {
		records = s.live.HOSDailyLogs(now, driverIDs, startDate, endDate)
	}

	page, pagination, err := paginate(records, request.URL.Query(), 512)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|hos-daily-logs", payload)
}

func parseDailyLogDateRange(
	request *http.Request,
	now time.Time,
) (startDate, endDate time.Time, err error) {
	endDate = now.UTC().Truncate(24 * time.Hour)
	rawEnd := queryValue(request, "endDate")
	if rawEnd != "" {
		parsed, parseErr := time.Parse(dailyLogDateLayout, rawEnd)
		if parseErr != nil {
			return time.Time{}, time.Time{}, ErrInvalidBody
		}
		endDate = parsed.UTC()
	}

	startDate = endDate.Add(-time.Duration(dailyLogDefaultRangeDays-1) * 24 * time.Hour)
	rawStart := queryValue(request, "startDate")
	if rawStart != "" {
		parsed, parseErr := time.Parse(dailyLogDateLayout, rawStart)
		if parseErr != nil {
			return time.Time{}, time.Time{}, ErrInvalidBody
		}
		startDate = parsed.UTC()
	}

	if endDate.Before(startDate) {
		return time.Time{}, time.Time{}, ErrInvalidBody
	}
	if endDate.Sub(startDate) > dailyLogMaxRangeDays*24*time.Hour {
		return time.Time{}, time.Time{}, ErrInvalidBody
	}
	return startDate, endDate, nil
}

func (l *LiveSimulator) HOSDailyLogs(
	now time.Time,
	driverIDs []string,
	startDate time.Time,
	endDate time.Time,
) []Record {
	now = now.UTC()
	startDate = startDate.UTC().Truncate(24 * time.Hour)
	endDate = endDate.UTC().Truncate(24 * time.Hour)
	if endDate.Before(startDate) {
		return []Record{}
	}

	roster := l.loadDriverRoster()
	ids := selectDriverIDs(driverIDs, map[string]Record{}, roster)
	if len(ids) == 0 {
		return []Record{}
	}

	assets := l.loadAssetMetadata()
	waypoints := l.loadAssetWaypoints()
	geometryCache := map[string]*routeGeometry{}
	dayCount := int(endDate.Sub(startDate)/(24*time.Hour)) + 1

	out := make([]Record, 0, len(ids)*dayCount)
	for _, driverID := range ids {
		entry := roster[driverID]
		vehicleID := strings.TrimSpace(entry.VehicleID)
		ctx := dailyLogDriverContext{
			DriverID:   driverID,
			DriverName: firstNonEmpty(entry.Name, driverID),
			VehicleID:  vehicleID,
			Timeline: l.driverTimelineSegments(
				driverID,
				startDate.Add(-24*time.Hour),
				endDate.Add(48*time.Hour),
				now,
			),
			Events: l.pairEventsInWindow(
				driverID,
				vehicleID,
				startDate.Add(-2*time.Hour),
				endDate.Add(26*time.Hour),
			),
			Geometry: l.cachedRouteGeometry(geometryCache, waypoints, vehicleID),
			Assets:   assets,
		}
		for day := endDate; !day.Before(startDate); day = day.Add(-24 * time.Hour) {
			if day.After(now) {
				continue
			}
			out = append(out, l.dailyLogRecord(&ctx, day, now))
		}
	}

	sort.Slice(out, func(i, j int) bool {
		left := nestedString(out[i], "driver", "id")
		right := nestedString(out[j], "driver", "id")
		if left != right {
			return left < right
		}
		return stringValue(out[i], "startTime") > stringValue(out[j], "startTime")
	})
	return out
}

func (l *LiveSimulator) cachedRouteGeometry(
	cache map[string]*routeGeometry,
	waypoints map[string][]routePoint,
	vehicleID string,
) *routeGeometry {
	if vehicleID == "" {
		return nil
	}
	if geometry, ok := cache[vehicleID]; ok {
		return geometry
	}
	geometry := l.routeGeometryForAsset(vehicleID, waypoints[vehicleID])
	cache[vehicleID] = &geometry
	return &geometry
}

func (l *LiveSimulator) dailyLogRecord(
	ctx *dailyLogDriverContext,
	day time.Time,
	now time.Time,
) Record {
	dayCtx := l.buildDailyEventContext(ctx.DriverID, ctx.VehicleID, day)
	shiftStart := l.shiftStartForDay(ctx.DriverID, day).Truncate(time.Second)
	dayEnd := dayCtx.DayEnd.Truncate(time.Second)
	effectiveEnd := minTime(dayEnd, now).Truncate(time.Second)
	if effectiveEnd.Before(shiftStart) {
		effectiveEnd = shiftStart
	}

	durations := map[string]time.Duration{}
	driveMeters := 0.0
	for idx := range ctx.Timeline {
		segment := &ctx.Timeline[idx]
		overlapStart := maxTime(segment.Start, shiftStart)
		overlapEnd := minTime(segment.End, effectiveEnd)
		if !overlapEnd.After(overlapStart) {
			continue
		}
		status := l.dailyLogSegmentStatus(ctx, segment, now)
		durations[status] += overlapEnd.Sub(overlapStart)
		if status == hosStatusDriving {
			driveMeters += l.integrateDriveDistance(ctx, overlapStart, overlapEnd, shiftStart, now)
		}
	}

	durationsPayload := dailyLogDurationsPayload(durations)
	return Record{
		"driver": map[string]any{
			"id":       ctx.DriverID,
			"name":     ctx.DriverName,
			"timezone": dailyLogDriverTimezone,
		},
		"startTime": shiftStart.UTC().Format(time.RFC3339),
		"endTime":   effectiveEnd.UTC().Format(time.RFC3339),
		"distanceTraveled": map[string]any{
			"driveDistanceMeters":              int64(math.Round(driveMeters)),
			"personalConveyanceDistanceMeters": int64(0),
			"yardMoveDistanceMeters":           int64(0),
		},
		"dutyStatusDurations":        durationsPayload,
		"pendingDutyStatusDurations": cloneMap(durationsPayload),
		"logMetaData":                l.dailyLogMetadata(ctx, day, dayEnd, now),
	}
}

func (l *LiveSimulator) dailyLogSegmentStatus(
	ctx *dailyLogDriverContext,
	segment *timelineSegment,
	now time.Time,
) string {
	status := normalizeDutyStatusForVehicle(
		segment.Status,
		ctx.VehicleID,
		l.dailyLogVehicleMovingAt(ctx, segment.Start, now),
	)
	if primaryEvent := pickPrimarySimEventAt(ctx.Events, segment.Start); primaryEvent != nil {
		status = dutyStatusForSimEvent(primaryEvent, status)
	}
	return status
}

func (l *LiveSimulator) dailyLogVehicleMovingAt(
	ctx *dailyLogDriverContext,
	at time.Time,
	now time.Time,
) bool {
	if ctx.VehicleID == "" || ctx.Geometry == nil || len(ctx.Geometry.Points) == 0 {
		return false
	}
	windowStart := at.Add(-time.Minute)
	state := l.routeStateForGeometry(ctx.VehicleID, ctx.Geometry, at, windowStart, now)
	state = l.applyVehicleEventsToGeometryState(
		ctx.VehicleID,
		ctx.Geometry,
		ctx.Events,
		at,
		windowStart,
		now,
		state,
	)
	return state.SpeedMPS > movingSpeedThresholdMPS
}

func (l *LiveSimulator) integrateDriveDistance(
	ctx *dailyLogDriverContext,
	start time.Time,
	end time.Time,
	windowStart time.Time,
	now time.Time,
) float64 {
	if ctx.VehicleID == "" || ctx.Geometry == nil || len(ctx.Geometry.Points) == 0 {
		return 0
	}

	total := 0.0
	for cursor := start; cursor.Before(end); cursor = cursor.Add(defaultAssetSampleStep) {
		step := defaultAssetSampleStep
		if remaining := end.Sub(cursor); remaining < step {
			step = remaining
		}
		state := l.routeStateForGeometry(ctx.VehicleID, ctx.Geometry, cursor, windowStart, now)
		state = l.applyVehicleEventsToGeometryState(
			ctx.VehicleID,
			ctx.Geometry,
			ctx.Events,
			cursor,
			windowStart,
			now,
			state,
		)
		total += state.SpeedMPS * step.Seconds()
	}
	return total
}

func dailyLogDurationsPayload(durations map[string]time.Duration) map[string]any {
	driveMs := float64(durations[hosStatusDriving].Milliseconds())
	onDutyMs := float64(durations[hosStatusOnDuty].Milliseconds())
	offDutyMs := float64(durations[hosStatusOffDuty].Milliseconds())
	sleeperMs := float64(durations[hosStatusSleeperBed].Milliseconds())
	return map[string]any{
		"activeDurationMs":             driveMs + onDutyMs,
		"driveDurationMs":              driveMs,
		"onDutyDurationMs":             onDutyMs,
		"offDutyDurationMs":            offDutyMs,
		"sleeperBerthDurationMs":       sleeperMs,
		"personalConveyanceDurationMs": float64(0),
		"yardMoveDurationMs":           float64(0),
		"waitingTimeDurationMs":        float64(0),
	}
}

func (l *LiveSimulator) dailyLogMetadata(
	ctx *dailyLogDriverContext,
	day time.Time,
	dayEnd time.Time,
	now time.Time,
) map[string]any {
	vehicles := []any{}
	if ctx.VehicleID != "" {
		vehicles = append(vehicles, map[string]any{
			"id":   ctx.VehicleID,
			"name": vehicleIDToName(ctx.VehicleID, ctx.Assets),
		})
	}

	metadata := map[string]any{
		"isCertified":        false,
		"carrierName":        dailyLogCarrierName,
		"carrierUsDotNumber": dailyLogCarrierUsDotNumber,
		"homeTerminalName":   l.dailyLogHomeTerminal(ctx.DriverID),
		"shippingDocs":       dailyLogShippingDoc(ctx.DriverID, day),
		"trailerNames":       []any{},
		"vehicles":           vehicles,
	}
	certified, certifiedAt := l.dailyLogCertification(ctx.DriverID, day, dayEnd, now)
	if certified {
		metadata["isCertified"] = true
		metadata["certifiedAtTime"] = certifiedAt.UTC().Format(time.RFC3339)
	}
	return metadata
}

func (l *LiveSimulator) dailyLogCertification(
	driverID string,
	day time.Time,
	dayEnd time.Time,
	now time.Time,
) (bool, time.Time) {
	dayKey := day.UTC().Format(dailyLogDateLayout)
	certified := now.Sub(dayEnd) > dailyLogCertifyGrace
	if !certified {
		certified = l.hashFraction("daily-log-certified", driverID, dayKey) < dailyLogCertifyRate
	}
	if !certified {
		return false, time.Time{}
	}

	offsetMinutes := 30 + 90*l.hashFraction("daily-log-certified-at", driverID, dayKey)
	certifiedAt := dayEnd.Add(time.Duration(offsetMinutes * float64(time.Minute)))
	return true, minTime(certifiedAt, now)
}

func (l *LiveSimulator) dailyLogHomeTerminal(driverID string) string {
	index := int(float64(len(dailyLogHomeTerminals)) * l.hashFraction("daily-log-terminal", driverID))
	if index >= len(dailyLogHomeTerminals) {
		index = len(dailyLogHomeTerminals) - 1
	}
	return dailyLogHomeTerminals[index]
}

func dailyLogShippingDoc(driverID string, day time.Time) string {
	var builder strings.Builder
	builder.Grow(len(driverID))
	for _, char := range strings.ToUpper(driverID) {
		if (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
		}
	}
	reference := builder.String()
	if reference == "" {
		reference = "DRV"
	}
	return "SD-" + day.UTC().Format("20060102") + "-" + reference
}
