package sim

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	simEventStopTrafficDelay = "stop.traffic_delay"
	simEventStopFuelBreak    = "stop.fuel_break"
	simEventDutyOffDutyPause = "duty.off_duty_pause"
	simEventDutySleeperBlock = "duty.sleeper_berth_block"
	simEventSpeedMinor       = "speeding.burst_minor"
	simEventSpeedMajor       = "speeding.burst_major"
	simEventViolationBreak   = "hos.violation.break"
	simEventViolationShift   = "hos.violation.shift"
	simEventViolationDrive   = "hos.violation.drive"
	simEventViolationCycle   = "hos.violation.cycle"
)

type SimEvent struct {
	ID        string
	Type      string
	DriverID  string
	VehicleID string
	StartsAt  time.Time
	EndsAt    time.Time
	Severity  string
	Metadata  map[string]any
}

func (e *SimEvent) IsActive(at time.Time) bool {
	return (at.Equal(e.StartsAt) || at.After(e.StartsAt)) && at.Before(e.EndsAt)
}

func (e *SimEvent) Overlaps(start, end time.Time) bool {
	return e.EndsAt.After(start) && e.StartsAt.Before(end)
}

func (e *SimEvent) ToRecord() map[string]any {
	record := map[string]any{
		"id":        e.ID,
		"type":      e.Type,
		"startsAt":  e.StartsAt.UTC().Format(time.RFC3339),
		"endsAt":    e.EndsAt.UTC().Format(time.RFC3339),
		"severity":  e.Severity,
		"driverId":  e.DriverID,
		"vehicleId": e.VehicleID,
	}
	if len(e.Metadata) > 0 {
		record["metadata"] = cloneAny(e.Metadata)
	}
	return record
}

func (l *LiveSimulator) ActiveEvents(
	now time.Time,
	driverIDs []string,
	vehicleIDs []string,
) []SimEvent {
	window := l.EventsWindow(
		now.Add(-18*time.Hour),
		now.Add(18*time.Hour),
		driverIDs,
		vehicleIDs,
		0,
	)
	active := make([]SimEvent, 0, len(window))
	for idx := range window {
		event := &window[idx]
		if event.IsActive(now.UTC()) {
			active = append(active, *event)
		}
	}
	return active
}

func (l *LiveSimulator) EventsWindow(
	start time.Time,
	end time.Time,
	driverIDs []string,
	vehicleIDs []string,
	limit int,
) []SimEvent {
	start, end = normalizeEventWindow(start, end)
	roster := l.loadDriverRoster()
	selectedDriverIDs := l.selectedDriverIDs(roster, driverIDs)
	driverFilter, vehicleFilter := eventFilters(selectedDriverIDs, driverIDs, vehicleIDs)
	events := l.generatedEventsWindow(
		start,
		end,
		selectedDriverIDs,
		roster,
		driverFilter,
		vehicleFilter,
	)
	events = l.mergeScriptedEvents(start, end, events, driverFilter, vehicleFilter)

	sort.Slice(events, func(i, j int) bool {
		if events[i].StartsAt.Equal(events[j].StartsAt) {
			return events[i].ID < events[j].ID
		}
		return events[i].StartsAt.Before(events[j].StartsAt)
	})
	if limit > 0 && len(events) > limit {
		return events[:limit]
	}
	return events
}

func normalizeEventWindow(start, end time.Time) (normalizedStart, normalizedEnd time.Time) {
	if end.Before(start) {
		start, end = end, start
	}
	return start.UTC(), end.UTC()
}

func (l *LiveSimulator) selectedDriverIDs(
	roster map[string]driverRoster,
	driverIDs []string,
) []string {
	selectedDriverIDs := selectDriverIDs(driverIDs, map[string]Record{}, roster)
	if maxFleet := l.options.FleetSize; maxFleet > 0 && len(selectedDriverIDs) > maxFleet {
		selectedDriverIDs = selectedDriverIDs[:maxFleet]
	}
	return selectedDriverIDs
}

