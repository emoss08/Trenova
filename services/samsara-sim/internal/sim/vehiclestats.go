package sim

import (
	"math"
	"net/http"
	"strconv"
	"time"
)

const (
	vehicleAuxInputCount    = 13
	maxVehicleStatTypes     = 3
	maxVehicleDecorations   = 2
	auxInputGroupedFirst    = 3
	auxInputGroupedLast     = 10
	auxInputGroupName       = "auxInput3-10"
	statTypeGPS             = "gps"
	statTypeEngineStates    = "engineStates"
	statTypeFuelPercents    = "fuelPercents"
	statTypeObdOdometer     = "obdOdometerMeters"
	vehicleStatsSignature   = "|vehicle-stats"
	vehicleFeedSignature    = "|vehicle-stats-feed"
	vehicleHistorySignature = "|vehicle-stats-history"
)

var auxInputLabels = map[string]string{
	auxInputNone:            rulesetNone,
	auxInputEmergencyLights: "Emergency Lights",
	"emergencyAlarm":        "Emergency Alarm",
	"stopPaddle":            "Stop Paddle",
	auxInputPowerTakeOff:    "Power Take-Off",
	"plow":                  "Plow",
	"sweeper":               "Sweeper",
	"salter":                "Salter",
	auxInputReefer:          "Reefer",
	auxInputDoor:            "Door",
	"boom":                  "Boom",
	auxInputAuxiliaryEngine: "Auxiliary Engine",
	auxInputGenerator:       "Generator",
	auxInputEightWayLights:  "Eight-Way Lights",
	"panicButton":           "Panic Button",
	"privacyButton":         "Privacy Button",
	"frontAxleDrive":        "Front Axle Drive",
	"weightSensor":          "Weight Sensor",
	"other":                 "Other",
	"secondaryFuelSource":   "Secondary Fuel Source",
	auxInputEcuPowerTakeOff: "ECU Power Take-Off",
}

var (
	electricVehicleStatTypes = []string{
		"evStateOfChargeMilliPercent", "evChargingStatus", "evChargingEnergyMicroWh",
		"evChargingVoltageMilliVolt", "evChargingCurrentMilliAmp", "evConsumedEnergyMicroWh",
		"evRegeneratedEnergyMicroWh", "evBatteryVoltageMilliVolt", "evBatteryCurrentMilliAmp",
		"evBatteryStateOfHealthMilliPercent", "evAverageBatteryTemperatureMilliCelsius",
		"evDistanceDrivenMeters",
	}
	spreaderStatTypes = []string{
		"spreaderLiquidRate", "spreaderGranularRate", "spreaderPrewetRate", "spreaderAirTemp",
		"spreaderRoadTemp", "spreaderOnState", "spreaderActive", "spreaderBlastState",
		"spreaderGranularName", "spreaderPrewetName", "spreaderLiquidName", "spreaderPlowStatus",
	}
	vehicleStatRegistry = buildVehicleStatRegistry()
)

