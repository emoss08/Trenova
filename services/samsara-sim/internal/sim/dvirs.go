package sim

import (
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	dvirTypePreTrip  = "preTrip"
	dvirTypePostTrip = "postTrip"

	dvirSafetyStatusSafe     = "safe"
	dvirSafetyStatusUnsafe   = "unsafe"
	dvirSafetyStatusResolved = "resolved"

	dvirSignatureTypeDriver = "driver"

	dvirUnsafeRate          = 0.10
	dvirSecondDefectRate    = 0.4
	dvirDefectResolveAge    = 48 * time.Hour
	dvirHistoryMaxRangeDays = 30
)

type dvirDefectCatalogEntry struct {
	DefectType string
	Comment    string
}

var dvirDefectCatalog = []dvirDefectCatalogEntry{
	{DefectType: "Brake Hose", Comment: "Brake hose chafing against frame rail"},
	{DefectType: "Tires", Comment: "Steer tire tread depth near minimum"},
	{DefectType: "Lights", Comment: "Marker lamp out on passenger side"},
	{DefectType: "Air Compressor", Comment: "Air compressor slow to build pressure"},
	{DefectType: "Wipers", Comment: "Wiper blade streaking on driver side"},
	{DefectType: "Mirrors", Comment: "Passenger mirror bracket loose"},
}

type dvirGenerationContext struct {
	Now            time.Time
	Assets         map[string]Record
	StatsTemplates map[string]Record
	Waypoints      map[string][]routePoint
	GeometryCache  map[string]*routeGeometry
}

type dvirDriverDay struct {
	DriverID   string
	DriverName string
	VehicleID  string
	Day        time.Time
}

func (s *Server) registerDvirRoutes() {
	s.mux.HandleFunc("GET /fleet/dvirs/history", s.handleDvirHistory)
}