func eventFilters(
	selectedDriverIDs []string,
	driverIDs []string,
	vehicleIDs []string,
) (driverFilter, vehicleFilter map[string]struct{}) {
	driverFilter = toStringSet(selectedDriverIDs)
	if len(driverIDs) > 0 {
		driverFilter = toStringSet(driverIDs)
	}
	vehicleFilter = toStringSet(vehicleIDs)
	return driverFilter, vehicleFilter
}

func (l *LiveSimulator) generatedEventsWindow(
	start time.Time,
	end time.Time,
	selectedDriverIDs []string,
	roster map[string]driverRoster,
	driverFilter map[string]struct{},
	vehicleFilter map[string]struct{},
) []SimEvent {
	events := make([]SimEvent, 0, len(selectedDriverIDs)*8)
	for _, driverID := range selectedDriverIDs {
		rosterEntry := roster[driverID]
		vehicleID := strings.TrimSpace(rosterEntry.VehicleID)
		pairEvents := l.pairEventsInWindow(driverID, vehicleID, start, end)
		for idx := range pairEvents {
			event := &pairEvents[idx]
			if !matchesStringFilter(driverFilter, event.DriverID) {
				continue
			}
			if !matchesStringFilter(vehicleFilter, event.VehicleID) {
				continue
			}
			events = append(events, *event)
		}
	}
	return events
}

func matchesStringFilter(filter map[string]struct{}, value string) bool {
	if len(filter) == 0 {
		return true
	}
	_, ok := filter[value]
	return ok
}

func (l *LiveSimulator) mergeScriptedEvents(
	start time.Time,
	end time.Time,
	events []SimEvent,
	driverFilter map[string]struct{},
	vehicleFilter map[string]struct{},
) []SimEvent {
	scriptedEvents := l.scriptedEventsWindow(start, end, driverFilter, vehicleFilter)
	if len(scriptedEvents) == 0 {
		return events
	}

	if l.options.ScriptMode == scriptModeOverride {
		overrideKeys := make(map[string]struct{}, len(scriptedEvents))
		for idx := range scriptedEvents {
			overrideKeys[eventDayKey(&scriptedEvents[idx])] = struct{}{}
		}

		filtered := make([]SimEvent, 0, len(events))
		for idx := range events {
			if _, shouldOverride := overrideKeys[eventDayKey(&events[idx])]; shouldOverride {
				continue
			}
			filtered = append(filtered, events[idx])
		}
		events = filtered
	}

	return append(events, scriptedEvents...)
}

func (l *LiveSimulator) scriptedEventsWindow(
	start time.Time,
	end time.Time,
	driverFilter map[string]struct{},
	vehicleFilter map[string]struct{},
) []SimEvent {
	if l.scripts == nil {
		return []SimEvent{}
	}
	return l.scripts.EventsWindow(start, end, driverFilter, vehicleFilter)
}

func eventDayKey(event *SimEvent) string {
	if event == nil {
		return ""
	}
	return strings.Join([]string{
		event.DriverID,
		event.VehicleID,
		event.StartsAt.UTC().Truncate(24 * time.Hour).Format("2006-01-02"),
	}, "|")
}

func (l *LiveSimulator) ActiveEventSummary(
	now time.Time,
) (byType map[string]int, violations, speeding int) {
	active := l.ActiveEvents(now, nil, nil)
	byType = map[string]int{}
	for idx := range active {
		event := &active[idx]
		byType[event.Type]++
		if strings.HasPrefix(event.Type, "hos.violation.") {
			violations++
		}
		if strings.HasPrefix(event.Type, "speeding.") {
			speeding++
		}
	}
	return byType, violations, speeding
}

