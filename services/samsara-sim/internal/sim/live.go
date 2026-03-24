package sim

import (
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAssetSampleStep   = 2 * time.Minute
	defaultAssetLookback     = 20 * time.Minute
	maxAssetSamplesPerAsset  = 96
	defaultAssetSpeedMPS     = 12.5
	minLoopFloorHours        = 1
	minRouteSpeedMPS         = 2.0
	maxRouteSpeedMPS         = 31.3 // ~70 mph
	defaultGPSJitterMeters   = 1.0
	minSegmentDistanceMeters = 1.0
	earthRadiusMeters        = 6371000.0
	metersPerDegreeLatitude  = 111320.0
	movingSpeedThresholdMPS  = 0.75
	hosStatusOnDuty          = "onDuty"
	hosStatusDriving         = "driving"
	hosStatusOffDuty         = "offDuty"
	hosStatusSleeperBed      = "sleeperBed"
	routeStatusPlanned       = "planned"
	routeStatusAssigned      = "assigned"
	routeStatusEnRoute       = "enRoute"
	routeStatusAtStop        = "atStop"
	routeStatusCompleted     = "completed"
	routeStatusCanceled      = "canceled"
	routeStopStatusPending   = "pending"
	routeStopStatusAtStop    = "atStop"
	routeStopStatusCompleted = "completed"
	routeStopStatusMissed    = "missed"
	eldBreakLimit            = 8 * time.Hour
	eldBreakResetDuration    = 30 * time.Minute
	eldDriveLimit            = 11 * time.Hour
	eldShiftLimit            = 14 * time.Hour
	eldShiftResetDuration    = 10 * time.Hour
	eldCycleLimit            = 70 * time.Hour
	eldCycleWindow           = 8 * 24 * time.Hour
	eldLookbackDuration      = eldCycleWindow + 24*time.Hour
	eldLookaheadDuration     = 36 * time.Hour
)

type LiveSimulator struct {
	store      *Store
	seed       string
	startedAt  time.Time
	anchorTime time.Time
	options    LiveSimulationOptions
	scripts    *ScriptEngine
}

type LiveSimulationOptions struct {
	FleetSize      int
	TripHoursMin   int
	TripHoursMax   int
	EventIntensity string
	ViolationRate  float64
	SpeedingRate   float64
	ScriptMode     string
}

type routePoint struct {
	Latitude  float64
	Longitude float64
	Heading   float64
	SpeedMPS  float64
	Address   map[string]any
}

type routeState struct {
	Latitude  float64
	Longitude float64
	Heading   float64
	SpeedMPS  float64
	Address   map[string]any
}

type routeSegment struct {
	From            routePoint
	To              routePoint
	DistanceMeters  float64
	SpeedMPS        float64
	Duration        time.Duration
	CumulativeStart time.Duration
}

type routeStopLifecycle struct {
	ID                   string
	Sequence             int
	Name                 string
	Latitude             float64
	Longitude            float64
	ScheduledWindowStart time.Time
	ScheduledWindowEnd   time.Time
	ETA                  time.Time
	Arrival              *time.Time
	Departure            *time.Time
	Status               string
	DwellDuration        time.Duration
}

type routeLifecycle struct {
	RouteID         string
	VehicleID       string
	DriverID        string
	Status          string
	TripIndex       int64
	ScheduledStart  time.Time
	AssignedAt      time.Time
	EnRouteAt       time.Time
	CompletedAt     time.Time
	CanceledAt      *time.Time
	CurrentStopID   string
	NextStopID      string
	ProgressPercent float64
	Stops           []routeStopLifecycle
}

type driverRoster struct {
	Name      string
	VehicleID string
}

type dutySegment struct {
	Status   string
	Duration time.Duration
}

type timelineSegment struct {
	Status string
	Start  time.Time
	End    time.Time
}

type eldSnapshot struct {
	DutyStatus            string
	BreakRemaining        time.Duration
	DriveRemaining        time.Duration
	ShiftRemaining        time.Duration
	CycleRemaining        time.Duration
	CycleTomorrow         time.Duration
	CycleWindowStartedAt  time.Time
	CycleViolation        time.Duration
	ShiftDrivingViolation time.Duration
}

func NewLiveSimulator(store *Store, seed string, options ...LiveSimulationOptions) *LiveSimulator {
	now := time.Now().UTC()
	opts := defaultLiveSimulationOptions()
	if len(options) > 0 {
		opts = normalizeLiveSimulationOptions(options[0])
	}
	return &LiveSimulator{
		store:      store,
		seed:       strings.TrimSpace(seed),
		startedAt:  now,
		anchorTime: now.Truncate(time.Minute),
		options:    opts,
	}
}

func defaultLiveSimulationOptions() LiveSimulationOptions {
	return LiveSimulationOptions{
		FleetSize:      12,
		TripHoursMin:   8,
		TripHoursMax:   12,
		EventIntensity: "balanced",
		ViolationRate:  0.08,
		SpeedingRate:   0.14,
		ScriptMode:     scriptModeMerge,
	}
}

func normalizeLiveSimulationOptions(options LiveSimulationOptions) LiveSimulationOptions {
	normalized := options
	defaults := defaultLiveSimulationOptions()

	if normalized.FleetSize <= 0 {
		normalized.FleetSize = defaults.FleetSize
	}
	if normalized.TripHoursMin <= 0 {
		normalized.TripHoursMin = defaults.TripHoursMin
	}
	if normalized.TripHoursMax <= 0 {
		normalized.TripHoursMax = defaults.TripHoursMax
	}
	if normalized.TripHoursMax < normalized.TripHoursMin {
		normalized.TripHoursMax = normalized.TripHoursMin
	}
	if strings.TrimSpace(normalized.EventIntensity) == "" {
		normalized.EventIntensity = defaults.EventIntensity
	}
	if normalized.ViolationRate < 0 || normalized.ViolationRate > 1 {
		normalized.ViolationRate = defaults.ViolationRate
	}
	if normalized.SpeedingRate < 0 || normalized.SpeedingRate > 1 {
		normalized.SpeedingRate = defaults.SpeedingRate
	}
	normalized.ScriptMode = normalizeScriptMode(normalized.ScriptMode)
	return normalized
}

func (l *LiveSimulator) SetScriptEngine(engine *ScriptEngine) {
	l.scripts = engine
}

func (l *LiveSimulator) AssetStream(
	now time.Time,
	assetIDs []string,
	startTime *time.Time,
	endTime *time.Time,
) []Record {
	waypoints := l.loadAssetWaypoints()
	selectedAssetIDs := selectAssetIDs(assetIDs, waypoints)
	if len(selectedAssetIDs) == 0 {
		return []Record{}
	}

	windowStart, windowEnd := resolveWindow(now, startTime, endTime, defaultAssetLookback)
	times := sampleTimes(windowStart, windowEnd, defaultAssetSampleStep, maxAssetSamplesPerAsset)
	if len(times) == 0 {
		return []Record{}
	}

	assetMetadata := l.loadAssetMetadata()
	driverByVehicle := l.driverByVehicleMap()
	eventsByVehicle := l.vehicleEventsForWindow(
		selectedAssetIDs,
		driverByVehicle,
		windowStart.Add(-2*time.Hour),
		windowEnd.Add(2*time.Hour),
	)
	stream := make([]Record, 0, len(selectedAssetIDs)*len(times))
	for _, assetID := range selectedAssetIDs {
		points := waypoints[assetID]
		if len(points) == 0 {
			continue
		}
		for _, sampleTime := range times {
			state := l.routeStateForSample(assetID, points, sampleTime, windowStart, now)
			state = l.applyVehicleEventsToRouteState(
				assetID,
				points,
				eventsByVehicle[assetID],
				sampleTime,
				windowStart,
				now,
				state,
			)
			record := Record{
				"asset": map[string]any{
					"id": assetID,
				},
				"happenedAtTime": sampleTime.UTC().Format(time.RFC3339),
				"location": map[string]any{
					"latitude":       round(state.Latitude, 6),
					"longitude":      round(state.Longitude, 6),
					"headingDegrees": int64(math.Round(state.Heading)),
				},
				"speed": map[string]any{
					"gpsSpeedMetersPerSecond": round(state.SpeedMPS, 2),
					"ecuSpeedMetersPerSecond": round(state.SpeedMPS*0.985, 2),
				},
			}

			if len(state.Address) > 0 {
				if location, ok := anyAsMap(record["location"]); ok {
					location["address"] = cloneAny(state.Address)
					record["location"] = location
				}
			}
			if meta, ok := assetMetadata[assetID]; ok {
				if externalIDs, okMeta := meta["externalIds"]; okMeta {
					if asset, okAsset := anyAsMap(record["asset"]); okAsset {
						asset["externalIds"] = cloneAny(externalIDs)
						record["asset"] = asset
					}
				}
			}
			stream = append(stream, record)
		}
	}

	sort.Slice(stream, func(i, j int) bool {
		ti := stringValue(stream[i], "happenedAtTime")
		tj := stringValue(stream[j], "happenedAtTime")
		if ti == tj {
			return nestedString(stream[i], "asset", "id") < nestedString(stream[j], "asset", "id")
		}
		return ti < tj
	})

	return stream
}