func buildVehicleStatRegistry() *statRegistry {
	specs := make([]*statSpec, 0, 64)
	specs = append(specs, scalarStat("ambientAirTemperatureMilliC", "", true, ambientAirValue))
	for index := 1; index <= vehicleAuxInputCount; index++ {
		specs = append(specs, auxInputStat(index))
	}
	specs = append(
		specs,
		scalarStat("barometricPressurePa", "", true, barometricPressureValue),
		scalarStat("batteryMilliVolts", "", true, batteryMilliVoltsValue),
		scalarStat("defLevelMilliPercent", "", true, defLevelValue),
		withDecorationTime(scalarStat("ecuDoorStatus", "", true, ecuDoorStatusValue)),
		scalarStat("ecuSpeedMph", "", true, ecuSpeedValue),
		scalarStat("engineCoolantTemperatureMilliC", "", true, coolantTemperatureValue),
		&statSpec{
			Type:                keyEngineImmobilizer,
			SnapshotKey:         keyEngineImmobilizer,
			FeedKey:             keyEngineImmobilizer,
			Kind:                statEvent,
			Decoration:          true,
			DecorationKeepsTime: true,
			Sample:              engineImmobilizerSample,
			EventTimes:          immobilizerEventTimes,
		},
		scalarStat("engineLoadPercent", "", true, engineLoadValue),
		scalarStat("engineOilPressureKPa", "", true, oilPressureValue),
		scalarStat("engineRpm", "", true, vehicleEngineRPMValue),
		scalarStat(statTypeEngineStates, "engineState", true, vehicleEngineStateValue),
		&statSpec{
			Type:        keyFaultCodes,
			SnapshotKey: keyFaultCodes,
			FeedKey:     keyFaultCodes,
			Kind:        statEvent,
			Decoration:  true,
			Sample:      faultCodesSample,
			EventTimes:  faultCodeEventTimes,
		},
		scalarStat(statTypeFuelPercents, "fuelPercent", true, fuelPercentValue),
		scalarStat("fuelConsumedMilliliters", "", true, fuelConsumedValue),
		&statSpec{
			Type:        statTypeGPS,
			SnapshotKey: statTypeGPS,
			FeedKey:     statTypeGPS,
			Decoration:  true,
			Sample:      vehicleGPSSample,
		},
		scalarStat("gpsDistanceMeters", "", true, gpsDistanceValue),
		scalarStat("gpsOdometerMeters", "", false, manualOdometerMeters),
		scalarStat("idlingDurationMilliseconds", "", true, idlingDurationValue),
		scalarStat("intakeManifoldTemperatureMilliC", "", true, intakeManifoldValue),
		&statSpec{
			Type:        "nfcCardScans",
			SnapshotKey: "nfcCardScan",
			FeedKey:     "nfcCardScans",
			Kind:        statEvent,
			Decoration:  true,
			Sample:      nfcCardScanSample,
			EventTimes:  nfcCardScanTimes,
		},
		scalarStat("obdEngineSeconds", "", true, obdEngineSecondsValue),
		scalarStat(statTypeObdOdometer, "", true, obdOdometerValue),
		scalarStat("syntheticEngineSeconds", "", false, manualEngineSeconds),
	)
	for _, evType := range electricVehicleStatTypes {
		specs = append(specs, unsupportedStat(evType, false))
	}
	for _, spreaderType := range spreaderStatTypes {
		specs = append(specs, unsupportedStat(spreaderType, true))
	}
	specs = append(
		specs,
		withDecorationTime(scalarStat("seatbeltDriver", "", true, seatbeltValue)),
		unsupportedStat("tellTales", true),
	)
	return newStatRegistry(keyVehicle, maxVehicleStatTypes, maxVehicleDecorations, specs, nil)
}

func vehicleEngineRPMValue(ctx *statContext, at time.Time) (any, bool) {
	return anyValue(vehicleEngineRPM(ctx, at))
}

func vehicleEngineStateValue(ctx *statContext, at time.Time) (any, bool) {
	return anyValue(vehicleEngineState(ctx, at))
}

func scalarStat(
	statType string,
	snapshotKey string,
	decoration bool,
	value func(ctx *statContext, at time.Time) (any, bool),
) *statSpec {
	if snapshotKey == "" {
		snapshotKey = statType
	}
	return &statSpec{
		Type:        statType,
		SnapshotKey: snapshotKey,
		FeedKey:     statType,
		Decoration:  decoration,
		Sample: func(ctx *statContext, at time.Time) map[string]any {
			result, ok := value(ctx, at)
			if !ok {
				return nil
			}
			return map[string]any{keyValue: result}
		},
	}
}

func withDecorationTime(spec *statSpec) *statSpec {
	spec.DecorationKeepsTime = true
	return spec
}

func unsupportedStat(statType string, decoration bool) *statSpec {
	return &statSpec{
		Type:                statType,
		SnapshotKey:         statType,
		FeedKey:             statType,
		Decoration:          decoration,
		DecorationKeepsTime: true,
		Sample: func(*statContext, time.Time) map[string]any {
			return nil
		},
	}
}

func auxInputStat(index int) *statSpec {
	statType := "auxInput" + strconv.Itoa(index)
	group := statType
	if index >= auxInputGroupedFirst && index <= auxInputGroupedLast {
		group = auxInputGroupName
	}
	return &statSpec{
		Type:        statType,
		SnapshotKey: statType,
		FeedKey:     statType,
		Group:       group,
		Decoration:  true,
		Sample: func(ctx *statContext, at time.Time) map[string]any {
			inputType := stringValue(ctx.asset, "auxInputType"+strconv.Itoa(index))
			if inputType == "" || inputType == auxInputNone {
				return nil
			}
			if _, ok := ctx.position(at); !ok {
				return nil
			}
			return map[string]any{
				keyName:  auxInputLabels[inputType],
				keyValue: auxInputActive(ctx, inputType, at),
			}
		},
	}
}