func (l *LiveSimulator) EventWebhookType(eventType string) string {
	switch eventType {
	case simEventSpeedMinor, simEventSpeedMajor:
		return "VehicleSpeeding"
	case simEventViolationBreak,
		simEventViolationShift,
		simEventViolationDrive,
		simEventViolationCycle:
		return "DriverHosViolationDetected"
	case simEventStopTrafficDelay:
		return "RouteProgressDelayed"
	case simEventStopFuelBreak:
		return "VehicleStopped"
	case simEventDutyOffDutyPause, simEventDutySleeperBlock:
		return "DriverHosStatusChanged"
	default:
		return "RouteProgressDelayed"
	}
}

func (l *LiveSimulator) EventWebhookPayload(event *SimEvent) map[string]any {
	payload := map[string]any{
		"id":       event.ID,
		"type":     event.Type,
		"startsAt": event.StartsAt.UTC().Format(time.RFC3339),
		"endsAt":   event.EndsAt.UTC().Format(time.RFC3339),
		"severity": event.Severity,
	}
	if strings.TrimSpace(event.DriverID) != "" {
		payload["driver"] = map[string]any{"id": event.DriverID}
	}
	if strings.TrimSpace(event.VehicleID) != "" {
		payload["vehicle"] = map[string]any{"id": event.VehicleID}
	}
	if len(event.Metadata) > 0 {
		payload["metadata"] = cloneAny(event.Metadata)
	}
	return payload
}

func (l *LiveSimulator) pairEventsInWindow(
	driverID string,
	vehicleID string,
	windowStart time.Time,
	windowEnd time.Time,
) []SimEvent {
	startDay := windowStart.Add(-24 * time.Hour).UTC().Truncate(24 * time.Hour)
	endDay := windowEnd.Add(24 * time.Hour).UTC().Truncate(24 * time.Hour)

	events := make([]SimEvent, 0, 16)
	for day := startDay; !day.After(endDay); day = day.Add(24 * time.Hour) {
		daily := l.generateDailyPairEvents(driverID, vehicleID, day)
		for idx := range daily {
			event := &daily[idx]
			if event.Overlaps(windowStart, windowEnd) {
				events = append(events, *event)
			}
		}
	}
	return events
}

func (l *LiveSimulator) generateDailyPairEvents(
	driverID string,
	vehicleID string,
	dayStart time.Time,
) []SimEvent {
	if strings.TrimSpace(driverID) == "" {
		return []SimEvent{}
	}

	ctx := l.buildDailyEventContext(driverID, vehicleID, dayStart)
	appender := newSimEventAppender(dayStart, driverID, vehicleID)
	if strings.TrimSpace(vehicleID) != "" {
		l.appendVehicleOperationalEvents(&ctx, appender)
	}
	l.appendDriverDutyEvents(&ctx, appender)
	l.appendViolationEvents(&ctx, appender)

	events := appender.List()
	sort.Slice(events, func(i, j int) bool {
		if events[i].StartsAt.Equal(events[j].StartsAt) {
			return events[i].Type < events[j].Type
		}
		return events[i].StartsAt.Before(events[j].StartsAt)
	})
	return events
}

type dailyEventContext struct {
	DriverID     string
	VehicleID    string
	DayStart     time.Time
	DayEnd       time.Time
	DrivingStart time.Time
	DrivingEnd   time.Time
	TripHours    int
	Intensity    eventIntensityTuning
}