func (l *LiveSimulator) VehicleStats(now time.Time, vehicleIDs []string) []Record {
	templates := l.loadVehicleStatsTemplates()
	waypoints := l.loadAssetWaypoints()
	assets := l.loadAssetMetadata()

	ids := selectVehicleIDs(vehicleIDs, templates, waypoints, assets)
	if len(ids) == 0 {
		return []Record{}
	}

	windowStart := now.Add(-15 * time.Minute)
	driverByVehicle := l.driverByVehicleMap()
	eventsByVehicle := l.vehicleEventsForWindow(
		ids,
		driverByVehicle,
		windowStart.Add(-2*time.Hour),
		now.Add(2*time.Hour),
	)
	output := make([]Record, 0, len(ids))
	for _, vehicleID := range ids {
		base, hasBase := templates[vehicleID]
		record := Record{
			"id":   vehicleID,
			"name": vehicleName(vehicleID, base, assets),
		}
		if hasBase {
			record = cloneRecord(base)
			record["id"] = vehicleID
			record["name"] = vehicleName(vehicleID, base, assets)
		}

		points := waypoints[vehicleID]
		if len(points) > 0 {
			state := l.routeStateForSample(vehicleID, points, now, windowStart, now)
			state = l.applyVehicleEventsToRouteState(
				vehicleID,
				points,
				eventsByVehicle[vehicleID],
				now,
				windowStart,
				now,
				state,
			)
			gps := map[string]any{
				"latitude":       round(state.Latitude, 6),
				"longitude":      round(state.Longitude, 6),
				"headingDegrees": int64(math.Round(state.Heading)),
				"speedMilesPerHour": round(
					state.SpeedMPS*2.23694,
					2,
				),
				"isEcuSpeed": true,
				"time":       now.UTC().Format(time.RFC3339),
			}
			if len(state.Address) > 0 {
				gps["address"] = cloneAny(state.Address)
			}
			record["gps"] = gps
			record["ecuSpeedMph"] = map[string]any{
				"time":  now.UTC().Format(time.RFC3339),
				"value": round(state.SpeedMPS*2.23694, 2),
			}
			engineOn := state.SpeedMPS > 0.75
			record["engineState"] = map[string]any{
				"time":  now.UTC().Format(time.RFC3339),
				"value": ternary(engineOn, "On", "Off"),
			}
		}

		fuelValue := dynamicFuelPercent(base, now, l.startedAt, l.hashFraction(vehicleID))
		record["fuelPercent"] = map[string]any{
			"time":  now.UTC().Format(time.RFC3339),
			"value": fuelValue,
		}
		record["obdEngineSeconds"] = map[string]any{
			"time":  now.UTC().Format(time.RFC3339),
			"value": dynamicEngineSeconds(base, now, l.startedAt, l.hashFraction(vehicleID)),
		}
		record["batteryMilliVolts"] = map[string]any{
			"time":  now.UTC().Format(time.RFC3339),
			"value": dynamicBatteryMilliVolts(base, now, l.hashFraction(vehicleID)),
		}

		output = append(output, record)
	}

	sort.Slice(output, func(i, j int) bool {
		return recordID(output[i]) < recordID(output[j])
	})
	return output
}

func (l *LiveSimulator) Routes(now time.Time, routeIDs, statusFilters []string) []Record {
	routes, err := l.store.List(ResourceRoutes)
	if err != nil {
		return []Record{}
	}

	selected := routes
	if len(routeIDs) > 0 {
		selected = filterByIDs(routes, routeIDs)
	}
	allowedStatus := normalizeRouteStatusFilters(statusFilters)
	waypoints := l.loadAssetWaypoints()
	out := make([]Record, 0, len(selected))
	for _, route := range selected {
		lifecycle := l.routeLifecycleForRecord(now, route, waypoints)
		if len(allowedStatus) > 0 {
			if _, ok := allowedStatus[strings.ToLower(lifecycle.Status)]; !ok {
				continue
			}
		}
		out = append(out, augmentRouteWithLifecycle(route, &lifecycle))
	}

	sort.Slice(out, func(i, j int) bool {
		return recordID(out[i]) < recordID(out[j])
	})
	return out
}

func (l *LiveSimulator) RouteByID(now time.Time, routeID string) (Record, bool) {
	cleanID := strings.TrimSpace(routeID)
	if cleanID == "" {
		return nil, false
	}
	routes := l.Routes(now, []string{cleanID}, nil)
	if len(routes) == 0 {
		return nil, false
	}
	return cloneRecord(routes[0]), true
}

func normalizeRouteStatusFilters(filters []string) map[string]struct{} {
	if len(filters) == 0 {
		return map[string]struct{}{}
	}
	allowed := map[string]struct{}{}
	for _, value := range filters {
		normalized := strings.ToLower(strings.TrimSpace(value))
		switch normalized {
		case strings.ToLower(routeStatusPlanned),
			strings.ToLower(routeStatusAssigned),
			strings.ToLower(routeStatusEnRoute),
			strings.ToLower(routeStatusAtStop),
			strings.ToLower(routeStatusCompleted),
			strings.ToLower(routeStatusCanceled):
			allowed[normalized] = struct{}{}
		}
	}
	return allowed
}

func augmentRouteWithLifecycle(route Record, lifecycle *routeLifecycle) Record {
	augmented := cloneRecord(route)
	if lifecycle == nil {
		return augmented
	}

	stops := make([]any, 0, len(lifecycle.Stops))
	completedStops := 0
	missedStops := 0
	for idx := range lifecycle.Stops {
		stop := lifecycle.Stops[idx]
		stopPayload := map[string]any{
			"id":       stop.ID,
			"sequence": stop.Sequence,
			"name":     stop.Name,
			"status":   stop.Status,
			"location": map[string]any{
				"latitude":  round(stop.Latitude, 6),
				"longitude": round(stop.Longitude, 6),
			},
			"scheduledWindowStartTime": stop.ScheduledWindowStart.UTC().Format(time.RFC3339),
			"scheduledWindowEndTime":   stop.ScheduledWindowEnd.UTC().Format(time.RFC3339),
			"etaTime":                  stop.ETA.UTC().Format(time.RFC3339),
		}
		if stop.Arrival != nil {
			stopPayload["arrivalTime"] = stop.Arrival.UTC().Format(time.RFC3339)
		}
		if stop.Departure != nil {
			stopPayload["departureTime"] = stop.Departure.UTC().Format(time.RFC3339)
			stopPayload["dwellDurationMs"] = float64(stop.DwellDuration.Milliseconds())
		}
		switch stop.Status {
		case routeStopStatusCompleted:
			completedStops++
		case routeStopStatusMissed:
			completedStops++
			missedStops++
		}
		stops = append(stops, stopPayload)
	}

	augmented["status"] = lifecycle.Status
	augmented["tripIndex"] = lifecycle.TripIndex
	augmented["stops"] = stops
	augmented["progress"] = map[string]any{
		"percentComplete": round(lifecycle.ProgressPercent, 2),
		"stopsCompleted":  completedStops,
		"stopsMissed":     missedStops,
		"totalStops":      len(stops),
		"currentStopId":   lifecycle.CurrentStopID,
		"nextStopId":      lifecycle.NextStopID,
	}
	lifecyclePayload := map[string]any{
		"status":             lifecycle.Status,
		"tripIndex":          lifecycle.TripIndex,
		"scheduledStartTime": lifecycle.ScheduledStart.UTC().Format(time.RFC3339),
		"assignedAtTime":     lifecycle.AssignedAt.UTC().Format(time.RFC3339),
		"enRouteAtTime":      lifecycle.EnRouteAt.UTC().Format(time.RFC3339),
		"completedAtTime":    lifecycle.CompletedAt.UTC().Format(time.RFC3339),
		"progressPercent":    round(lifecycle.ProgressPercent, 2),
		"currentStopId":      lifecycle.CurrentStopID,
		"nextStopId":         lifecycle.NextStopID,
	}
	if lifecycle.CanceledAt != nil {
		lifecyclePayload["canceledAtTime"] = lifecycle.CanceledAt.UTC().Format(time.RFC3339)
	}
	augmented["lifecycle"] = lifecyclePayload
	return augmented
}

func (l *LiveSimulator) routeLifecycleForRecord(
	now time.Time,
	route Record,
	waypoints map[string][]routePoint,
) routeLifecycle {
	routeID := recordID(route)
	driverID := nestedString(route, "driver", "id")
	vehicleID := nestedString(route, "vehicle", "id")
	seedKey := firstNonEmpty(routeID, driverID, vehicleID)
	if seedKey == "" {
		seedKey = "route"
	}

	frame := l.routeLifecycleFrame(now, vehicleID, seedKey)

	stopCount := clampInt(
		3+int(math.Floor(4*l.hashFraction("route-stop-count", routeID))),
		3,
		6,
	)
	stopPoints := l.routeStopPointsForLifecycle(seedKey, waypoints[vehicleID], stopCount)
	missedStopIndex := l.routeMissedStopIndex(routeID, frame.TripIndex, stopCount)
	stopBuild := l.buildRouteStopsForLifecycle(
		now,
		routeID,
		&frame,
		stopPoints,
		missedStopIndex,
	)
	isCanceled, canceledAt := l.routeCancellation(routeID, &frame)
	status := resolveRouteLifecycleStatus(
		isCanceled,
		frame.PhaseFraction,
		&frame,
		stopBuild.CurrentStopID,
	)
	progressPercent := routeLifecycleProgressPercent(
		status,
		&frame,
		stopCount,
		stopBuild.CompletedStops,
	)

	return routeLifecycle{
		RouteID:         routeID,
		VehicleID:       vehicleID,
		DriverID:        driverID,
		Status:          status,
		TripIndex:       frame.TripIndex,
		ScheduledStart:  frame.TripStart,
		AssignedAt:      frame.AssignedAt,
		EnRouteAt:       frame.AssignedAt,
		CompletedAt:     frame.CompletedAt,
		CanceledAt:      canceledAt,
		CurrentStopID:   stopBuild.CurrentStopID,
		NextStopID:      stopBuild.NextStopID,
		ProgressPercent: progressPercent,
		Stops:           stopBuild.Stops,
	}
}

