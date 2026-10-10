package sim

import (
	"math"
	"time"
)

const (
	tankRangeMiles           = 1350.0
	defTankDieselRatioMiles  = 4200.0
	engineRunFraction        = 0.93
	idleRunFraction          = 0.055
	gatewayInstallFallback   = 30 * 24 * time.Hour
	defaultDieselMPGLow      = 6.1
	defaultDieselMPGSpan     = 1.3
	engineStateOn            = "On"
	engineStateOff           = "Off"
	engineStateIdle          = "Idle"
	stoppedIdleProbability   = 0.3
	faultCodeDailyRate       = 0.18
	immobilizerBlipRate      = 0.08
	immobilizerBlipDuration  = 4 * time.Minute
	ignitionEnabled          = "ignition_enabled"
	defaultCanBusType        = "CANBUS_J1939_500"
	reeferDefrostInterval    = 8 * time.Hour
	reeferDefrostDuration    = 20 * time.Minute
	reeferContinuousRunShare = 1.0
	reeferStartStopRunShare  = 0.58
)

var telemetryEpoch = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

type j1939Fault struct {
	SpnID          int64
	FmiID          int64
	SpnDescription string
	FmiDescription string
	Source         string
	Lamp           string
}

var j1939FaultCatalog = []j1939Fault{
	{
		3364,
		1,
		"Aftertreatment 1 DEF Quality",
		"Data Valid But Below Normal Operational Range",
		faultSourceDPF,
		"emissions",
	},
	{
		3251,
		0,
		"Aftertreatment 1 DPF Differential Pressure",
		"Data Valid But Above Normal Operational Range",
		faultSourceDPF,
		faultLampWarning,
	},
	{
		100,
		1,
		"Engine Oil Pressure",
		"Data Valid But Below Normal Operational Range",
		faultSourceEngine,
		"stop",
	},
	{
		110,
		16,
		"Engine Coolant Temperature",
		"Data Valid But Above Normal Operational Range - Moderately Severe Level",
		faultSourceEngine,
		faultLampWarning,
	},
	{1569, 31, "Engine Protection Torque Derate", "Condition Exists", faultSourceEngine, "protect"},
	{
		4364,
		18,
		"Aftertreatment 1 SCR Conversion Efficiency",
		"Data Valid But Below Normal Operating Range - Moderately Severe Level",
		faultSourceDPF,
		"emissions",
	},
	{
		629,
		12,
		"Controller #1",
		"Bad Intelligent Device Or Component",
		"Transmission #1",
		faultLampWarning,
	},
}

func (l *LiveSimulator) vehicleAverageSpeedMPS(snapshot *fleetSnapshot, vehicleID string) float64 {
	geometry, ok := snapshot.geometries[vehicleID]
	if !ok || geometry.Period <= 0 {
		return 0
	}
	total := 0.0
	for idx := range geometry.Segments {
		total += geometry.Segments[idx].DistanceMeters
	}
	return total / geometry.Period.Seconds()
}

func (l *LiveSimulator) vehicleDistanceSinceEpoch(
	snapshot *fleetSnapshot,
	vehicleID string,
	at time.Time,
) float64 {
	elapsed := at.Sub(telemetryEpoch).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return elapsed * l.vehicleAverageSpeedMPS(snapshot, vehicleID)
}

func (l *LiveSimulator) vehicleOdometerMeters(
	snapshot *fleetSnapshot,
	vehicleID string,
	at time.Time,
) int64 {
	base := int64(
		floatFromAny(nestedAny(snapshot.templates[vehicleID], "obdOdometerMeters", keyValue)),
	)
	if base <= 0 {
		base = int64(180_000_000 + l.hashFraction(vehicleID, "odometer-base")*240_000_000)
	}
	return base + int64(l.vehicleDistanceSinceEpoch(snapshot, vehicleID, at))
}

func (l *LiveSimulator) vehicleEngineSeconds(
	snapshot *fleetSnapshot,
	vehicleID string,
	at time.Time,
) int64 {
	base := int64(
		floatFromAny(nestedAny(snapshot.templates[vehicleID], "obdEngineSeconds", keyValue)),
	)
	if base <= 0 {
		base = int64(5_000_000 + l.hashFraction(vehicleID, "engine-base")*2_000_000)
	}
	return base + int64(math.Max(at.Sub(telemetryEpoch).Seconds(), 0)*engineRunFraction)
}