func vehicleGPSSample(ctx *statContext, at time.Time) map[string]any {
	state, ok := ctx.position(at)
	if !ok {
		return nil
	}
	engineState, _ := vehicleEngineState(ctx, at)
	sample := map[string]any{
		keyLatitude:          round(state.Latitude, 6),
		keyLongitude:         round(state.Longitude, 6),
		keyHeadingDegrees:    round(state.Heading, 1),
		keySpeedMilesPerHour: speedMilesPerHour(state),
		"isEcuSpeed":         engineState != engineStateOff,
	}
	if formatted := ctx.view.snap.reverseGeocode(state); formatted != "" {
		sample[keyReverseGeo] = map[string]any{keyFormattedLocation: formatted}
	}
	if address := ctx.view.snap.addressRef(state); address != nil {
		sample["address"] = address
	}
	return sample
}

func manualOdometerMeters(ctx *statContext, at time.Time) (any, bool) {
	reading, ok := anyAsMap(ctx.asset["simOdometerReading"])
	if !ok {
		return nil, false
	}
	readAt, err := parseRFC3339(stringValue(Record(reading), fieldTime))
	if err != nil || at.Before(readAt) {
		return nil, false
	}
	if _, fix := ctx.position(at); !fix {
		return nil, false
	}
	meters := floatFromAny(reading["meters"])
	hostID := ctx.track.HostID
	if hostID == "" {
		return int64(meters), true
	}
	live := ctx.view.live
	traveled := live.vehicleDistanceSinceEpoch(ctx.view.snap, hostID, at) -
		live.vehicleDistanceSinceEpoch(ctx.view.snap, hostID, readAt)
	return int64(meters + math.Max(traveled, 0)), true
}

func manualEngineSeconds(ctx *statContext, at time.Time) (any, bool) {
	reading, ok := anyAsMap(ctx.asset["simEngineHoursReading"])
	if !ok {
		return nil, false
	}
	readAt, err := parseRFC3339(stringValue(Record(reading), fieldTime))
	if err != nil || at.Before(readAt) {
		return nil, false
	}
	hours := floatFromAny(reading["hours"])
	return int64(hours*3600 + at.Sub(readAt).Seconds()*engineRunFraction), true
}

func nfcCardScanTimes(ctx *statContext, from, to time.Time) []time.Time {
	driverID := ctx.view.snap.cardHolderForVehicle(ctx.assetID)
	if driverID == "" {
		return nil
	}
	intervals := ctx.view.live.derivedDriverAppAssignments(
		ctx.view.snap,
		driverID,
		from,
		to,
		ctx.view.now,
	)
	out := make([]time.Time, 0, len(intervals))
	for idx := range intervals {
		start := intervals[idx].Start
		if intervals[idx].VehicleID == ctx.assetID && !start.Before(from) && !start.After(to) {
			out = append(out, start)
		}
	}
	return out
}

func nfcCardScanSample(ctx *statContext, at time.Time) map[string]any {
	driverID := ctx.view.snap.cardHolderForVehicle(ctx.assetID)
	if driverID == "" {
		return nil
	}
	code := stringValue(ctx.view.snap.driverByID[driverID], "currentIdCardCode")
	return map[string]any{"card": map[string]any{keyID: code}}
}

func (f *fleetSnapshot) cardHolderForVehicle(vehicleID string) string {
	for driverID, entry := range f.baseRoster {
		if entry.VehicleID != vehicleID {
			continue
		}
		if driver, ok := f.driverByID[driverID]; ok &&
			stringValue(driver, "currentIdCardCode") != "" && !driverDeactivated(driver) {
			return driverID
		}
	}
	return ""
}

func (s *Server) registerVehicleStatsRoutes() {
	s.mux.HandleFunc("GET /fleet/vehicles/stats", s.handleVehicleStats)
	s.mux.HandleFunc("GET /fleet/vehicles/stats/feed", s.handleVehicleStatsFeed)
	s.mux.HandleFunc("GET /fleet/vehicles/stats/history", s.handleVehicleStatsHistory)
}

type statsRequest struct {
	Types       []*statSpec
	Decorations []*statSpec
	AssetIDs    map[string]struct{}
}