type routeLifecycleFrame struct {
	Period              time.Duration
	TripIndex           int64
	PhaseFraction       float64
	TripStart           time.Time
	PlannedEndFraction  float64
	AssignedEndFraction float64
	CompletedFraction   float64
	AssignedAt          time.Time
	CompletedAt         time.Time
}

type routeStopBuild struct {
	Stops          []routeStopLifecycle
	CurrentStopID  string
	NextStopID     string
	CompletedStops int
}

func (l *LiveSimulator) routeLifecycleFrame(
	now time.Time,
	vehicleID string,
	seedKey string,
) routeLifecycleFrame {
	period := l.routeLoopTargetPeriod(firstNonEmpty(vehicleID, seedKey))
	if period < 6*time.Hour {
		period = 6 * time.Hour
	}
	offset := l.phaseOffset("route-lifecycle|"+seedKey, period)
	elapsed := now.UTC().Sub(l.anchorTime) + offset
	tripIndex := int64(0)
	for elapsed < 0 {
		elapsed += period
		tripIndex--
	}
	if period > 0 {
		tripIndex += int64(elapsed / period)
	}
	phase := normalizePhase(elapsed, period)
	phaseFraction := float64(phase) / float64(period)
	tripStart := now.UTC().Add(-phase)
	plannedEndFraction := 0.06
	assignedEndFraction := 0.12
	completedFraction := 0.95
	assignedAt := tripStart.Add(time.Duration(assignedEndFraction * float64(period)))
	completedAt := tripStart.Add(time.Duration(completedFraction * float64(period)))
	return routeLifecycleFrame{
		Period:              period,
		TripIndex:           tripIndex,
		PhaseFraction:       phaseFraction,
		TripStart:           tripStart,
		PlannedEndFraction:  plannedEndFraction,
		AssignedEndFraction: assignedEndFraction,
		CompletedFraction:   completedFraction,
		AssignedAt:          assignedAt,
		CompletedAt:         completedAt,
	}
}

func (l *LiveSimulator) routeMissedStopIndex(routeID string, tripIndex int64, stopCount int) int {
	if stopCount <= 0 {
		return -1
	}
	if l.hashFraction(
		"route-missed-stop-enabled",
		routeID,
		strconv.FormatInt(tripIndex, 10),
	) >= 0.22 {
		return -1
	}
	index := int(
		math.Floor(
			float64(stopCount) * l.hashFraction(
				"route-missed-stop-index",
				routeID,
				strconv.FormatInt(tripIndex, 10),
			),
		),
	)
	if index < 0 {
		index = 0
	}
	if index >= stopCount {
		index = stopCount - 1
	}
	return index
}

func (l *LiveSimulator) buildRouteStopsForLifecycle(
	now time.Time,
	routeID string,
	frame *routeLifecycleFrame,
	stopPoints []routePoint,
	missedStopIndex int,
) routeStopBuild {
	stopCount := len(stopPoints)
	stops := make([]routeStopLifecycle, 0, stopCount)
	currentStopID := ""
	nextStopID := ""
	completedStops := 0
	for idx := 0; idx < stopCount; idx++ {
		stop := l.routeStopLifecycle(
			now,
			routeID,
			frame,
			stopPoints[idx],
			stopCount,
			idx,
			missedStopIndex,
		)
		if stop.Status == routeStopStatusAtStop && currentStopID == "" {
			currentStopID = stop.ID
		}
		if stop.Status == routeStopStatusPending && nextStopID == "" {
			nextStopID = stop.ID
		}
		if stop.Status == routeStopStatusCompleted || stop.Status == routeStopStatusMissed {
			completedStops++
		}
		stops = append(stops, stop)
	}
	return routeStopBuild{
		Stops:          stops,
		CurrentStopID:  currentStopID,
		NextStopID:     nextStopID,
		CompletedStops: completedStops,
	}
}

func (l *LiveSimulator) routeStopLifecycle(
	now time.Time,
	routeID string,
	frame *routeLifecycleFrame,
	stopPoint routePoint,
	stopCount int,
	idx int,
	missedStopIndex int,
) routeStopLifecycle {
	stopID := strings.Join([]string{routeID, "stop", strconv.Itoa(idx + 1)}, "-")
	stopName := "Stop " + strconv.Itoa(idx+1)
	progress := 0.14
	if stopCount > 1 {
		progress = 0.14 + (0.72 * float64(idx) / float64(stopCount-1))
	}
	dwellFraction := 0.012 + 0.02*l.hashFraction("route-dwell", routeID, strconv.Itoa(idx))
	arrivalAt := frame.TripStart.Add(time.Duration(progress * float64(frame.Period)))
	departureAt := arrivalAt.Add(time.Duration(dwellFraction * float64(frame.Period)))
	maxDeparture := frame.TripStart.Add(time.Duration(0.94 * float64(frame.Period)))
	if departureAt.After(maxDeparture) {
		departureAt = maxDeparture
	}
	departureFraction := float64(departureAt.Sub(frame.TripStart)) / float64(frame.Period)
	etaDrift := ((2 * l.hashFraction(
		"route-eta-drift",
		routeID,
		strconv.Itoa(idx),
		now.UTC().Truncate(5*time.Minute).Format(time.RFC3339),
	)) - 1) * 6
	etaTime := arrivalAt.Add(time.Duration(etaDrift * float64(time.Minute)))
	status := routeStopStatusPending

	var actualArrival *time.Time
	if frame.PhaseFraction >= progress {
		jitter := ((2 * l.hashFraction(
			"route-arrival-jitter",
			routeID,
			strconv.Itoa(idx),
			strconv.FormatInt(frame.TripIndex, 10),
		)) - 1) * 3
		arrived := arrivalAt.Add(time.Duration(jitter * float64(time.Minute)))
		actualArrival = &arrived
		status = routeStopStatusAtStop
	}

	var actualDeparture *time.Time
	if frame.PhaseFraction >= departureFraction {
		departed := departureAt
		actualDeparture = &departed
		status = routeStopStatusCompleted
	}
	if idx == missedStopIndex && frame.PhaseFraction >= departureFraction {
		status = routeStopStatusMissed
	}

	dwellDuration := time.Duration(0)
	if actualArrival != nil && actualDeparture != nil && actualDeparture.After(*actualArrival) {
		dwellDuration = actualDeparture.Sub(*actualArrival)
	}
	return routeStopLifecycle{
		ID:                   stopID,
		Sequence:             idx + 1,
		Name:                 stopName,
		Latitude:             stopPoint.Latitude,
		Longitude:            stopPoint.Longitude,
		ScheduledWindowStart: arrivalAt,
		ScheduledWindowEnd:   departureAt,
		ETA:                  etaTime,
		Arrival:              actualArrival,
		Departure:            actualDeparture,
		Status:               status,
		DwellDuration:        dwellDuration,
	}
}

func (l *LiveSimulator) routeCancellation(
	routeID string,
	frame *routeLifecycleFrame,
) (bool, *time.Time) {
	isCanceled := l.hashFraction(
		"route-canceled",
		routeID,
		strconv.FormatInt(frame.TripIndex, 10),
	) < 0.04 &&
		frame.PhaseFraction >= frame.AssignedEndFraction &&
		frame.PhaseFraction < 0.28
	if !isCanceled {
		return false, nil
	}
	value := frame.TripStart.Add(time.Duration(0.2 * float64(frame.Period)))
	return true, &value
}

func resolveRouteLifecycleStatus(
	isCanceled bool,
	phaseFraction float64,
	frame *routeLifecycleFrame,
	currentStopID string,
) string {
	switch {
	case isCanceled:
		return routeStatusCanceled
	case phaseFraction < frame.PlannedEndFraction:
		return routeStatusPlanned
	case phaseFraction < frame.AssignedEndFraction:
		return routeStatusAssigned
	case phaseFraction >= frame.CompletedFraction:
		return routeStatusCompleted
	case currentStopID != "":
		return routeStatusAtStop
	default:
		return routeStatusEnRoute
	}
}

func routeLifecycleProgressPercent(
	status string,
	frame *routeLifecycleFrame,
	stopCount int,
	completedStops int,
) float64 {
	switch status {
	case routeStatusPlanned:
		return 0
	case routeStatusAssigned:
		return 5
	case routeStatusCompleted:
		return 100
	case routeStatusCanceled:
		if stopCount <= 0 {
			return 0
		}
		return (float64(completedStops) / float64(stopCount)) * 100
	default:
		return clampFloat64(
			((frame.PhaseFraction-frame.AssignedEndFraction)/
				(frame.CompletedFraction-frame.AssignedEndFraction))*100,
			0,
			100,
		)
	}
}

