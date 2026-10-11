package sim

import (
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	dailyLogDateLayout          = "2006-01-02"
	dailyLogCarrierName         = "Trenova Logistics"
	dailyLogCarrierUsDotNumber  = int64(1234567)
	dailyLogCarrierAddress      = "100 Fleet Ave, Austin, TX 78701"
	dailyLogDefaultTerminalName = "Austin Terminal"
	dailyLogCertifyGrace        = 24 * time.Hour
	dailyLogCertifyRate         = 0.85
)

var dailyLogTerminalAddresses = map[string]string{
	"Austin Terminal":            "100 Fleet Ave, Austin, TX 78701",
	"San Antonio Terminal":       "4100 SE Loop 410, San Antonio, TX 78222",
	"Dallas-Fort Worth Terminal": "200 Warehouse Rd, Dallas, TX 75201",
	"Houston Terminal":           "860 Bay Area Blvd, Houston, TX 77058",
	"Rio Grande Valley Terminal": "3500 N 23rd St, McAllen, TX 78501",
	"El Paso Terminal":           "9400 Gateway Blvd N, El Paso, TX 79924",
	"Permian Basin Terminal":     "2700 W Interstate 20, Odessa, TX 79763",
	"Panhandle Terminal":         "5100 E Amarillo Blvd, Amarillo, TX 79107",
}

type dailyLogDriverContext struct {
	DriverID  string
	Driver    Record
	VehicleID string
	Timeline  []timelineSegment
	Events    []SimEvent
	Geometry  *routeGeometry
	Assets    map[string]Record
	Expand    map[string]struct{}
}

type dailyLogKey struct {
	Driver Record
	Start  time.Time
	End    time.Time
	Date   string
}

func (s *Server) handleHOSDailyLogList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	startDate, endDate, err := parseDailyLogDateRange(values)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	drivers, err := view.hosDrivers(values, hosDriverQuery{ExternalRefs: true, ActivationParam: true})
	if err != nil {
		s.writeError(writer, err)
		return
	}

	keys := dailyLogKeys(drivers, startDate, endDate, view.now)
	index := make(map[string]dailyLogKey, len(keys))
	placeholders := make([]Record, 0, len(keys))
	for _, key := range keys {
		placeholder := Record{
			keyDriver:      map[string]any{keyID: recordID(key.Driver)},
			fieldStartTime: key.Start.Format(time.RFC3339),
		}
		index[recordIdentityKey(placeholder)] = key
		placeholders = append(placeholders, placeholder)
	}
	page, pagination, err := paginate(placeholders, request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	pageKeys := make([]dailyLogKey, 0, len(page))
	for _, placeholder := range page {
		pageKeys = append(pageKeys, index[recordIdentityKey(placeholder)])
	}
	records := s.live.dailyLogRecords(pageKeys, view.now, parseExpand(values))
	payload := map[string]any{keyData: recordsAsAny(records), keyPagination: pagination}
	s.respondJSON(writer, request, requestSignature(request)+"|hos-daily-logs", payload)
}

func parseDailyLogDate(values url.Values, name string) (time.Time, error) {
	raw := strings.TrimSpace(values.Get(name))
	if raw == "" {
		return time.Time{}, invalidParameter(name, "is required")
	}
	parsed, err := time.Parse(dailyLogDateLayout, raw)
	if err != nil {
		return time.Time{}, invalidParameter(name, "must be a YYYY-MM-DD date")
	}
	return parsed, nil
}

func parseDailyLogDateRange(values url.Values) (startDate, endDate time.Time, err error) {
	startDate, err = parseDailyLogDate(values, paramStartDate)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endDate, err = parseDailyLogDate(values, paramEndDate)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if endDate.Before(startDate) {
		return time.Time{}, time.Time{}, invalidParameter(
			paramEndDate,
			"must be greater than or equal to startDate",
		)
	}
	return startDate, endDate, nil
}

func dailyLogKeys(drivers []Record, startDate, endDate, now time.Time) []dailyLogKey {
	dayCount := int(endDate.Sub(startDate)/(24*time.Hour)) + 1
	out := make([]dailyLogKey, 0, len(drivers)*dayCount)
	for _, driver := range drivers {
		for date := endDate; !date.Before(startDate); date = date.AddDate(0, 0, -1) {
			start, end := driverLogDay(driver, date.Year(), date.Month(), date.Day())
			if start.After(now) {
				continue
			}
			out = append(out, dailyLogKey{
				Driver: driver,
				Start:  start,
				End:    end,
				Date:   date.Format(dailyLogDateLayout),
			})
		}
	}
	return out
}