func (l *LiveSimulator) vehicleIdleMilliseconds(vehicleID string, at time.Time) int64 {
	base := int64(900_000_000 + l.hashFraction(vehicleID, "idle-base")*400_000_000)
	return base + int64(math.Max(at.Sub(telemetryEpoch).Seconds(), 0)*idleRunFraction*1000)
}

func (l *LiveSimulator) vehicleMPG(vehicleID string) float64 {
	return defaultDieselMPGLow + defaultDieselMPGSpan*l.hashFraction(vehicleID, "mpg")
}

func (l *LiveSimulator) vehicleFuelConsumedMilliliters(
	snapshot *fleetSnapshot,
	vehicleID string,
	at time.Time,
) int64 {
	base := int64(95_000_000 + l.hashFraction(vehicleID, "fuel-base")*60_000_000)
	miles := l.vehicleDistanceSinceEpoch(snapshot, vehicleID, at) / metersPerMile
	return base + int64(miles/l.vehicleMPG(vehicleID)*litersPerGallon*1000)
}

func (l *LiveSimulator) vehicleFuelPercent(
	snapshot *fleetSnapshot,
	vehicleID string,
	at time.Time,
) int64 {
	miles := l.vehicleDistanceSinceEpoch(snapshot, vehicleID, at) / metersPerMile
	phase := l.hashFraction(vehicleID, "fuel-phase")
	cycle := math.Mod(miles/tankRangeMiles+phase, 1)
	return clampInt64(int64(math.Round(98-84*cycle)), 8, 100)
}

func (l *LiveSimulator) vehicleDefMilliPercent(
	snapshot *fleetSnapshot,
	vehicleID string,
	at time.Time,
) int64 {
	miles := l.vehicleDistanceSinceEpoch(snapshot, vehicleID, at) / metersPerMile
	phase := l.hashFraction(vehicleID, "def-phase")
	cycle := math.Mod(miles/defTankDieselRatioMiles+phase, 1)
	return clampInt64(int64(math.Round(99_000-76_000*cycle)), 0, 99_999)
}

func vehicleEngineState(ctx *statContext, at time.Time) (string, bool) {
	state, ok := ctx.position(at)
	if !ok {
		return "", false
	}
	if state.SpeedMPS > movingSpeedThresholdMPS {
		return engineStateOn, true
	}
	if state.SpeedMPS > 0 {
		return engineStateIdle, true
	}
	if ctx.hash("stopped-idle", at.UTC().Truncate(10*time.Minute).Format(time.RFC3339)) <
		stoppedIdleProbability {
		return engineStateIdle, true
	}
	return engineStateOff, true
}

func vehicleEngineRPM(ctx *statContext, at time.Time) (int64, bool) {
	engineState, ok := vehicleEngineState(ctx, at)
	if !ok {
		return 0, false
	}
	state, _ := ctx.position(at)
	jitter := ctx.hash("rpm", formatSampleTime(at))
	switch engineState {
	case engineStateOn:
		return int64(1050 + math.Min(state.SpeedMPS*14, 420) + 60*jitter), true
	case engineStateIdle:
		return int64(620 + 70*jitter), true
	default:
		return 0, true
	}
}

func (l *LiveSimulator) gatewayInstallTime(asset Record) time.Time {
	if created := recordTime(asset, fieldCreatedAtTime); !created.IsZero() {
		return created
	}
	return telemetryEpoch.Add(-gatewayInstallFallback)
}

func vehicleFaultCodes(ctx *statContext, at time.Time) map[string]any {
	day := at.UTC().Truncate(24 * time.Hour)
	dayKey := day.Format(dateLayout)
	codes := make([]any, 0, 2)
	lights := map[string]any{
		"emissionsIsOn": false,
		"protectIsOn":   false,
		"stopIsOn":      false,
		"warningIsOn":   false,
	}
	if ctx.hash("fault-day", dayKey) < faultCodeDailyRate {
		fault := j1939FaultCatalog[int(ctx.hash("fault-pick", dayKey)*float64(len(j1939FaultCatalog)))%len(j1939FaultCatalog)]
		codes = append(codes, map[string]any{
			"fmiDescription":    fault.FmiDescription,
			"fmiId":             fault.FmiID,
			"milStatus":         int64(1),
			"occurrenceCount":   int64(1 + ctx.hash("fault-count", dayKey)*6),
			"sourceAddressName": fault.Source,
			"spnDescription":    fault.SpnDescription,
			"spnId":             fault.SpnID,
			"txId":              int64(0),
		})
		lights[fault.Lamp+"IsOn"] = true
	}
	return map[string]any{
		"canBusType": defaultCanBusType,
		"j1939": map[string]any{
			"checkEngineLights":      lights,
			"diagnosticTroubleCodes": codes,
		},
	}
}