func (l *LiveSimulator) routeStopPointsForLifecycle(
	seedKey string,
	points []routePoint,
	stopCount int,
) []routePoint {
	if stopCount <= 0 {
		return []routePoint{}
	}
	if len(points) == 0 {
		fallback := make([]routePoint, 0, stopCount)
		for idx := 0; idx < stopCount; idx++ {
			lat := 31.2 + ((2 * l.hashFraction(seedKey, "route-stop-fallback-lat", strconv.Itoa(idx))) - 1)
			lon := -97.5 + ((2 * l.hashFraction(seedKey, "route-stop-fallback-lon", strconv.Itoa(idx))) - 1.6)
			fallback = append(fallback, routePoint{
				Latitude:  lat,
				Longitude: lon,
			})
		}
		return fallback
	}
	if len(points) == 1 {
		single := make([]routePoint, 0, stopCount)
		for idx := 0; idx < stopCount; idx++ {
			single = append(single, points[0])
		}
		return single
	}

	stops := make([]routePoint, 0, stopCount)
	for idx := 0; idx < stopCount; idx++ {
		fraction := 0.0
		if stopCount > 1 {
			fraction = float64(idx) / float64(stopCount-1)
		}
		pointIndex := int(math.Round(fraction * float64(len(points)-1)))
		if pointIndex < 0 {
			pointIndex = 0
		}
		if pointIndex >= len(points) {
			pointIndex = len(points) - 1
		}
		stops = append(stops, points[pointIndex])
	}
	return stops
}

func (l *LiveSimulator) HOSClocks(now time.Time, driverIDs []string) []Record {
	templates := l.loadDriverTemplateRecords(ResourceHOSClocks)
	roster := l.loadDriverRoster()
	assets := l.loadAssetMetadata()
	ids := selectDriverIDs(driverIDs, templates, roster)
	if len(ids) == 0 {
		return []Record{}
	}

	out := make([]Record, 0, len(ids))
	for _, driverID := range ids {
		base := cloneRecord(templates[driverID])
		if len(base) == 0 {
			base = Record{}
		}

		snapshot := l.eldSnapshotForDriver(now, driverID)
		driver := roster[driverID]
		vehicleID := strings.TrimSpace(driver.VehicleID)
		primaryEvent := l.primaryDriverEventAt(driverID, vehicleID, now)
		if primaryEvent != nil {
			applyHOSSnapshotEvent(&snapshot, primaryEvent, now)
		}
		dutyStatus := normalizeDutyStatusForVehicle(
			snapshot.DutyStatus,
			vehicleID,
			l.shouldForceDriving(vehicleID, now, now),
		)
		if primaryEvent != nil {
			dutyStatus = dutyStatusForSimEvent(primaryEvent, dutyStatus)
		}

		record := Record{
			"driver": map[string]any{
				"id":   driverID,
				"name": driver.Name,
			},
			"currentDutyStatus": map[string]any{
				"hosStatusType": dutyStatus,
			},
			"clocks": map[string]any{
				"break": map[string]any{
					"timeUntilBreakDurationMs": float64(snapshot.BreakRemaining.Milliseconds()),
				},
				"drive": map[string]any{
					"driveRemainingDurationMs": float64(snapshot.DriveRemaining.Milliseconds()),
				},
				"shift": map[string]any{
					"shiftRemainingDurationMs": float64(snapshot.ShiftRemaining.Milliseconds()),
				},
				"cycle": map[string]any{
					"cycleRemainingDurationMs": float64(snapshot.CycleRemaining.Milliseconds()),
					"cycleTomorrowDurationMs":  float64(snapshot.CycleTomorrow.Milliseconds()),
					"cycleStartedAtTime": snapshot.CycleWindowStartedAt.UTC().
						Format(time.RFC3339),
				},
			},
			"violations": map[string]any{
				"cycleViolationDurationMs": float64(snapshot.CycleViolation.Milliseconds()),
				"shiftDrivingViolationDurationMs": float64(
					snapshot.ShiftDrivingViolation.Milliseconds(),
				),
			},
		}

		if vehicleID != "" {
			record["currentVehicle"] = map[string]any{
				"id":   vehicleID,
				"name": vehicleIDToName(vehicleID, assets),
			}
		}

		overlayRecord(base, record)
		out = append(out, base)
	}

	sort.Slice(out, func(i, j int) bool {
		return nestedString(out[i], "driver", "id") < nestedString(out[j], "driver", "id")
	})
	return out
}

func (l *LiveSimulator) HOSLogs(
	now time.Time,
	driverIDs []string,
	startTime *time.Time,
	endTime *time.Time,
) []Record {
	templates := l.loadDriverTemplateRecords(ResourceHOSLogs)
	roster := l.loadDriverRoster()
	ids := selectDriverIDs(driverIDs, templates, roster)
	if len(ids) == 0 {
		return []Record{}
	}

	windowStart, windowEnd := resolveWindow(now, startTime, endTime, 24*time.Hour)
	assets := l.loadAssetMetadata()
	out := make([]Record, 0, len(ids))
	for _, driverID := range ids {
		base := cloneRecord(templates[driverID])
		if len(base) == 0 {
			base = Record{}
		}
		logs := l.generateDriverLogs(
			driverID,
			roster[driverID],
			assets,
			windowStart,
			windowEnd,
			now,
		)

		record := Record{
			"driver": map[string]any{
				"id":   driverID,
				"name": roster[driverID].Name,
			},
			"hosLogs": logs,
		}
		overlayRecord(base, record)
		out = append(out, base)
	}

	sort.Slice(out, func(i, j int) bool {
		return nestedString(out[i], "driver", "id") < nestedString(out[j], "driver", "id")
	})
	return out
}

func (l *LiveSimulator) generateDriverLogs(
	driverID string,
	roster driverRoster,
	assets map[string]Record,
	windowStart time.Time,
	windowEnd time.Time,
	now time.Time,
) []any {
	logs := make([]any, 0, 16)
	timeline := l.driverTimelineSegments(
		driverID,
		windowStart.Add(-24*time.Hour),
		windowEnd.Add(24*time.Hour),
		now,
	)
	pairEvents := l.pairEventsInWindow(
		driverID,
		strings.TrimSpace(roster.VehicleID),
		windowStart.Add(-24*time.Hour),
		windowEnd.Add(24*time.Hour),
	)
	for _, segment := range timeline {
		if segment.Start.Before(windowStart) || segment.Start.After(windowEnd) {
			continue
		}

		vehicleID := strings.TrimSpace(roster.VehicleID)
		primaryEvent := pickPrimarySimEventAt(pairEvents, segment.Start)
		status := normalizeDutyStatusForVehicle(
			segment.Status,
			vehicleID,
			l.shouldForceDriving(vehicleID, segment.Start, now),
		)
		if primaryEvent != nil {
			status = dutyStatusForSimEvent(primaryEvent, status)
		}

		entry := map[string]any{
			"hosStatusType": status,
			"logStartTime":  segment.Start.UTC().Format(time.RFC3339),
		}
		if segment.End.Before(now) {
			entry["logEndTime"] = segment.End.UTC().Format(time.RFC3339)
		}

		if vehicleID != "" && isOnDutyStatus(status) {
			entry["vehicle"] = map[string]any{
				"id":   vehicleID,
				"name": vehicleIDToName(vehicleID, assets),
			}
			state := l.routeStateForSingleTime(vehicleID, segment.Start, now)
			entry["logRecordedLocation"] = map[string]any{
				"latitude":  round(state.Latitude, 6),
				"longitude": round(state.Longitude, 6),
			}
		}
		if primaryEvent != nil && strings.HasPrefix(primaryEvent.Type, "hos.violation.") {
			entry["remark"] = "Simulated HOS violation: " + primaryEvent.Type
		}
		logs = append(logs, entry)
	}

	sort.Slice(logs, func(i, j int) bool {
		left, _ := logs[i].(map[string]any)
		right, _ := logs[j].(map[string]any)
		return stringValue(left, "logStartTime") < stringValue(right, "logStartTime")
	})
	return logs
}

func (l *LiveSimulator) routeStateForSingleTime(
	assetID string,
	sampleTime time.Time,
	now time.Time,
) routeState {
	waypoints := l.loadAssetWaypoints()
	points := waypoints[assetID]
	if len(points) == 0 {
		return routeState{}
	}
	base := l.routeStateForSample(assetID, points, sampleTime, sampleTime.Add(-time.Minute), now)
	driverByVehicle := l.driverByVehicleMap()
	eventsByVehicle := l.vehicleEventsForWindow(
		[]string{assetID},
		driverByVehicle,
		sampleTime.Add(-2*time.Hour),
		sampleTime.Add(2*time.Hour),
	)
	return l.applyVehicleEventsToRouteState(
		assetID,
		points,
		eventsByVehicle[assetID],
		sampleTime,
		sampleTime.Add(-time.Minute),
		now,
		base,
	)
}