func (l *LiveSimulator) buildDailyEventContext(
	driverID string,
	vehicleID string,
	dayStart time.Time,
) dailyEventContext {
	intensity := eventIntensityProfile(l.options.EventIntensity)
	tripHours := l.tripHoursForDriver(driverID)
	shiftStart := l.shiftStartForDay(driverID, dayStart)
	drivingStart := shiftStart.Add(
		20*time.Minute + time.Duration(
			40*l.hashFraction("evt|prep|"+driverID+"|"+dayStart.Format("2006-01-02")),
		)*time.Minute,
	)
	drivingEnd := drivingStart.Add(time.Duration(tripHours) * time.Hour)
	dayEnd := dayStart.Add(23*time.Hour + 45*time.Minute)
	if drivingEnd.After(dayEnd) {
		drivingEnd = dayEnd
	}
	if !drivingEnd.After(drivingStart.Add(2 * time.Hour)) {
		drivingEnd = drivingStart.Add(2 * time.Hour)
	}
	return dailyEventContext{
		DriverID:     driverID,
		VehicleID:    vehicleID,
		DayStart:     dayStart,
		DayEnd:       dayEnd,
		DrivingStart: drivingStart,
		DrivingEnd:   drivingEnd,
		TripHours:    tripHours,
		Intensity:    intensity,
	}
}

func (l *LiveSimulator) appendVehicleOperationalEvents(
	ctx *dailyEventContext,
	appender *simEventAppender,
) {
	driverID := ctx.DriverID
	dayKey := ctx.DayStart.Format("2006-01-02")

	trafficScore := l.hashFraction("evt|traffic|" + driverID + "|" + dayKey)
	if trafficScore < ctx.Intensity.TrafficRate {
		offset := time.Duration(
			90+180*l.hashFraction("evt|traffic-offset|"+driverID+"|"+dayKey),
		) * time.Minute
		duration := time.Duration(
			10+26*l.hashFraction("evt|traffic-duration|"+driverID+"|"+dayKey),
		) * time.Minute
		appender.Add(
			simEventStopTrafficDelay,
			clampTime(
				ctx.DrivingStart.Add(offset),
				ctx.DrivingStart,
				ctx.DrivingEnd.Add(-10*time.Minute),
			),
			duration,
			"info",
			map[string]any{"reason": "traffic", "speedMultiplier": 0.35},
		)
	}

	fuelScore := l.hashFraction("evt|fuel|" + driverID + "|" + dayKey)
	if ctx.TripHours >= 6 && fuelScore < ctx.Intensity.FuelRate {
		offset := time.Duration(
			180+120*l.hashFraction("evt|fuel-offset|"+driverID+"|"+dayKey),
		) * time.Minute
		duration := time.Duration(
			20+20*l.hashFraction("evt|fuel-duration|"+driverID+"|"+dayKey),
		) * time.Minute
		appender.Add(
			simEventStopFuelBreak,
			clampTime(
				ctx.DrivingStart.Add(offset),
				ctx.DrivingStart,
				ctx.DrivingEnd.Add(-20*time.Minute),
			),
			duration,
			"info",
			map[string]any{"reason": "fuel_stop", "speedMultiplier": 0},
		)
	}

	speedingRate := clampFloat64(l.options.SpeedingRate*ctx.Intensity.SpeedingMultiplier, 0, 1)
	speedingScore := l.hashFraction("evt|speeding|" + driverID + "|" + dayKey)
	if speedingScore >= speedingRate {
		return
	}

	burstType := simEventSpeedMinor
	severity := "warning"
	multiplier := 1.18
	if l.hashFraction("evt|speeding-major|"+driverID+"|"+dayKey) < 0.34 {
		burstType = simEventSpeedMajor
		severity = "critical"
		multiplier = 1.33
	}
	offset := time.Duration(
		35+int(100*l.hashFraction("evt|speeding-offset|"+driverID+"|"+dayKey)),
	) * time.Minute
	duration := time.Duration(
		8+18*l.hashFraction("evt|speeding-duration|"+driverID+"|"+dayKey),
	) * time.Minute
	appender.Add(
		burstType,
		clampTime(
			ctx.DrivingStart.Add(offset),
			ctx.DrivingStart,
			ctx.DrivingEnd.Add(-8*time.Minute),
		),
		duration,
		severity,
		map[string]any{"speedMultiplier": round(multiplier, 2)},
	)
}