func faultCodeEventTimes(ctx *statContext, from, to time.Time) []time.Time {
	floor := ctx.view.live.gatewayInstallTime(ctx.asset)
	return dailyEventTimes(maxTime(from, floor), to, func(day time.Time) (time.Duration, bool) {
		minutes := 300 + ctx.hash("fault-time", day.Format(dateLayout))*120
		return time.Duration(minutes * float64(time.Minute)), true
	})
}

func immobilizerInstalledAt(asset Record) (time.Time, bool) {
	raw, ok := anyAsMap(asset["simImmobilizer"])
	if !ok {
		return time.Time{}, false
	}
	at, err := parseRFC3339(stringValue(Record(raw), "installedAtTime"))
	if err != nil {
		return time.Time{}, false
	}
	return at, true
}

type immobilizerEvent struct {
	At        time.Time
	Connected bool
}

func immobilizerEvents(ctx *statContext, from, to time.Time) []immobilizerEvent {
	installed, ok := immobilizerInstalledAt(ctx.asset)
	if !ok || to.Before(installed) {
		return nil
	}
	out := make([]immobilizerEvent, 0, 4)
	if !installed.Before(from) {
		out = append(out, immobilizerEvent{At: installed, Connected: true})
	}
	start := maxTime(from, installed).UTC().Truncate(24 * time.Hour)
	for day := start; !day.After(to); day = day.Add(24 * time.Hour) {
		dayKey := day.Format(dateLayout)
		if !day.After(installed) || ctx.hash("immobilizer-blip", dayKey) >= immobilizerBlipRate {
			continue
		}
		dropAt := day.Add(
			time.Duration((2 + ctx.hash("immobilizer-time", dayKey)*20) * float64(time.Hour)),
		)
		for _, event := range []immobilizerEvent{
			{At: dropAt, Connected: false},
			{At: dropAt.Add(immobilizerBlipDuration), Connected: true},
		} {
			if !event.At.Before(from) && !event.At.After(to) && !event.At.Before(installed) {
				out = append(out, event)
			}
		}
	}
	return out
}

func immobilizerEventTimes(ctx *statContext, from, to time.Time) []time.Time {
	installed, ok := immobilizerInstalledAt(ctx.asset)
	if !ok || to.Before(installed) {
		return nil
	}
	events := immobilizerEvents(ctx, from, to)
	out := make([]time.Time, 0, len(events)+1)
	if installed.Before(from) {
		out = append(out, installed)
	}
	for _, event := range events {
		out = append(out, event.At)
	}
	return out
}

func immobilizerConnectedAt(ctx *statContext, at time.Time) (bool, time.Time, bool) {
	events := immobilizerEvents(ctx, at.Add(-statEventLookback), at)
	if len(events) == 0 {
		installed, ok := immobilizerInstalledAt(ctx.asset)
		if !ok || at.Before(installed) {
			return false, time.Time{}, false
		}
		return true, installed, true
	}
	last := events[len(events)-1]
	return last.Connected, last.At, true
}

func auxInputActive(ctx *statContext, inputType string, at time.Time) bool {
	state, ok := ctx.position(at)
	if !ok {
		return false
	}
	bucket := at.UTC().Truncate(15 * time.Minute).Format(time.RFC3339)
	switch inputType {
	case auxInputPowerTakeOff, auxInputEcuPowerTakeOff:
		return state.SpeedMPS <= movingSpeedThresholdMPS && ctx.hash("aux-pto", bucket) < 0.6
	case auxInputEmergencyLights, auxInputEightWayLights:
		return ctx.hash("aux-lights", inputType, bucket) < 0.04
	case auxInputDoor:
		return state.SpeedMPS <= movingSpeedThresholdMPS && ctx.hash("aux-door", bucket) < 0.35
	case auxInputReefer, auxInputAuxiliaryEngine, auxInputGenerator:
		return ctx.hash("aux-engine", bucket) < 0.7
	default:
		return ctx.hash("aux-other", inputType, bucket) < 0.1
	}
}