func (l *LiveSimulator) routeStateForSample(
	assetID string,
	points []routePoint,
	sampleTime time.Time,
	windowStart time.Time,
	now time.Time,
) routeState {
	if len(points) == 0 {
		return routeState{}
	}
	if len(points) == 1 {
		return routeState{
			Latitude:  points[0].Latitude,
			Longitude: points[0].Longitude,
			Heading:   normalizeHeading(points[0].Heading),
			SpeedMPS:  clampFloat64(points[0].SpeedMPS, minRouteSpeedMPS, maxRouteSpeedMPS),
			Address:   cloneMap(points[0].Address),
		}
	}

	segments, period := buildRouteSegments(points)
	if len(segments) == 0 || period <= 0 {
		return routeState{
			Latitude:  points[0].Latitude,
			Longitude: points[0].Longitude,
			Heading:   normalizeHeading(points[0].Heading),
			SpeedMPS:  clampFloat64(points[0].SpeedMPS, minRouteSpeedMPS, maxRouteSpeedMPS),
			Address:   cloneMap(points[0].Address),
		}
	}
	segments, period = l.tuneRouteSegmentsForDuration(assetID, segments, period)

	offset := l.phaseOffset(assetID, period)
	elapsed := now.Sub(l.startedAt) + sampleTime.Sub(windowStart) + offset
	phase := normalizePhase(elapsed, period)
	segment := segmentForPhase(segments, phase)
	if segment == nil {
		return routeState{}
	}

	segmentFraction := 0.0
	if segment.Duration > 0 {
		segmentFraction = float64(phase-segment.CumulativeStart) / float64(segment.Duration)
	}
	segmentFraction = clampFloat64(segmentFraction, 0, 1)

	lat := lerp(segment.From.Latitude, segment.To.Latitude, segmentFraction)
	lon := lerp(segment.From.Longitude, segment.To.Longitude, segmentFraction)

	heading := bearingDegrees(
		segment.From.Latitude,
		segment.From.Longitude,
		segment.To.Latitude,
		segment.To.Longitude,
	)
	if heading == 0 {
		heading = lerpHeading(segment.From.Heading, segment.To.Heading, segmentFraction)
	}

	varianceKey := assetID + "|" + sampleTime.UTC().
		Format(time.RFC3339) +
		"|" + now.UTC().
		Truncate(2*time.Second).
		Format(time.RFC3339)
	speedVariance := 0.96 + 0.08*l.hashFraction(varianceKey)
	speed := clampFloat64(segment.SpeedMPS*speedVariance, minRouteSpeedMPS, maxRouteSpeedMPS)

	jitterKey := assetID + "|" + sampleTime.UTC().
		Format(time.RFC3339) +
		"|" + now.UTC().
		Truncate(3*time.Second).
		Format(time.RFC3339)
	jitterMeters := (l.hashFraction(jitterKey) - 0.5) * (2 * defaultGPSJitterMeters)
	latJitter := jitterMeters / metersPerDegreeLatitude
	lonJitter := 0.0
	lonMetersPerDegree := metersPerDegreeLongitude(lat)
	if lonMetersPerDegree > 0 {
		lonJitter = (jitterMeters * 0.65) / lonMetersPerDegree
	}

	return routeState{
		Latitude:  lat + latJitter,
		Longitude: lon - lonJitter,
		Heading:   normalizeHeading(heading),
		SpeedMPS:  speed,
		Address:   chooseAddress(segment.From.Address, segment.To.Address, segmentFraction),
	}
}

func buildRouteSegments(points []routePoint) ([]routeSegment, time.Duration) {
	if len(points) < 2 {
		return []routeSegment{}, 0
	}

	segments := make([]routeSegment, 0, 2*len(points))
	cumulative := time.Duration(0)
	appendSegment := func(from routePoint, to routePoint) {
		distance := haversineMeters(from.Latitude, from.Longitude, to.Latitude, to.Longitude)
		if distance < minSegmentDistanceMeters {
			distance = minSegmentDistanceMeters
		}

		targetSpeed := clampFloat64(
			(from.SpeedMPS+to.SpeedMPS)/2,
			minRouteSpeedMPS,
			maxRouteSpeedMPS,
		)
		durationSeconds := distance / targetSpeed
		duration := time.Duration(durationSeconds * float64(time.Second))
		if duration < time.Second {
			duration = time.Second
		}

		segments = append(segments, routeSegment{
			From:            from,
			To:              to,
			DistanceMeters:  distance,
			SpeedMPS:        targetSpeed,
			Duration:        duration,
			CumulativeStart: cumulative,
		})
		cumulative += duration
	}

	// Move forward through the route.
	for idx := 0; idx < len(points)-1; idx++ {
		appendSegment(points[idx], points[idx+1])
	}

	// Return on the same road geometry instead of jumping from end to start.
	for idx := len(points) - 1; idx > 0; idx-- {
		appendSegment(points[idx], points[idx-1])
	}
	return segments, cumulative
}

func (l *LiveSimulator) tuneRouteSegmentsForDuration(
	assetID string,
	segments []routeSegment,
	period time.Duration,
) ([]routeSegment, time.Duration) {
	if len(segments) == 0 || period <= 0 {
		return segments, period
	}

	target := l.routeLoopTargetPeriod(assetID)
	if target <= 0 || period >= target {
		return segments, period
	}

	scale := float64(target) / float64(period)
	if scale <= 1 {
		return segments, period
	}

	scaled := make([]routeSegment, 0, len(segments))
	cumulative := time.Duration(0)
	for idx := range segments {
		segment := segments[idx]
		duration := time.Duration(float64(segment.Duration) * scale)
		if duration < time.Second {
			duration = time.Second
		}
		speedMPS := segment.DistanceMeters / duration.Seconds()
		speedMPS = clampFloat64(speedMPS, minRouteSpeedMPS, maxRouteSpeedMPS)
		scaled = append(scaled, routeSegment{
			From:            segment.From,
			To:              segment.To,
			DistanceMeters:  segment.DistanceMeters,
			SpeedMPS:        speedMPS,
			Duration:        duration,
			CumulativeStart: cumulative,
		})
		cumulative += duration
	}
	return scaled, cumulative
}

func (l *LiveSimulator) routeLoopTargetPeriod(assetID string) time.Duration {
	minHours := l.options.TripHoursMin
	maxHours := l.options.TripHoursMax
	if minHours < minLoopFloorHours {
		minHours = minLoopFloorHours
	}
	if maxHours < minHours {
		maxHours = minHours
	}

	targetHours := minHours
	if maxHours > minHours {
		span := maxHours - minHours
		offset := int(
			float64(span+1) * l.hashFraction("loop-period-hours|"+strings.TrimSpace(assetID)),
		)
		if offset > span {
			offset = span
		}
		targetHours += offset
	}
	return time.Duration(targetHours) * time.Hour
}

func segmentForPhase(segments []routeSegment, phase time.Duration) *routeSegment {
	for idx := range segments {
		segment := &segments[idx]
		end := segment.CumulativeStart + segment.Duration
		if phase < end {
			return segment
		}
	}
	if len(segments) == 0 {
		return nil
	}
	return &segments[len(segments)-1]
}

func normalizePhase(elapsed, period time.Duration) time.Duration {
	if period <= 0 {
		return 0
	}
	for elapsed < 0 {
		elapsed += period
	}
	return elapsed % period
}

func (l *LiveSimulator) loadAssetWaypoints() map[string][]routePoint {
	records, err := l.store.List(ResourceAssetLocation)
	if err != nil {
		return map[string][]routePoint{}
	}

	out := map[string][]routePoint{}
	for _, record := range records {
		assetID := nestedString(record, "asset", "id")
		if assetID == "" {
			continue
		}
		location, ok := anyAsMap(record["location"])
		if !ok {
			continue
		}
		point := routePoint{
			Latitude:  floatFromAny(location["latitude"]),
			Longitude: floatFromAny(location["longitude"]),
			Heading:   floatFromAny(location["headingDegrees"]),
			SpeedMPS:  defaultAssetSpeedMPS,
		}
		if address, okAddress := anyAsMap(location["address"]); okAddress {
			point.Address = cloneMap(address)
		}
		if speed, okSpeed := anyAsMap(record["speed"]); okSpeed {
			point.SpeedMPS = maxFloat64(
				floatFromAny(speed["gpsSpeedMetersPerSecond"]),
				floatFromAny(speed["ecuSpeedMetersPerSecond"]),
			)
		}
		if point.SpeedMPS <= 0 {
			point.SpeedMPS = defaultAssetSpeedMPS
		}
		out[assetID] = append(out[assetID], point)
	}

	for assetID := range out {
		if len(out[assetID]) < 2 {
			continue
		}
	}
	return out
}

func (l *LiveSimulator) loadAssetMetadata() map[string]Record {
	records, err := l.store.List(ResourceAssets)
	if err != nil {
		return map[string]Record{}
	}
	out := map[string]Record{}
	for _, record := range records {
		id := recordID(record)
		if id != "" {
			out[id] = record
		}
	}
	return out
}

func (l *LiveSimulator) loadVehicleStatsTemplates() map[string]Record {
	records, err := l.store.List(ResourceVehicleStats)
	if err != nil {
		return map[string]Record{}
	}
	out := map[string]Record{}
	for _, record := range records {
		id := recordID(record)
		if id != "" {
			out[id] = record
		}
	}
	return out
}

func (l *LiveSimulator) loadDriverTemplateRecords(resource Resource) map[string]Record {
	records, err := l.store.List(resource)
	if err != nil {
		return map[string]Record{}
	}
	out := map[string]Record{}
	for _, record := range records {
		driverID := nestedString(record, "driver", "id")
		if driverID != "" {
			out[driverID] = record
		}
	}
	return out
}