func (l *LiveSimulator) appendDriverDutyEvents(
	ctx *dailyEventContext,
	appender *simEventAppender,
) {
	driverID := ctx.DriverID
	dayKey := ctx.DayStart.Format("2006-01-02")

	offDutyScore := l.hashFraction("evt|off-duty|" + driverID + "|" + dayKey)
	if offDutyScore < ctx.Intensity.OffDutyRate {
		offset := time.Duration(
			25+90*l.hashFraction("evt|off-duty-offset|"+driverID+"|"+dayKey),
		) * time.Minute
		duration := time.Duration(
			45+95*l.hashFraction("evt|off-duty-duration|"+driverID+"|"+dayKey),
		) * time.Minute
		appender.Add(
			simEventDutyOffDutyPause,
			clampTime(
				ctx.DrivingEnd.Add(offset),
				ctx.DrivingStart.Add(2*time.Hour),
				ctx.DayEnd.Add(-45*time.Minute),
			),
			duration,
			"info",
			map[string]any{"dutyStatus": hosStatusOffDuty},
		)
	}

	sleeperScore := l.hashFraction("evt|sleeper|" + driverID + "|" + dayKey)
	if sleeperScore < ctx.Intensity.SleeperRate {
		offset := time.Duration(
			60+120*l.hashFraction("evt|sleeper-offset|"+driverID+"|"+dayKey),
		) * time.Minute
		duration := time.Duration(
			120+240*l.hashFraction("evt|sleeper-duration|"+driverID+"|"+dayKey),
		) * time.Minute
		appender.Add(
			simEventDutySleeperBlock,
			clampTime(
				ctx.DrivingEnd.Add(offset),
				ctx.DrivingStart.Add(3*time.Hour),
				ctx.DayEnd.Add(-2*time.Hour),
			),
			duration,
			"info",
			map[string]any{"dutyStatus": hosStatusSleeperBed},
		)
	}
}

func (l *LiveSimulator) appendViolationEvents(
	ctx *dailyEventContext,
	appender *simEventAppender,
) {
	driverID := ctx.DriverID
	dayKey := ctx.DayStart.Format("2006-01-02")
	violationRate := clampFloat64(l.options.ViolationRate*ctx.Intensity.ViolationMultiplier, 0, 1)
	violationScore := l.hashFraction("evt|violation|" + driverID + "|" + dayKey)
	if violationScore >= violationRate {
		return
	}

	violationTypes := []string{
		simEventViolationBreak,
		simEventViolationShift,
		simEventViolationDrive,
		simEventViolationCycle,
	}
	if strings.TrimSpace(ctx.VehicleID) == "" {
		violationTypes = []string{simEventViolationCycle}
	}

	index := int(
		float64(len(violationTypes)) * l.hashFraction("evt|violation-type|"+driverID+"|"+dayKey),
	)
	if index >= len(violationTypes) {
		index = len(violationTypes) - 1
	}
	violationType := violationTypes[index]

	offset := time.Duration(
		40+180*l.hashFraction("evt|violation-offset|"+driverID+"|"+dayKey),
	) * time.Minute
	duration := time.Duration(
		20+70*l.hashFraction("evt|violation-duration|"+driverID+"|"+dayKey),
	) * time.Minute
	startAt := clampTime(
		ctx.DrivingEnd.Add(-offset),
		ctx.DrivingStart.Add(time.Hour),
		ctx.DayEnd.Add(-20*time.Minute),
	)
	appender.Add(
		violationType,
		startAt,
		duration,
		"warning",
		map[string]any{"category": "hos"},
	)
}

type simEventAppender struct {
	dayStart  time.Time
	driverID  string
	vehicleID string
	indexes   map[string]int
	events    []SimEvent
}

func newSimEventAppender(dayStart time.Time, driverID, vehicleID string) *simEventAppender {
	return &simEventAppender{
		dayStart:  dayStart,
		driverID:  driverID,
		vehicleID: vehicleID,
		indexes:   map[string]int{},
		events:    make([]SimEvent, 0, 12),
	}
}