func (s *Server) parseVehicleStatsRequest(
	request *http.Request,
	withDecorations bool,
) (*fleetView, *statsRequest, error) {
	values := request.URL.Query()
	types, err := vehicleStatRegistry.parseTypes(statTypesFromQuery(values, "types"))
	if err != nil {
		return nil, nil, err
	}
	parsed := &statsRequest{Types: types}
	if withDecorations {
		parsed.Decorations, err = vehicleStatRegistry.parseDecorations(
			statTypesFromQuery(values, "decorations"),
		)
		if err != nil {
			return nil, nil, err
		}
	}
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	parsed.AssetIDs = view.selectVehicles(
		csvQueryValues(values, "vehicleIds"),
		tagIDs,
		parentTagIDs,
		false,
	)
	return view, parsed, nil
}

func (v *fleetView) statRowBase(asset Record, auto autoExternalIDs) Record {
	row := Record{keyID: recordID(asset), keyName: stringValue(asset, keyName)}
	if externalIDs := renderExternalIDs(asset, auto); len(externalIDs) > 0 {
		row[fieldExternalIDs] = externalIDs
	}
	return row
}

func (s *Server) handleVehicleStats(writer http.ResponseWriter, request *http.Request) {
	view, parsed, err := s.parseVehicleStatsRequest(request, false)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	at, err := snapshotTime(request, view.now)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.dispatchLiveEvents(request, view.now, setKeys(parsed.AssetIDs))
	records := make([]Record, 0, len(parsed.AssetIDs))
	for _, vehicle := range view.vehicleRecords() {
		if _, ok := parsed.AssetIDs[recordID(vehicle)]; !ok {
			continue
		}
		row := view.statRowBase(vehicle, vehicleAutoExternalIDs)
		ctx := view.newStatContext(vehicle, at, at)
		snapshotStats(ctx, parsed.Types, at, row)
		records = append(records, row)
	}
	s.respondPage(writer, request, records, vehicleStatsSignature)
}

func snapshotTime(request *http.Request, now time.Time) (time.Time, error) {
	at, present, err := parseOptionalTime(request.URL.Query(), fieldTime)
	if err != nil {
		return time.Time{}, err
	}
	if !present || at.After(now) {
		return now.Truncate(time.Second), nil
	}
	return at.Truncate(time.Second), nil
}

func setKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	return out
}

func (s *Server) handleVehicleStatsFeed(writer http.ResponseWriter, request *http.Request) {
	view, parsed, err := s.parseVehicleStatsRequest(request, true)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	window, after, err := parseFeedWindow(
		request,
		view.now,
		len(parsed.AssetIDs),
		pagePolicyFor(request).Size,
	)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.dispatchLiveEvents(request, view.now, setKeys(parsed.AssetIDs))
	if len(window.Times) == 0 {
		s.respondFeed(writer, request, []Record{}, after, false, vehicleFeedSignature)
		return
	}
	records := make([]Record, 0, len(parsed.AssetIDs))
	for _, vehicle := range view.vehicleRecords() {
		if _, ok := parsed.AssetIDs[recordID(vehicle)]; !ok {
			continue
		}
		row := view.statRowBase(vehicle, vehicleAutoExternalIDs)
		ctx := view.newStatContext(vehicle, window.Times[0], window.Times[len(window.Times)-1])
		feedStats(ctx, parsed.Types, parsed.Decorations, &window.feedWindow, row)
		records = append(records, row)
	}
	s.respondFeed(writer, request, records, window.EndCursor, window.HasNext, vehicleFeedSignature)
}

func (s *Server) handleVehicleStatsHistory(writer http.ResponseWriter, request *http.Request) {
	view, parsed, err := s.parseVehicleStatsRequest(request, true)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	window, err := parseHistoryWindow(request, view.now)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	records := make([]Record, 0, len(parsed.AssetIDs))
	for _, vehicle := range view.vehicleRecords() {
		if _, ok := parsed.AssetIDs[recordID(vehicle)]; !ok {
			continue
		}
		row := view.statRowBase(vehicle, vehicleAutoExternalIDs)
		ctx := view.newStatContext(vehicle, window.EventFrom, window.EventTo)
		feedStats(ctx, parsed.Types, parsed.Decorations, window, row)
		records = append(records, row)
	}
	s.respondPage(writer, request, records, vehicleHistorySignature)
}