func (l *LiveSimulator) loadDriverRoster() map[string]driverRoster {
	drivers, err := l.store.List(ResourceDrivers)
	if err != nil {
		drivers = []Record{}
	}
	roster := map[string]driverRoster{}
	for _, driver := range drivers {
		id := recordID(driver)
		if id == "" {
			continue
		}
		roster[id] = driverRoster{
			Name: stringValue(driver, "name"),
		}
	}

	clockTemplates := l.loadDriverTemplateRecords(ResourceHOSClocks)
	for driverID, record := range clockTemplates {
		vehicleID := nestedString(record, "currentVehicle", "id")
		entry := roster[driverID]
		if entry.Name == "" {
			entry.Name = nestedString(record, "driver", "name")
		}
		entry.VehicleID = firstNonEmpty(entry.VehicleID, vehicleID)
		roster[driverID] = entry
	}

	routes, err := l.store.List(ResourceRoutes)
	if err == nil {
		for _, route := range routes {
			driverID := nestedString(route, "driver", "id")
			if driverID == "" {
				continue
			}
			entry := roster[driverID]
			entry.VehicleID = firstNonEmpty(entry.VehicleID, nestedString(route, "vehicle", "id"))
			entry.Name = firstNonEmpty(entry.Name, nestedString(route, "driver", "name"))
			roster[driverID] = entry
		}
	}

	for driverID, entry := range roster {
		if strings.TrimSpace(entry.Name) == "" {
			entry.Name = driverID
			roster[driverID] = entry
		}
	}
	l.assignVehiclesToUnassignedDrivers(roster)
	return roster
}

func (l *LiveSimulator) assignVehiclesToUnassignedDrivers(roster map[string]driverRoster) {
	if len(roster) == 0 {
		return
	}

	assignedVehicles := map[string]struct{}{}
	for _, entry := range roster {
		vehicleID := strings.TrimSpace(entry.VehicleID)
		if vehicleID != "" {
			assignedVehicles[vehicleID] = struct{}{}
		}
	}

	availableVehicleIDs := l.listKnownVehicleIDs()
	candidates := make([]string, 0, len(availableVehicleIDs))
	for _, vehicleID := range availableVehicleIDs {
		if _, alreadyAssigned := assignedVehicles[vehicleID]; alreadyAssigned {
			continue
		}
		candidates = append(candidates, vehicleID)
	}
	if len(candidates) == 0 {
		return
	}

	driverIDs := make([]string, 0, len(roster))
	for driverID, entry := range roster {
		if strings.TrimSpace(entry.VehicleID) == "" {
			driverIDs = append(driverIDs, driverID)
		}
	}
	sort.Strings(driverIDs)
	for idx, driverID := range driverIDs {
		if idx >= len(candidates) {
			break
		}
		entry := roster[driverID]
		entry.VehicleID = candidates[idx]
		roster[driverID] = entry
	}
}

func (l *LiveSimulator) listKnownVehicleIDs() []string {
	assets := l.loadAssetMetadata()
	waypoints := l.loadAssetWaypoints()
	templates := l.loadVehicleStatsTemplates()

	candidates := make([]string, 0, len(assets)+len(waypoints)+len(templates))
	for id, record := range assets {
		if strings.EqualFold(stringValue(record, "type"), "vehicle") {
			candidates = append(candidates, id)
		}
	}
	for id := range waypoints {
		candidates = append(candidates, id)
	}
	for id := range templates {
		candidates = append(candidates, id)
	}
	sort.Strings(candidates)
	return uniqueStrings(candidates)
}

func (l *LiveSimulator) driverByVehicleMap() map[string]string {
	roster := l.loadDriverRoster()
	out := make(map[string]string, len(roster))
	for driverID, entry := range roster {
		vehicleID := strings.TrimSpace(entry.VehicleID)
		if vehicleID == "" {
			continue
		}
		if _, exists := out[vehicleID]; exists {
			continue
		}
		out[vehicleID] = driverID
	}
	return out
}

func (l *LiveSimulator) vehicleEventsForWindow(
	vehicleIDs []string,
	driverByVehicle map[string]string,
	windowStart time.Time,
	windowEnd time.Time,
) map[string][]SimEvent {
	ids := uniqueStrings(vehicleIDs)
	out := make(map[string][]SimEvent, len(ids))
	for _, vehicleID := range ids {
		driverID := strings.TrimSpace(driverByVehicle[vehicleID])
		if driverID == "" {
			out[vehicleID] = []SimEvent{}
			continue
		}
		out[vehicleID] = l.pairEventsInWindow(driverID, vehicleID, windowStart, windowEnd)
	}
	return out
}

func (l *LiveSimulator) applyVehicleEventsToRouteState(
	vehicleID string,
	points []routePoint,
	events []SimEvent,
	sampleTime time.Time,
	windowStart time.Time,
	now time.Time,
	base routeState,
) routeState {
	primaryEvent := pickPrimarySimEventAt(events, sampleTime)
	if primaryEvent == nil {
		return base
	}

	switch primaryEvent.Type {
	case simEventStopFuelBreak, simEventDutyOffDutyPause, simEventDutySleeperBlock:
		frozen := l.routeStateForSample(
			vehicleID,
			points,
			primaryEvent.StartsAt,
			windowStart,
			now,
		)
		frozen.SpeedMPS = 0
		return frozen
	case simEventStopTrafficDelay:
		frozen := l.routeStateForSample(
			vehicleID,
			points,
			primaryEvent.StartsAt,
			windowStart,
			now,
		)
		speedMultiplier := floatFromAny(primaryEvent.Metadata["speedMultiplier"])
		if speedMultiplier <= 0 {
			speedMultiplier = 0.35
		}
		frozen.SpeedMPS = clampFloat64(base.SpeedMPS*speedMultiplier, 0.4, maxRouteSpeedMPS*0.45)
		return frozen
	case simEventSpeedMinor, simEventSpeedMajor:
		speedMultiplier := floatFromAny(primaryEvent.Metadata["speedMultiplier"])
		if speedMultiplier <= 0 {
			speedMultiplier = 1.2
		}
		boosted := base
		boosted.SpeedMPS = clampFloat64(boosted.SpeedMPS*speedMultiplier, minRouteSpeedMPS, 40.2)
		return boosted
	default:
		return base
	}
}

func (l *LiveSimulator) primaryDriverEventAt(
	driverID string,
	vehicleID string,
	at time.Time,
) *SimEvent {
	events := l.pairEventsInWindow(
		driverID,
		vehicleID,
		at.Add(-18*time.Hour),
		at.Add(18*time.Hour),
	)
	return pickPrimarySimEventAt(events, at)
}

func (l *LiveSimulator) hashFraction(parts ...string) float64 {
	key := strings.TrimSpace(strings.Join(parts, "|"))
	input := strings.TrimSpace(l.seed) + "|" + key
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(input))
	return float64(hasher.Sum64()%10000) / 10000.0
}

func (l *LiveSimulator) phaseOffset(key string, period time.Duration) time.Duration {
	if period <= 0 {
		return 0
	}
	fraction := l.hashFraction(key)
	return time.Duration(float64(period) * fraction)
}

func (l *LiveSimulator) eldSnapshotForDriver(now time.Time, driverID string) eldSnapshot {
	timeline := l.driverTimelineSegments(
		driverID,
		now.Add(-eldLookbackDuration),
		now.Add(eldLookaheadDuration),
		now,
	)
	if len(timeline) == 0 {
		return eldSnapshot{
			DutyStatus:           hosStatusOffDuty,
			BreakRemaining:       eldBreakLimit,
			DriveRemaining:       eldDriveLimit,
			ShiftRemaining:       eldShiftLimit,
			CycleRemaining:       eldCycleLimit,
			CycleTomorrow:        eldCycleLimit,
			CycleWindowStartedAt: now.Add(-eldCycleWindow),
		}
	}

	currentIdx := timelineSegmentIndexAt(timeline, now)
	currentStatus := hosStatusOffDuty
	if currentIdx >= 0 {
		currentStatus = timeline[currentIdx].Status
	}

	shiftResetAt := lastShiftResetTime(timeline, now)
	driveUsed := durationForStatuses(timeline, shiftResetAt, now, hosStatusDriving)
	shiftElapsed := maxDuration(now.Sub(shiftResetAt), 0)
	drivingSinceBreak := drivingSinceLastBreak(timeline, shiftResetAt, now)

	cycleWindowStart := now.Add(-eldCycleWindow)
	cycleUsed := durationForStatuses(
		timeline,
		cycleWindowStart,
		now,
		hosStatusOnDuty,
		hosStatusDriving,
	)

	tomorrow := now.Add(24 * time.Hour)
	cycleTomorrowUsed := durationForStatuses(
		timeline,
		tomorrow.Add(-eldCycleWindow),
		tomorrow,
		hosStatusOnDuty,
		hosStatusDriving,
	)

	return eldSnapshot{
		DutyStatus:           currentStatus,
		BreakRemaining:       remainingDuration(eldBreakLimit, drivingSinceBreak),
		DriveRemaining:       remainingDuration(eldDriveLimit, driveUsed),
		ShiftRemaining:       remainingDuration(eldShiftLimit, shiftElapsed),
		CycleRemaining:       remainingDuration(eldCycleLimit, cycleUsed),
		CycleTomorrow:        remainingDuration(eldCycleLimit, cycleTomorrowUsed),
		CycleWindowStartedAt: cycleWindowStart.UTC(),
		CycleViolation:       durationOver(cycleUsed, eldCycleLimit),
		ShiftDrivingViolation: maxDuration(
			durationOver(driveUsed, eldDriveLimit),
			durationOver(shiftElapsed, eldShiftLimit),
		),
	}
}