func (a *simEventAppender) Add(
	eventType string,
	start time.Time,
	duration time.Duration,
	severity string,
	metadata map[string]any,
) {
	if duration <= 0 {
		return
	}
	endsAt := start.Add(duration)
	if !endsAt.After(start) {
		return
	}
	index := a.indexes[eventType]
	a.indexes[eventType] = index + 1
	a.events = append(a.events, SimEvent{
		ID:        buildSimEventID(a.dayStart, a.driverID, a.vehicleID, eventType, index),
		Type:      eventType,
		DriverID:  a.driverID,
		VehicleID: a.vehicleID,
		StartsAt:  start.UTC(),
		EndsAt:    endsAt.UTC(),
		Severity:  severity,
		Metadata:  metadata,
	})
}

func (a *simEventAppender) List() []SimEvent {
	return a.events
}

func (l *LiveSimulator) tripHoursForDriver(driverID string) int {
	minHours := l.options.TripHoursMin
	maxHours := l.options.TripHoursMax
	if maxHours < minHours {
		maxHours = minHours
	}
	if maxHours == minHours {
		return minHours
	}

	rangeHours := maxHours - minHours
	offset := int(float64(rangeHours+1) * l.hashFraction("evt|trip-hours|"+driverID))
	if offset > rangeHours {
		offset = rangeHours
	}
	return minHours + offset
}

func (l *LiveSimulator) shiftStartForDay(driverID string, dayStart time.Time) time.Time {
	offsetHours := 5 + 8*l.hashFraction(
		"evt|shift-start|"+driverID+"|"+dayStart.Format("2006-01-02"),
	)
	return dayStart.Add(time.Duration(offsetHours * float64(time.Hour)))
}

type eventIntensityTuning struct {
	TrafficRate         float64
	FuelRate            float64
	OffDutyRate         float64
	SleeperRate         float64
	SpeedingMultiplier  float64
	ViolationMultiplier float64
}

func eventIntensityProfile(eventIntensity string) eventIntensityTuning {
	switch strings.ToLower(strings.TrimSpace(eventIntensity)) {
	case "compliance":
		return eventIntensityTuning{
			TrafficRate:         0.48,
			FuelRate:            0.82,
			OffDutyRate:         0.42,
			SleeperRate:         0.36,
			SpeedingMultiplier:  0.65,
			ViolationMultiplier: 1.35,
		}
	case "driving":
		return eventIntensityTuning{
			TrafficRate:         0.38,
			FuelRate:            0.7,
			OffDutyRate:         0.28,
			SleeperRate:         0.22,
			SpeedingMultiplier:  1.55,
			ViolationMultiplier: 0.7,
		}
	default:
		return eventIntensityTuning{
			TrafficRate:         0.44,
			FuelRate:            0.78,
			OffDutyRate:         0.34,
			SleeperRate:         0.27,
			SpeedingMultiplier:  1,
			ViolationMultiplier: 1,
		}
	}
}

func buildSimEventID(
	dayStart time.Time,
	driverID string,
	vehicleID string,
	eventType string,
	index int,
) string {
	cleanEventType := strings.NewReplacer(".", "-", "_", "-").Replace(strings.TrimSpace(eventType))
	return fmt.Sprintf(
		"evt-%s-%s-%s-%s-%d",
		dayStart.UTC().Format("20060102"),
		strings.TrimSpace(driverID),
		firstNonEmpty(strings.TrimSpace(vehicleID), "none"),
		cleanEventType,
		index,
	)
}

func clampTime(value, lower, upper time.Time) time.Time {
	if upper.Before(lower) {
		upper = lower
	}
	if value.Before(lower) {
		return lower
	}
	if value.After(upper) {
		return upper
	}
	return value
}

func toStringSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return map[string]struct{}{}
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		clean := strings.TrimSpace(value)
		if clean == "" {
			continue
		}
		out[clean] = struct{}{}
	}
	return out
}