func (s *Server) handleDvirHistory(writer http.ResponseWriter, request *http.Request) {
	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	if startTime == nil || endTime == nil {
		s.writeAPIError(writer, http.StatusBadRequest, ErrTimeRangeRequired)
		return
	}
	if endTime.Sub(*startTime) > dvirHistoryMaxRangeDays*24*time.Hour {
		s.writeAPIError(writer, http.StatusBadRequest, ErrInvalidBody)
		return
	}

	driverIDs := idsFromQuery(request.URL.Query(), "driverIds")
	vehicleIDs := idsFromQuery(request.URL.Query(), "vehicleIds")
	records := []Record{}
	if s.live != nil {
		records = s.live.Dvirs(s.simNow(), *startTime, *endTime, driverIDs, vehicleIDs)
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
	s.respondJSON(writer, request, requestSignature(request)+"|dvir-history", payload)
}

func (l *LiveSimulator) Dvirs(
	now time.Time,
	windowStart time.Time,
	windowEnd time.Time,
	driverIDs []string,
	vehicleIDs []string,
) []Record {
	now = now.UTC()
	windowStart = windowStart.UTC()
	windowEnd = windowEnd.UTC()
	if windowEnd.Before(windowStart) {
		windowStart, windowEnd = windowEnd, windowStart
	}

	roster := l.loadDriverRoster()
	ids := selectDriverIDs(driverIDs, map[string]Record{}, roster)
	if len(ids) == 0 {
		return []Record{}
	}
	vehicleFilter := toStringSet(vehicleIDs)

	ctx := dvirGenerationContext{
		Now:            now,
		Assets:         l.loadAssetMetadata(),
		StatsTemplates: l.loadVehicleStatsTemplates(),
		Waypoints:      l.loadAssetWaypoints(),
		GeometryCache:  map[string]*routeGeometry{},
	}

	startDay := windowStart.Add(-24 * time.Hour).Truncate(24 * time.Hour)
	endDay := windowEnd.Truncate(24 * time.Hour)
	dayCount := int(endDay.Sub(startDay)/(24*time.Hour)) + 1

	out := make([]Record, 0, len(ids)*dayCount*2)
	for _, driverID := range ids {
		entry := roster[driverID]
		vehicleID := strings.TrimSpace(entry.VehicleID)
		if vehicleID == "" {
			continue
		}
		if !matchesStringFilter(vehicleFilter, vehicleID) {
			continue
		}
		for day := startDay; !day.After(endDay); day = day.Add(24 * time.Hour) {
			driverDay := dvirDriverDay{
				DriverID:   driverID,
				DriverName: firstNonEmpty(entry.Name, driverID),
				VehicleID:  vehicleID,
				Day:        day,
			}
			for _, record := range l.driverDayDvirs(&ctx, &driverDay) {
				endsAt := stringValue(record, "endTime")
				if endsAt <= windowStart.Format(time.RFC3339) ||
					endsAt > windowEnd.Format(time.RFC3339) {
					continue
				}
				out = append(out, record)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		left := stringValue(out[i], "endTime")
		right := stringValue(out[j], "endTime")
		if left == right {
			return recordID(out[i]) < recordID(out[j])
		}
		return left < right
	})
	return out
}

func (l *LiveSimulator) driverDayDvirs(
	ctx *dvirGenerationContext,
	driverDay *dvirDriverDay,
) []Record {
	dayKey := driverDay.Day.Format("2006-01-02")
	dayCtx := l.buildDailyEventContext(driverDay.DriverID, driverDay.VehicleID, driverDay.Day)
	shiftStart := l.shiftStartForDay(driverDay.DriverID, driverDay.Day).Truncate(time.Second)

	preDuration := time.Duration(
		(8 + 9*l.hashFraction("dvir-duration", driverDay.DriverID, dayKey, dvirTypePreTrip)) *
			float64(time.Minute),
	)
	postStart := dayCtx.DrivingEnd.Truncate(time.Second)
	postDuration := time.Duration(
		(6 + 9*l.hashFraction("dvir-duration", driverDay.DriverID, dayKey, dvirTypePostTrip)) *
			float64(time.Minute),
	)

	out := make([]Record, 0, 2)
	for _, spec := range []struct {
		Type  string
		Start time.Time
		End   time.Time
	}{
		{Type: dvirTypePreTrip, Start: shiftStart, End: shiftStart.Add(preDuration)},
		{Type: dvirTypePostTrip, Start: postStart, End: postStart.Add(postDuration)},
	} {
		if spec.End.After(ctx.Now) {
			continue
		}
		out = append(out, l.buildDvirRecord(ctx, driverDay, spec.Type, spec.Start, spec.End))
	}
	return out
}

func (l *LiveSimulator) buildDvirRecord(
	ctx *dvirGenerationContext,
	driverDay *dvirDriverDay,
	dvirType string,
	startTime time.Time,
	endTime time.Time,
) Record {
	dayKey := driverDay.Day.Format("2006-01-02")
	dvirID := strings.Join([]string{
		"dvir",
		driverDay.Day.UTC().Format("20060102"),
		driverDay.DriverID,
		dvirTypeSuffix(dvirType),
	}, "-")

	unsafeRoll := l.hashFraction("dvir-unsafe", driverDay.DriverID, dayKey, dvirType)
	isUnsafe := unsafeRoll < dvirUnsafeRate
	resolved := isUnsafe && ctx.Now.Sub(endTime) >= dvirDefectResolveAge

	safetyStatus := dvirSafetyStatusSafe
	switch {
	case resolved:
		safetyStatus = dvirSafetyStatusResolved
	case isUnsafe:
		safetyStatus = dvirSafetyStatusUnsafe
	}

	record := Record{
		"id":             dvirID,
		"type":           dvirType,
		"safetyStatus":   safetyStatus,
		"startTime":      startTime.UTC().Format(time.RFC3339),
		"endTime":        endTime.UTC().Format(time.RFC3339),
		"odometerMeters": l.dvirOdometerMeters(ctx, driverDay.VehicleID, endTime),
		"location":       l.dvirLocation(ctx, driverDay, startTime),
		"authorSignature": map[string]any{
			"signatoryUser": map[string]any{
				"id":   driverDay.DriverID,
				"name": driverDay.DriverName,
			},
			"signedAtTime": endTime.UTC().Format(time.RFC3339),
			"type":         dvirSignatureTypeDriver,
		},
		"driver": map[string]any{
			"id":   driverDay.DriverID,
			"name": driverDay.DriverName,
		},
		"vehicle": map[string]any{
			"id":   driverDay.VehicleID,
			"name": vehicleIDToName(driverDay.VehicleID, ctx.Assets),
		},
	}

	if asset, ok := ctx.Assets[driverDay.VehicleID]; ok {
		if licensePlate := stringValue(asset, "licensePlate"); licensePlate != "" {
			record["licensePlate"] = licensePlate
		}
	}
	if isUnsafe {
		record["vehicleDefects"] = l.dvirDefects(ctx, driverDay, dvirID, dvirType, endTime, resolved)
	}
	return record
}

func (l *LiveSimulator) dvirDefects(
	ctx *dvirGenerationContext,
	driverDay *dvirDriverDay,
	dvirID string,
	dvirType string,
	endTime time.Time,
	resolved bool,
) []any {
	dayKey := driverDay.Day.Format("2006-01-02")
	count := 1
	if l.hashFraction("dvir-defect-count", driverDay.DriverID, dayKey, dvirType) < dvirSecondDefectRate {
		count = 2
	}
	baseIndex := int(
		math.Floor(
			float64(len(dvirDefectCatalog)) *
				l.hashFraction("dvir-defect", driverDay.DriverID, dayKey, dvirType),
		),
	)
	if baseIndex >= len(dvirDefectCatalog) {
		baseIndex = len(dvirDefectCatalog) - 1
	}

	out := make([]any, 0, count)
	for idx := 0; idx < count; idx++ {
		catalog := dvirDefectCatalog[(baseIndex+idx)%len(dvirDefectCatalog)]
		defect := map[string]any{
			"id":            dvirID + "-defect-" + strconv.Itoa(idx+1),
			"defectType":    catalog.DefectType,
			"comment":       catalog.Comment,
			"createdAtTime": endTime.UTC().Format(time.RFC3339),
			"isResolved":    resolved,
			"vehicle": map[string]any{
				"id":   driverDay.VehicleID,
				"name": vehicleIDToName(driverDay.VehicleID, ctx.Assets),
			},
		}
		if resolved {
			resolveDelay := time.Duration(
				(18 + 24*l.hashFraction(
					"dvir-defect-resolved",
					driverDay.DriverID,
					dayKey,
					dvirType,
					strconv.Itoa(idx),
				)) * float64(time.Hour),
			)
			resolvedAt := minTime(endTime.Add(resolveDelay), ctx.Now)
			defect["resolvedAtTime"] = resolvedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, defect)
	}
	return out
}

func (l *LiveSimulator) dvirOdometerMeters(
	ctx *dvirGenerationContext,
	vehicleID string,
	at time.Time,
) int64 {
	return dynamicOdometerMeters(
		ctx.StatsTemplates[vehicleID],
		at,
		l.startedAt,
		l.hashFraction(vehicleID),
	)
}

func (l *LiveSimulator) dvirLocation(
	ctx *dvirGenerationContext,
	driverDay *dvirDriverDay,
	at time.Time,
) string {
	geometry := l.cachedRouteGeometry(ctx.GeometryCache, ctx.Waypoints, driverDay.VehicleID)
	if geometry != nil && len(geometry.Points) > 0 {
		state := l.routeStateForGeometry(
			driverDay.VehicleID,
			geometry,
			at,
			at.Add(-time.Minute),
			ctx.Now,
		)
		if formatted := formattedLocationFromAddress(state.Address); formatted != "" {
			return formatted
		}
	}
	return l.dailyLogHomeTerminal(driverDay.DriverID) + ", TX"
}

func dvirTypeSuffix(dvirType string) string {
	if dvirType == dvirTypePreTrip {
		return "pre"
	}
	return "post"
}

func (l *LiveSimulator) DvirWebhookEmissions(
	now time.Time,
	windowStart time.Time,
	windowEnd time.Time,
) []WebhookEmission {
	dvirs := l.Dvirs(now, windowStart, windowEnd, nil, nil)
	if len(dvirs) == 0 {
		return []WebhookEmission{}
	}

	assets := l.loadAssetMetadata()
	out := make([]WebhookEmission, 0, len(dvirs))
	for _, record := range dvirs {
		out = append(out, WebhookEmission{
			EventType: "DvirSubmitted",
			UniqueKey: "DvirSubmitted|" + recordID(record),
			Data:      dvirWebhookData(record, assets),
		})
	}
	return out
}

func dvirWebhookData(record Record, assets map[string]Record) map[string]any {
	vehicleID := nestedString(record, "vehicle", "id")
	vehicle := vehicleWebhookPayload(vehicleID, assets)

	dvir := map[string]any{
		"id":                recordID(record),
		"type":              stringValue(record, "type"),
		"safetyStatus":      stringValue(record, "safetyStatus"),
		"startTime":         stringValue(record, "startTime"),
		"endTime":           stringValue(record, "endTime"),
		"odometerMeters":    record["odometerMeters"],
		"formattedLocation": stringValue(record, "location"),
		"authorSignature":   cloneAny(record["authorSignature"]),
		"needsCorrection":   stringValue(record, "safetyStatus") == dvirSafetyStatusUnsafe,
	}
	if defects, ok := record["vehicleDefects"].([]any); ok && len(defects) > 0 {
		dvir["defects"] = cloneAny(defects)
	}

	return map[string]any{
		"driver":  cloneAny(record["driver"]),
		"vehicle": vehicle,
		"dvir":    dvir,
	}
}
