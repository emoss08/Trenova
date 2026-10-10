package sim

import (
	"math"
	"time"
)

func ambientAirValue(ctx *statContext, at time.Time) (any, bool) {
	state, ok := ctx.position(at)
	if !ok {
		return nil, false
	}
	return ambientTemperatureMilliC(state.Latitude, at, ctx.assetID), true
}

func barometricPressureValue(ctx *statContext, at time.Time) (any, bool) {
	if _, ok := ctx.position(at); !ok {
		return nil, false
	}
	swing := 900 * math.Sin(float64(at.Unix())/43200+ctx.hash("baro")*math.Pi*2)
	return int64(100_700 + swing + 300*ctx.hash("baro-offset")), true
}

func batteryMilliVoltsValue(ctx *statContext, at time.Time) (any, bool) {
	engineState, ok := vehicleEngineState(ctx, at)
	if !ok {
		return nil, false
	}
	jitter := 80 * ctx.hash("battery", formatSampleTime(at))
	if engineState == engineStateOff {
		return int64(12_480 + jitter), true
	}
	return int64(13_850 + jitter + 200*ctx.hash("alternator")), true
}

func defLevelValue(ctx *statContext, at time.Time) (any, bool) {
	if _, ok := ctx.position(at); !ok {
		return nil, false
	}
	return ctx.view.live.vehicleDefMilliPercent(ctx.view.snap, ctx.assetID, at), true
}

func ecuDoorStatusValue(ctx *statContext, at time.Time) (any, bool) {
	state, ok := ctx.position(at)
	if !ok {
		return nil, false
	}
	bucket := at.UTC().Truncate(10 * time.Minute).Format(time.RFC3339)
	if state.SpeedMPS <= movingSpeedThresholdMPS && ctx.hash(auxInputDoor, bucket) < 0.2 {
		return "Open", true
	}
	return "Closed", true
}

func ecuSpeedValue(ctx *statContext, at time.Time) (any, bool) {
	state, ok := ctx.position(at)
	if !ok {
		return nil, false
	}
	return round(state.SpeedMPS*2.23694*0.985, 2), true
}

func coolantTemperatureValue(ctx *statContext, at time.Time) (any, bool) {
	engineState, ok := vehicleEngineState(ctx, at)
	if !ok {
		return nil, false
	}
	state, _ := ctx.position(at)
	if engineState == engineStateOff {
		return ambientTemperatureMilliC(state.Latitude, at, ctx.assetID) + 14_000, true
	}
	return int64(84_000 + 7_000*ctx.hash("coolant", formatSampleTime(at))), true
}

func engineImmobilizerSample(ctx *statContext, at time.Time) map[string]any {
	connected, _, ok := immobilizerConnectedAt(ctx, at)
	if !ok {
		return nil
	}
	return map[string]any{"connected": connected, keyState: ignitionEnabled}
}

func engineLoadValue(ctx *statContext, at time.Time) (any, bool) {
	engineState, ok := vehicleEngineState(ctx, at)
	if !ok {
		return nil, false
	}
	state, _ := ctx.position(at)
	switch engineState {
	case engineStateOn:
		load := 28 + state.SpeedMPS*1.6 + 12*ctx.hash("load", formatSampleTime(at))
		return int64(math.Min(load, 96)), true
	case engineStateIdle:
		return int64(8 + 4*ctx.hash("load-idle", formatSampleTime(at))), true
	default:
		return int64(0), true
	}
}

func oilPressureValue(ctx *statContext, at time.Time) (any, bool) {
	engineState, ok := vehicleEngineState(ctx, at)
	if !ok {
		return nil, false
	}
	switch engineState {
	case engineStateOn:
		return int64(310 + 60*ctx.hash("oil", formatSampleTime(at))), true
	case engineStateIdle:
		return int64(150 + 30*ctx.hash("oil-idle", formatSampleTime(at))), true
	default:
		return int64(0), true
	}
}

func faultCodesSample(ctx *statContext, at time.Time) map[string]any {
	if _, ok := ctx.position(at); !ok {
		return nil
	}
	return vehicleFaultCodes(ctx, at)
}

func fuelPercentValue(ctx *statContext, at time.Time) (any, bool) {
	if _, ok := ctx.position(at); !ok {
		return nil, false
	}
	return ctx.view.live.vehicleFuelPercent(ctx.view.snap, ctx.assetID, at), true
}

func fuelConsumedValue(ctx *statContext, at time.Time) (any, bool) {
	if _, ok := ctx.position(at); !ok {
		return nil, false
	}
	return ctx.view.live.vehicleFuelConsumedMilliliters(ctx.view.snap, ctx.assetID, at), true
}

func gpsDistanceValue(ctx *statContext, at time.Time) (any, bool) {
	if _, ok := ctx.position(at); !ok {
		return nil, false
	}
	installed := ctx.view.live.gatewayInstallTime(ctx.asset)
	if at.Before(installed) {
		return nil, false
	}
	live := ctx.view.live
	meters := live.vehicleDistanceSinceEpoch(ctx.view.snap, ctx.assetID, at) -
		live.vehicleDistanceSinceEpoch(ctx.view.snap, ctx.assetID, installed)
	return round(math.Max(meters, 0), 3), true
}

func idlingDurationValue(ctx *statContext, at time.Time) (any, bool) {
	if _, ok := ctx.position(at); !ok {
		return nil, false
	}
	return ctx.view.live.vehicleIdleMilliseconds(ctx.assetID, at), true
}

func intakeManifoldValue(ctx *statContext, at time.Time) (any, bool) {
	engineState, ok := vehicleEngineState(ctx, at)
	if !ok {
		return nil, false
	}
	state, _ := ctx.position(at)
	ambient := ambientTemperatureMilliC(state.Latitude, at, ctx.assetID)
	if engineState == engineStateOff {
		return ambient + 3_000, true
	}
	return ambient + int64(16_000+9_000*ctx.hash("intake", formatSampleTime(at))), true
}

func obdEngineSecondsValue(ctx *statContext, at time.Time) (any, bool) {
	if _, ok := ctx.position(at); !ok {
		return nil, false
	}
	return ctx.view.live.vehicleEngineSeconds(ctx.view.snap, ctx.assetID, at), true
}

func obdOdometerValue(ctx *statContext, at time.Time) (any, bool) {
	if _, ok := ctx.position(at); !ok {
		return nil, false
	}
	return ctx.view.live.vehicleOdometerMeters(ctx.view.snap, ctx.assetID, at), true
}

func seatbeltValue(ctx *statContext, at time.Time) (any, bool) {
	state, ok := ctx.position(at)
	if !ok {
		return nil, false
	}
	if state.SpeedMPS > movingSpeedThresholdMPS {
		return "Buckled", true
	}
	return "Unbuckled", true
}