func (l *LiveSimulator) driverTimelineSegments(
	driverID string,
	windowStart time.Time,
	windowEnd time.Time,
	now time.Time,
) []timelineSegment {
	if windowEnd.Before(windowStart) {
		windowStart, windowEnd = windowEnd, windowStart
	}
	windowStart = windowStart.UTC()
	windowEnd = windowEnd.UTC()

	pattern := weeklyDutyPattern()
	period := patternPeriod(pattern)
	if period <= 0 {
		return []timelineSegment{}
	}

	// Keep drivers in the 5 work days of the weekly pattern at anchor time.
	workSpan := 5 * 24 * time.Hour
	offset := l.phaseOffset("eld|"+driverID, workSpan)
	phase := normalizePhase(now.UTC().Sub(l.anchorTime)+offset, period)
	periodStart := now.UTC().Add(-phase)
	for periodStart.After(windowStart) {
		periodStart = periodStart.Add(-period)
	}

	out := make([]timelineSegment, 0, 128)
	for cycleStart := periodStart; cycleStart.Before(windowEnd); cycleStart = cycleStart.Add(period) {
		cursor := cycleStart
		for _, segment := range pattern {
			next := cursor.Add(segment.Duration)
			if next.After(windowStart) && cursor.Before(windowEnd) {
				out = append(out, timelineSegment{
					Status: segment.Status,
					Start:  cursor,
					End:    next,
				})
			}
			cursor = next
		}
	}

	return out
}

func weeklyDutyPattern() []dutySegment {
	// ELD-oriented workday:
	// - 10h off-duty reset.
	// - 11h on-duty/driving window with a 30m qualifying break.
	// - remaining rest to complete 24h.
	workday := []dutySegment{
		{Status: hosStatusOffDuty, Duration: 10 * time.Hour},
		{Status: hosStatusOnDuty, Duration: 30 * time.Minute},
		{Status: hosStatusDriving, Duration: 4 * time.Hour},
		{Status: hosStatusOnDuty, Duration: 15 * time.Minute},
		{Status: hosStatusDriving, Duration: 3*time.Hour + 30*time.Minute},
		{Status: hosStatusOffDuty, Duration: 30 * time.Minute},
		{Status: hosStatusDriving, Duration: 2 * time.Hour},
		{Status: hosStatusOnDuty, Duration: 45 * time.Minute},
		{Status: hosStatusOffDuty, Duration: 2*time.Hour + 30*time.Minute},
	}

	weekly := make([]dutySegment, 0, len(workday)*5+2)
	for day := 0; day < 5; day++ {
		weekly = append(weekly, workday...)
	}
	weekly = append(
		weekly,
		dutySegment{Status: hosStatusSleeperBed, Duration: 24 * time.Hour},
		dutySegment{Status: hosStatusOffDuty, Duration: 24 * time.Hour},
	)
	return weekly
}

func patternPeriod(segments []dutySegment) time.Duration {
	total := time.Duration(0)
	for _, segment := range segments {
		total += segment.Duration
	}
	return total
}

func timelineSegmentIndexAt(segments []timelineSegment, at time.Time) int {
	if len(segments) == 0 {
		return -1
	}
	for idx, segment := range segments {
		if (at.Equal(segment.Start) || at.After(segment.Start)) && at.Before(segment.End) {
			return idx
		}
	}
	if at.Before(segments[0].Start) {
		return 0
	}
	return len(segments) - 1
}

func lastShiftResetTime(segments []timelineSegment, at time.Time) time.Time {
	idx := timelineSegmentIndexAt(segments, at)
	if idx < 0 {
		return at
	}

	cursor := at
	offDutyStreak := time.Duration(0)
	offDutyStreakEnd := at

	for i := idx; i >= 0; i-- {
		segment := segments[i]
		segmentEnd := minTime(segment.End, cursor)
		if !segmentEnd.After(segment.Start) {
			continue
		}
		segmentDuration := segmentEnd.Sub(segment.Start)

		if isOffDutyStatus(segment.Status) {
			if offDutyStreak == 0 {
				offDutyStreakEnd = segmentEnd
			}
			offDutyStreak += segmentDuration
			cursor = segment.Start
			continue
		}

		if offDutyStreak >= eldShiftResetDuration {
			return offDutyStreakEnd
		}

		offDutyStreak = 0
		cursor = segment.Start
	}

	if offDutyStreak >= eldShiftResetDuration {
		return offDutyStreakEnd
	}
	return at
}

func drivingSinceLastBreak(
	segments []timelineSegment,
	lowerBound time.Time,
	at time.Time,
) time.Duration {
	if !at.After(lowerBound) {
		return 0
	}

	idx := timelineSegmentIndexAt(segments, at)
	if idx < 0 {
		return 0
	}

	cursor := at
	nonDrivingStreak := time.Duration(0)
	drivingDuration := time.Duration(0)

	for i := idx; i >= 0; i-- {
		segment := segments[i]
		segmentStart := maxTime(segment.Start, lowerBound)
		segmentEnd := minTime(segment.End, cursor)
		if !segmentEnd.After(segmentStart) {
			continue
		}
		duration := segmentEnd.Sub(segmentStart)

		if segment.Status == hosStatusDriving {
			if nonDrivingStreak >= eldBreakResetDuration {
				return drivingDuration
			}
			drivingDuration += duration
			nonDrivingStreak = 0
		} else {
			nonDrivingStreak += duration
			if nonDrivingStreak >= eldBreakResetDuration {
				return drivingDuration
			}
		}

		cursor = segmentStart
		if !cursor.After(lowerBound) {
			break
		}
	}

	return drivingDuration
}

func durationForStatuses(
	segments []timelineSegment,
	windowStart time.Time,
	windowEnd time.Time,
	statuses ...string,
) time.Duration {
	if !windowEnd.After(windowStart) || len(statuses) == 0 {
		return 0
	}

	allowed := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status] = struct{}{}
	}

	total := time.Duration(0)
	for _, segment := range segments {
		if _, ok := allowed[segment.Status]; !ok {
			continue
		}

		overlapStart := maxTime(segment.Start, windowStart)
		overlapEnd := minTime(segment.End, windowEnd)
		if overlapEnd.After(overlapStart) {
			total += overlapEnd.Sub(overlapStart)
		}
	}
	return total
}

func isOffDutyStatus(status string) bool {
	return status == hosStatusOffDuty || status == hosStatusSleeperBed
}

func isOnDutyStatus(status string) bool {
	return status == hosStatusOnDuty || status == hosStatusDriving
}

func applyHOSSnapshotEvent(snapshot *eldSnapshot, event *SimEvent, now time.Time) {
	elapsed := maxDuration(now.Sub(event.StartsAt), 0)
	switch event.Type {
	case simEventViolationBreak:
		snapshot.BreakRemaining = 0
		snapshot.ShiftDrivingViolation = maxDuration(snapshot.ShiftDrivingViolation, elapsed)
	case simEventViolationDrive:
		snapshot.DriveRemaining = 0
		snapshot.ShiftDrivingViolation = maxDuration(snapshot.ShiftDrivingViolation, elapsed)
	case simEventViolationShift:
		snapshot.ShiftRemaining = 0
		snapshot.ShiftDrivingViolation = maxDuration(snapshot.ShiftDrivingViolation, elapsed)
	case simEventViolationCycle:
		snapshot.CycleRemaining = 0
		snapshot.CycleViolation = maxDuration(snapshot.CycleViolation, elapsed)
	}
}

func dutyStatusForSimEvent(event *SimEvent, fallback string) string {
	switch event.Type {
	case simEventDutyOffDutyPause:
		return hosStatusOffDuty
	case simEventDutySleeperBlock:
		return hosStatusSleeperBed
	case simEventStopTrafficDelay, simEventStopFuelBreak:
		return hosStatusOnDuty
	case simEventSpeedMinor, simEventSpeedMajor:
		return hosStatusDriving
	default:
		return fallback
	}
}

func pickPrimarySimEventAt(events []SimEvent, at time.Time) *SimEvent {
	var selected *SimEvent
	selectedPriority := -1
	for idx := range events {
		event := &events[idx]
		if !event.IsActive(at.UTC()) {
			continue
		}
		priority := simEventPriority(event.Type)
		if selected == nil || priority > selectedPriority {
			selected = event
			selectedPriority = priority
		}
	}
	return selected
}

func simEventPriority(eventType string) int {
	switch eventType {
	case simEventDutySleeperBlock:
		return 100
	case simEventDutyOffDutyPause:
		return 90
	case simEventStopFuelBreak:
		return 80
	case simEventStopTrafficDelay:
		return 70
	case simEventSpeedMajor:
		return 60
	case simEventSpeedMinor:
		return 50
	case simEventViolationCycle:
		return 45
	case simEventViolationShift:
		return 44
	case simEventViolationDrive:
		return 43
	case simEventViolationBreak:
		return 42
	default:
		return 0
	}
}

func normalizeDutyStatusForVehicle(status, vehicleID string, vehicleMoving bool) string {
	if strings.TrimSpace(vehicleID) == "" {
		if status == hosStatusDriving {
			return hosStatusOffDuty
		}
		return status
	}
	if vehicleMoving {
		return hosStatusDriving
	}
	return status
}

func (l *LiveSimulator) shouldForceDriving(
	vehicleID string,
	sampleTime time.Time,
	now time.Time,
) bool {
	cleanVehicleID := strings.TrimSpace(vehicleID)
	if cleanVehicleID == "" {
		return false
	}
	state := l.routeStateForSingleTime(cleanVehicleID, sampleTime, now)
	return state.SpeedMPS > movingSpeedThresholdMPS
}