func (l *LiveSimulator) HOSDailyLogs(
	now time.Time,
	drivers []Record,
	startDate time.Time,
	endDate time.Time,
	expand map[string]struct{},
) []Record {
	return l.dailyLogRecords(dailyLogKeys(drivers, startDate, endDate, now.UTC()), now, expand)
}

func (l *LiveSimulator) dailyLogRecords(
	keys []dailyLogKey,
	now time.Time,
	expand map[string]struct{},
) []Record {
	now = now.UTC()
	if len(keys) == 0 {
		return []Record{}
	}
	roster := l.loadDriverRoster()
	assets := l.loadAssetMetadata()
	waypoints := l.loadAssetWaypoints()
	geometryCache := map[string]*routeGeometry{}
	contexts := map[string]*dailyLogDriverContext{}
	out := make([]Record, 0, len(keys))
	for idx := range keys {
		key := &keys[idx]
		driverID := recordID(key.Driver)
		ctx, ok := contexts[driverID]
		if !ok {
			first, last := key.Start, key.End
			for other := idx + 1; other < len(keys); other++ {
				if recordID(keys[other].Driver) != driverID {
					continue
				}
				first = minTime(first, keys[other].Start)
				last = maxTime(last, keys[other].End)
			}
			vehicleID := strings.TrimSpace(roster[driverID].VehicleID)
			ctx = &dailyLogDriverContext{
				DriverID:  driverID,
				Driver:    key.Driver,
				VehicleID: vehicleID,
				Timeline:  l.driverTimelineSegments(driverID, first.Add(-24*time.Hour), last.Add(24*time.Hour), now),
				Events: l.pairEventsInWindow(
					driverID,
					vehicleID,
					first.Add(-2*time.Hour),
					last.Add(2*time.Hour),
				),
				Geometry: l.cachedRouteGeometry(geometryCache, waypoints, vehicleID),
				Assets:   assets,
				Expand:   expand,
			}
			contexts[driverID] = ctx
		}
		out = append(out, l.dailyLogRecord(ctx, key, now))
	}
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
	key *dailyLogKey,
	now time.Time,
) Record {
	effectiveEnd := minTime(key.End, now)
	durations := map[string]time.Duration{}
	driveMeters := 0.0
	for idx := range ctx.Timeline {
		segment := &ctx.Timeline[idx]
		overlapStart := maxTime(segment.Start, key.Start).Truncate(time.Millisecond)
		overlapEnd := minTime(segment.End, effectiveEnd).Truncate(time.Millisecond)
		if !overlapEnd.After(overlapStart) {
			continue
		}
		status := l.dailyLogSegmentStatus(ctx, segment, now)
		durations[status] += overlapEnd.Sub(overlapStart)
		if status == hosStatusDriving {
			driveMeters += l.integrateDriveDistance(ctx, overlapStart, overlapEnd, key.Start, now)
		}
	}

	durationsPayload := dailyLogDurationsPayload(durations)
	driver := map[string]any{
		keyID:            ctx.DriverID,
		keyName:          stringValue(ctx.Driver, keyName),
		keyTimezone:      driverTimezoneName(ctx.Driver),
		fieldEldSettings: eldSettingsFor(ctx.Driver),
	}
	if stored, ok := anyAsMap(ctx.Driver[fieldEldSettings]); ok {
		driver[fieldEldSettings] = cloneMap(stored)
	}
	if externalIDs := externalIDsOf(ctx.Driver); len(externalIDs) > 0 {
		driver[fieldExternalIDs] = renderExternalIDs(ctx.Driver, nil)
	}
	return Record{
		keyDriver:      driver,
		fieldStartTime: key.Start.Format(time.RFC3339),
		fieldEndTime:   key.End.Format(time.RFC3339),
		"distanceTraveled": map[string]any{
			"driveDistanceMeters":              int64(math.Round(driveMeters)),
			"personalConveyanceDistanceMeters": int64(0),
			"yardMoveDistanceMeters":           int64(0),
		},
		"dutyStatusDurations":        durationsPayload,
		"pendingDutyStatusDurations": cloneMap(durationsPayload),
		"logMetaData":                l.dailyLogMetadata(ctx, key, now),
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
	key *dailyLogKey,
	now time.Time,
) map[string]any {
	vehicles := []any{}
	trailerNames := []any{}
	if ctx.VehicleID != "" {
		vehicles = append(vehicles, dailyLogVehicle(ctx, ctx.VehicleID))
		if trailer := coupledTrailer(ctx.Assets, ctx.VehicleID); trailer != nil {
			trailerNames = append(trailerNames, stringValue(trailer, keyName))
		}
	}

	carrier := Record(mapOf(ctx.Driver[keyCarrierSettings]))
	terminalName, terminalAddress := l.driverHomeTerminal(ctx.Driver)
	dotNumber := dailyLogCarrierUsDotNumber
	if value, ok := int64Value(carrier["dotNumber"]); ok {
		dotNumber = value
	}
	metadata := map[string]any{
		"adverseDrivingClaimed": false,
		"bigDayClaimed":         false,
		"isCertified":           false,
		"isUsShortHaulActive":   false,
		"carrierName": stringutils.FirstNonEmptyTrimmed(
			stringValue(carrier, "carrierName"),
			dailyLogCarrierName,
		),
		"carrierUsDotNumber": dotNumber,
		"carrierFormattedAddress": stringutils.FirstNonEmptyTrimmed(
			stringValue(carrier, "mainOfficeAddress"),
			dailyLogCarrierAddress,
		),
		"homeTerminalName": stringutils.FirstNonEmptyTrimmed(
			stringValue(carrier, "homeTerminalName"),
			terminalName,
		),
		"homeTerminalFormattedAddress": stringutils.FirstNonEmptyTrimmed(
			stringValue(carrier, "homeTerminalAddress"),
			terminalAddress,
		),
		"shippingDocs": dailyLogShippingDoc(ctx.DriverID, key.Start),
		"trailerNames": trailerNames,
		"vehicles":     vehicles,
	}
	certified, certifiedAt := l.dailyLogCertification(ctx.DriverID, key.Date, key.End, now)
	if certified {
		metadata["isCertified"] = true
		metadata["certifiedAtTime"] = certifiedAt.UTC().Format(time.RFC3339)
	}
	return metadata
}

func dailyLogVehicle(ctx *dailyLogDriverContext, vehicleID string) map[string]any {
	if _, expanded := ctx.Expand[expandVehicle]; !expanded {
		return map[string]any{keyID: vehicleID}
	}
	asset := ctx.Assets[vehicleID]
	out := map[string]any{
		keyID:            vehicleID,
		keyName:          vehicleIDToName(vehicleID, ctx.Assets),
		"assetType":      assetType(asset),
		fieldExternalIDs: renderExternalIDs(asset, vehicleAutoExternalIDs),
	}
	if plate := stringValue(asset, keyLicensePlate); plate != "" {
		out[keyLicensePlate] = plate
	}
	if vin := stringValue(asset, keyVIN); vin != "" {
		out["vehicleVin"] = vin
	}
	return out
}

func coupledTrailer(assets map[string]Record, vehicleID string) Record {
	var selected Record
	for _, asset := range assets {
		if assetType(asset) != assetTypeTrailer ||
			stringValue(asset, fieldSimCoupledVehicleID) != vehicleID {
			continue
		}
		if selected == nil || recordID(asset) < recordID(selected) {
			selected = asset
		}
	}
	return selected
}

func (l *LiveSimulator) driverHomeTerminal(driver Record) (name, address string) {
	snapshot := l.fleet()
	for _, field := range []string{keyVehicleGroupTagID, keyPeerGroupTagID} {
		tagID := stringValue(driver, field)
		if tagID == "" {
			continue
		}
		tag, ok := snapshot.tags.byID[tagID]
		if !ok {
			continue
		}
		tagName := stringValue(tag, keyName)
		if terminalAddress, known := dailyLogTerminalAddresses[tagName]; known {
			return tagName, terminalAddress
		}
	}
	return dailyLogDefaultTerminalName, dailyLogTerminalAddresses[dailyLogDefaultTerminalName]
}

func (l *LiveSimulator) dailyLogCertification(
	driverID string,
	dayKey string,
	dayEnd time.Time,
	now time.Time,
) (bool, time.Time) {
	if dayEnd.After(now) {
		return false, time.Time{}
	}
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