func remainingDuration(limit, used time.Duration) time.Duration {
	return maxDuration(limit-used, 0)
}

func durationOver(used, limit time.Duration) time.Duration {
	return maxDuration(used-limit, 0)
}

func minTime(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

func maxTime(left, right time.Time) time.Time {
	if left.After(right) {
		return left
	}
	return right
}

func resolveWindow(
	now time.Time,
	startTime *time.Time,
	endTime *time.Time,
	defaultLookback time.Duration,
) (start, end time.Time) {
	end = now.UTC()
	if endTime != nil {
		end = endTime.UTC()
	}
	start = end.Add(-defaultLookback)
	if startTime != nil {
		start = startTime.UTC()
	}
	if start.After(end) {
		start, end = end, start
	}
	return start, end
}

func sampleTimes(
	start time.Time,
	end time.Time,
	defaultStep time.Duration,
	maxSamples int,
) []time.Time {
	if !start.Before(end) {
		return []time.Time{start}
	}
	if maxSamples <= 1 {
		return []time.Time{start}
	}

	span := end.Sub(start)
	step := defaultStep
	if step <= 0 {
		step = time.Minute
	}

	estimated := int(span/step) + 1
	if estimated > maxSamples {
		step = time.Duration(float64(span) / float64(maxSamples-1))
		if step < 30*time.Second {
			step = 30 * time.Second
		}
	}

	result := make([]time.Time, 0, maxSamples)
	for cursor := start; !cursor.After(end) && len(result) < maxSamples; cursor = cursor.Add(step) {
		result = append(result, cursor.UTC())
	}
	if len(result) == 0 || !result[len(result)-1].Equal(end.UTC()) {
		result = append(result, end.UTC())
	}
	return result
}

func selectAssetIDs(assetIDs []string, waypoints map[string][]routePoint) []string {
	if len(assetIDs) > 0 {
		selected := make([]string, 0, len(assetIDs))
		for _, assetID := range assetIDs {
			clean := strings.TrimSpace(assetID)
			if clean == "" {
				continue
			}
			if _, ok := waypoints[clean]; ok {
				selected = append(selected, clean)
			}
		}
		sort.Strings(selected)
		return selected
	}

	selected := make([]string, 0, len(waypoints))
	for assetID := range waypoints {
		selected = append(selected, assetID)
	}
	sort.Strings(selected)
	return selected
}

func selectVehicleIDs(
	filterIDs []string,
	templates map[string]Record,
	waypoints map[string][]routePoint,
	assets map[string]Record,
) []string {
	if len(filterIDs) > 0 {
		selected := make([]string, 0, len(filterIDs))
		for _, id := range filterIDs {
			clean := strings.TrimSpace(id)
			if clean != "" {
				selected = append(selected, clean)
			}
		}
		sort.Strings(selected)
		return uniqueStrings(selected)
	}

	selected := make([]string, 0, len(templates)+len(waypoints)+len(assets))
	for id := range templates {
		selected = append(selected, id)
	}
	for id := range waypoints {
		selected = append(selected, id)
	}
	for id, record := range assets {
		if strings.EqualFold(stringValue(record, "type"), "vehicle") {
			selected = append(selected, id)
		}
	}
	sort.Strings(selected)
	return uniqueStrings(selected)
}

func selectDriverIDs(
	filterIDs []string,
	templates map[string]Record,
	roster map[string]driverRoster,
) []string {
	if len(filterIDs) > 0 {
		selected := make([]string, 0, len(filterIDs))
		for _, id := range filterIDs {
			clean := strings.TrimSpace(id)
			if clean != "" {
				selected = append(selected, clean)
			}
		}
		sort.Strings(selected)
		return uniqueStrings(selected)
	}

	selected := make([]string, 0, len(templates)+len(roster))
	for id := range templates {
		selected = append(selected, id)
	}
	for id := range roster {
		selected = append(selected, id)
	}
	sort.Strings(selected)
	return uniqueStrings(selected)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func dynamicFuelPercent(
	base Record,
	now time.Time,
	startedAt time.Time,
	hashFactor float64,
) int64 {
	baseValue := int64(floatFromAny(nestedAny(base, "fuelPercent", "value")))
	if baseValue <= 0 {
		baseValue = int64(70 + hashFactor*20)
	}
	drain := int64((now.Sub(startedAt).Minutes()/7.5)*(0.6+hashFactor*0.6)) % 55
	return clampInt64(baseValue-drain, 8, 100)
}

func dynamicEngineSeconds(
	base Record,
	now time.Time,
	startedAt time.Time,
	hashFactor float64,
) int64 {
	baseValue := int64(floatFromAny(nestedAny(base, "obdEngineSeconds", "value")))
	if baseValue <= 0 {
		baseValue = int64(5_000_000 + hashFactor*2_000_000)
	}
	increment := int64(now.Sub(startedAt).Seconds() * (0.72 + hashFactor*0.35))
	return baseValue + maxInt64(increment, 0)
}

func dynamicBatteryMilliVolts(base Record, now time.Time, hashFactor float64) int64 {
	baseValue := int64(floatFromAny(nestedAny(base, "batteryMilliVolts", "value")))
	if baseValue <= 0 {
		baseValue = int64(12_700 + hashFactor*600)
	}
	swing := int64(120 * math.Sin(float64(now.Unix())/180+hashFactor*math.Pi*2))
	return clampInt64(baseValue+swing, 11_800, 13_900)
}

func vehicleName(vehicleID string, base Record, assets map[string]Record) string {
	name := strings.TrimSpace(stringValue(base, "name"))
	if name != "" {
		return name
	}
	if asset, ok := assets[vehicleID]; ok {
		assetName := strings.TrimSpace(stringValue(asset, "name"))
		if assetName != "" {
			return assetName
		}
	}
	return vehicleID
}

func vehicleIDToName(vehicleID string, assets map[string]Record) string {
	if asset, ok := assets[vehicleID]; ok {
		name := strings.TrimSpace(stringValue(asset, "name"))
		if name != "" {
			return name
		}
	}
	return vehicleID
}

func overlayRecord(target, overlay Record) {
	for key, value := range overlay {
		target[key] = cloneAny(value)
	}
}

func cloneMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = cloneAny(item)
	}
	return out
}

func chooseAddress(
	left map[string]any,
	right map[string]any,
	segmentFraction float64,
) map[string]any {
	if segmentFraction < 0.5 && len(left) > 0 {
		return cloneMap(left)
	}
	if len(right) > 0 {
		return cloneMap(right)
	}
	return cloneMap(left)
}

func nestedAny(record Record, keys ...string) any {
	if len(keys) == 0 {
		return nil
	}

	var current any = record
	for _, key := range keys {
		mapped, ok := anyAsMap(current)
		if !ok {
			return nil
		}
		next, ok := mapped[key]
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

func floatFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	case uint64:
		return float64(typed)
	case uint32:
		return float64(typed)
	default:
		return 0
	}
}

func normalizeHeading(value float64) float64 {
	heading := math.Mod(value, 360)
	if heading < 0 {
		heading += 360
	}
	return heading
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := radians(lat1)
	lat2Rad := radians(lat2)
	deltaLat := radians(lat2 - lat1)
	deltaLon := radians(lon2 - lon1)

	sinLat := math.Sin(deltaLat / 2)
	sinLon := math.Sin(deltaLon / 2)
	a := sinLat*sinLat + math.Cos(lat1Rad)*math.Cos(lat2Rad)*sinLon*sinLon
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}

func bearingDegrees(lat1, lon1, lat2, lon2 float64) float64 {
	if lat1 == lat2 && lon1 == lon2 {
		return 0
	}

	lat1Rad := radians(lat1)
	lat2Rad := radians(lat2)
	deltaLon := radians(lon2 - lon1)

	y := math.Sin(deltaLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) -
		math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(deltaLon)
	heading := degrees(math.Atan2(y, x))
	return normalizeHeading(heading)
}

func metersPerDegreeLongitude(latitude float64) float64 {
	cosine := math.Cos(radians(latitude))
	if cosine <= 0 {
		return 1
	}
	return metersPerDegreeLatitude * cosine
}

func radians(value float64) float64 {
	return value * math.Pi / 180.0
}

func degrees(value float64) float64 {
	return value * 180.0 / math.Pi
}

func lerp(left, right, fraction float64) float64 {
	return left + (right-left)*fraction
}

func lerpHeading(left, right, fraction float64) float64 {
	diff := math.Mod((right-left)+540, 360) - 180
	return left + diff*fraction
}

func round(value float64, precision int) float64 {
	if precision < 0 {
		return value
	}
	power := math.Pow(10, float64(precision))
	return math.Round(value*power) / power
}

func clampFloat64(value, lower, upper float64) float64 {
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}

func maxFloat64(left, right float64) float64 {
	if left >= right {
		return left
	}
	return right
}

func maxInt64(left, right int64) int64 {
	if left >= right {
		return left
	}
	return right
}

func maxDuration(left, right time.Duration) time.Duration {
	if left >= right {
		return left
	}
	return right
}

func clampInt64(value, lower, upper int64) int64 {
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		clean := strings.TrimSpace(value)
		if clean != "" {
			return clean
		}
	}
	return ""
}

func ternary(condition bool, left, right string) string {
	if condition {
		return left
	}
	return right
}
