package sim

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxTrailerStatTypes    = 3
	maxTrailerDecorations  = 2
	maxTrailerLicensePlate = 12
	maxTrailerNotes        = 255
	fieldInstalledGateway  = "installedGateway"
	fieldSimReefer         = "simReefer"
	reeferStateCooling     = "Cooling"
	reeferStateDefrost     = "Defrost"
	reeferStateHeating     = "Heating"
	reeferRunContinuous    = "Continuous"
	reeferRunStartStop     = "Start/Stop"
	reeferTankHours        = 96.0
)

type reeferProfile struct {
	Zones       int
	SetPointM   int64
	Carrier     bool
	RunMode     string
	UnitModel   string
	Present     bool
	RunFraction float64
}

var trailerStatRegistry = buildTrailerStatRegistry()

func buildTrailerStatRegistry() *statRegistry {
	specs := []*statSpec{
		{
			Type:        statTypeGPS,
			SnapshotKey: statTypeGPS,
			FeedKey:     statTypeGPS,
			Decoration:  true,
			Sample:      trailerGPSSample,
		},
		trailerScalar("gpsOdometerMeters", manualOdometerMeters),
		trailerScalar(
			"reeferAmbientAirTemperatureMilliC",
			func(ctx *statContext, at time.Time) (any, bool) {
				if !reeferOf(ctx.asset).Present {
					return nil, false
				}
				state, ok := ctx.position(at)
				if !ok {
					return nil, false
				}
				return ambientTemperatureMilliC(state.Latitude, at, ctx.assetID), true
			},
		),
		trailerScalar("reeferObdEngineSeconds", func(ctx *statContext, at time.Time) (any, bool) {
			return anyValue(reeferEngineSeconds(ctx, at))
		}),
		trailerScalar("reeferFuelPercent", func(ctx *statContext, at time.Time) (any, bool) {
			return anyValue(reeferFuelPercent(ctx, at))
		}),
		trailerState("carrierReeferState", 0, true),
		trailerScalar("reeferRunMode", func(ctx *statContext, at time.Time) (any, bool) {
			profile := reeferOf(ctx.asset)
			if !profile.Present {
				return nil, false
			}
			return profile.RunMode, true
		}),
		{
			Type:        keyReeferAlarms,
			SnapshotKey: keyReeferAlarms,
			FeedKey:     keyReeferAlarms,
			Decoration:  true,
			Sample:      reeferAlarmsSample,
		},
	}
	for zone := 1; zone <= 3; zone++ {
		specs = append(
			specs,
			trailerZoneTemperature("reeferSupplyAirTemperatureMilliCZone", zone, -3200),
			trailerState("reeferStateZone"+strconv.Itoa(zone), zone, false),
			trailerZoneTemperature("reeferReturnAirTemperatureMilliCZone", zone, 2100),
			trailerZoneTemperature("reeferSetPointTemperatureMilliCZone", zone, 0),
			trailerScalar(
				"reeferDoorStateZone"+strconv.Itoa(zone),
				func(ctx *statContext, at time.Time) (any, bool) {
					return reeferDoorState(ctx, zone, at)
				},
			),
		)
	}
	return newStatRegistry("trailer", maxTrailerStatTypes, maxTrailerDecorations, specs, nil)
}

func trailerScalar(
	statType string,
	value func(ctx *statContext, at time.Time) (any, bool),
) *statSpec {
	return scalarStat(statType, "", true, value)
}

func trailerZoneTemperature(prefix string, zone int, offsetMilliC int64) *statSpec {
	statType := prefix + strconv.Itoa(zone)
	return trailerScalar(statType, func(ctx *statContext, at time.Time) (any, bool) {
		profile := reeferOf(ctx.asset)
		if !profile.Present || zone > profile.Zones {
			return nil, false
		}
		if _, ok := ctx.position(at); !ok {
			return nil, false
		}
		setPoint := profile.zoneSetPoint(zone)
		if offsetMilliC == 0 {
			return setPoint, true
		}
		jitter := int64(
			(ctx.hash(statType, at.UTC().Truncate(5*time.Minute).Format(time.RFC3339)) - 0.5) * 900,
		)
		if reeferStateAt(ctx, profile, zone, at) == reeferStateDefrost {
			return setPoint + 6_500 + jitter, true
		}
		return setPoint + offsetMilliC + jitter, true
	})
}

func trailerState(statType string, zone int, carrierOnly bool) *statSpec {
	return &statSpec{
		Type:        statType,
		SnapshotKey: statType,
		FeedKey:     statType,
		Decoration:  true,
		Sample: func(ctx *statContext, at time.Time) map[string]any {
			profile := reeferOf(ctx.asset)
			if !profile.Present {
				return nil
			}
			if _, ok := ctx.position(at); !ok {
				return nil
			}
			if carrierOnly {
				if !profile.Carrier {
					return nil
				}
				state := reeferStateAt(ctx, profile, 1, at)
				return map[string]any{keyValue: "On", "substateValue": state}
			}
			if zone > profile.Zones {
				return nil
			}
			state := reeferStateAt(ctx, profile, zone, at)
			sample := map[string]any{keyValue: state}
			if profile.RunMode == reeferRunStartStop && state == reeferStateCooling {
				sample["substateValue"] = "Start/Stop Cycle"
			}
			return sample
		},
	}
}

func reeferOf(asset Record) reeferProfile {
	raw, ok := anyAsMap(asset[fieldSimReefer])
	if !ok {
		return reeferProfile{}
	}
	record := Record(raw)
	zones, _ := int64Value(record["zones"])
	setPoint, _ := int64Value(record["setPointMilliC"])
	profile := reeferProfile{
		Zones:     int(max(zones, 1)),
		SetPointM: setPoint,
		Carrier:   record["carrier"] == true,
		RunMode: stringutils.FirstNonEmptyTrimmed(
			stringValue(record, "runMode"),
			reeferRunContinuous,
		),
		UnitModel: stringValue(record, "unitModel"),
		Present:   true,
	}
	profile.RunFraction = reeferContinuousRunShare
	if profile.RunMode == reeferRunStartStop {
		profile.RunFraction = reeferStartStopRunShare
	}
	return profile
}

func (p reeferProfile) zoneSetPoint(zone int) int64 {
	if zone <= 1 {
		return p.SetPointM
	}
	return 2_000 + int64(zone-2)*2_000
}

func reeferStateAt(ctx *statContext, profile reeferProfile, zone int, at time.Time) string {
	offset := time.Duration(
		ctx.hash("defrost-phase", strconv.Itoa(zone)) * float64(reeferDefrostInterval),
	)
	phase := at.Add(offset).Sub(telemetryEpoch) % reeferDefrostInterval
	if phase >= 0 && phase < reeferDefrostDuration {
		return reeferStateDefrost
	}
	state, ok := ctx.position(at)
	if ok &&
		profile.zoneSetPoint(zone) > ambientTemperatureMilliC(state.Latitude, at, ctx.assetID) {
		return reeferStateHeating
	}
	return reeferStateCooling
}

func reeferEngineSeconds(ctx *statContext, at time.Time) (int64, bool) {
	profile := reeferOf(ctx.asset)
	if !profile.Present {
		return 0, false
	}
	if _, ok := ctx.position(at); !ok {
		return 0, false
	}
	base := 8_000_000 + ctx.hash("reefer-hours")*9_000_000
	return int64(base + math.Max(at.Sub(telemetryEpoch).Seconds(), 0)*profile.RunFraction), true
}

func reeferFuelPercent(ctx *statContext, at time.Time) (int64, bool) {
	profile := reeferOf(ctx.asset)
	if !profile.Present {
		return 0, false
	}
	if _, ok := ctx.position(at); !ok {
		return 0, false
	}
	runHours := math.Max(at.Sub(telemetryEpoch).Hours(), 0) * profile.RunFraction
	cycle := math.Mod(runHours/reeferTankHours+ctx.hash("reefer-fuel-phase"), 1)
	return clampInt64(int64(math.Round(97-82*cycle)), 6, 100), true
}

func reeferDoorState(ctx *statContext, zone int, at time.Time) (any, bool) {
	profile := reeferOf(ctx.asset)
	if !profile.Present || zone > profile.Zones {
		return nil, false
	}
	state, ok := ctx.position(at)
	if !ok {
		return nil, false
	}
	bucket := at.UTC().Truncate(10 * time.Minute).Format(time.RFC3339)
	if state.SpeedMPS <= movingSpeedThresholdMPS &&
		ctx.hash("reefer-door", strconv.Itoa(zone), bucket) < 0.25 {
		return "open", true
	}
	return "closed", true
}

func reeferAlarmsSample(ctx *statContext, at time.Time) map[string]any {
	profile := reeferOf(ctx.asset)
	if !profile.Present {
		return nil
	}
	if _, ok := ctx.position(at); !ok {
		return nil
	}
	alarms := make([]any, 0, 2)
	if fuel, ok := reeferFuelPercent(ctx, at); ok && fuel < 20 {
		alarms = append(alarms, map[string]any{
			"alarmCode":      "00001",
			keyDescription:   "Low Fuel Level",
			"operatorAction": "Refuel the unit at the next opportunity",
			keySeverity:      int64(2),
		})
	}
	day := at.UTC().Truncate(24 * time.Hour).Format(dateLayout)
	if ctx.hash("reefer-alarm", day) < 0.12 {
		alarms = append(alarms, map[string]any{
			"alarmCode":      "00084",
			keyDescription:   "Restart Null",
			"operatorAction": "Check unit; it restarted after a null cycle",
			keySeverity:      int64(1),
		})
	}
	return map[string]any{"alarms": alarms}
}

func trailerGPSSample(ctx *statContext, at time.Time) map[string]any {
	state, ok := ctx.position(at)
	if !ok {
		return nil
	}
	sample := map[string]any{
		keyLatitude:          round(state.Latitude, 6),
		keyLongitude:         round(state.Longitude, 6),
		keyHeadingDegrees:    int64(math.Round(state.Heading)),
		keySpeedMilesPerHour: int64(math.Round(state.SpeedMPS * 2.23694)),
	}
	if formatted := ctx.view.snap.reverseGeocode(state); formatted != "" {
		sample[keyReverseGeo] = map[string]any{keyFormattedLocation: formatted}
	}
	return sample
}

func (s *Server) registerTrailerRoutes() {
	s.mux.HandleFunc("GET /fleet/trailers", s.handleTrailerList)
	s.mux.HandleFunc("POST /fleet/trailers", s.handleTrailerCreate)
	s.mux.HandleFunc("GET /fleet/trailers/stats", s.handleTrailerStats)
	s.mux.HandleFunc("GET /fleet/trailers/stats/feed", s.handleTrailerStatsFeed)
	s.mux.HandleFunc("GET /fleet/trailers/stats/history", s.handleTrailerStatsHistory)
	s.mux.HandleFunc("GET /fleet/trailers/{id}", s.handleTrailerGet)
	s.mux.HandleFunc("PATCH /fleet/trailers/{id}", s.handleTrailerPatch)
	s.mux.HandleFunc("DELETE /fleet/trailers/{id}", s.handleTrailerDelete)
}

func (v *fleetView) trailerView(trailer Record, single bool) Record {
	id := recordID(trailer)
	out := Record{
		keyID:               id,
		keyEnabledForMobile: trailer[keyEnabledForMobile] == true,
		fieldExternalIDs:    renderExternalIDs(trailer, nil),
		keyTags:             v.snap.tags.tinyTags(tagMembersAssets, id),
	}
	for field, viewField := range map[string]string{
		keyName:           keyName,
		keyLicensePlate:   keyLicensePlate,
		keyNotes:          keyNotes,
		fieldSerialNumber: keyTrailerSerialNumber,
	} {
		if value := stringValue(trailer, field); value != "" {
			out[viewField] = value
		}
	}
	if gateway, ok := anyAsMap(trailer[fieldInstalledGateway]); ok {
		out[fieldInstalledGateway] = map[string]any{
			keyModel:  stringValue(Record(gateway), keyModel),
			keySerial: stringValue(Record(gateway), keySerial),
		}
	}
	if single {
		out[fieldAttributesKey] = renderAttributes(trailer)
	}
	return out
}

func (s *Server) handleTrailerList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	filter := view.snap.tags.filter(tagIDs, parentTagIDs)
	trailers := view.assetsOfType(assetTypeTrailer)
	records := make([]Record, 0, len(trailers))
	for _, trailer := range trailers {
		if filter.matches(view.snap.tags, tagMembersAssets, recordID(trailer)) {
			records = append(records, view.trailerView(trailer, false))
		}
	}
	s.respondPage(writer, request, records, "|trailer-list")
}

func (s *Server) handleTrailerGet(writer http.ResponseWriter, request *http.Request) {
	ref, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	trailer, idx := findRecordByRef(
		view.assetsOfType(assetTypeTrailer),
		ref,
		vehicleAutoExternalIDs,
	)
	if idx < 0 {
		s.writeError(writer, notFound("trailer", ref))
		return
	}
	payload := map[string]any{keyData: view.trailerView(trailer, true)}
	s.respondJSON(writer, request, requestSignature(request)+"|trailer-get", payload)
}

func trailerBodyRules(mode bodyMode) []fieldRule {
	rules := []fieldRule{
		{Name: fieldAttributesKey, Kind: kindAttributes},
		{Name: keyEnabledForMobile, Kind: kindBool},
		{Name: fieldExternalIDs, Kind: kindExternalIDs, Nullable: true},
		{Name: keyLicensePlate, Kind: kindString, MaxLen: maxTrailerLicensePlate, Nullable: true},
		{
			Name:     keyName,
			Kind:     kindString,
			Required: mode == bodyCreate,
			MinLen:   1,
			MaxLen:   255,
			Check:    nonBlank(keyName),
		},
		{Name: keyNotes, Kind: kindString, MaxLen: maxTrailerNotes, Nullable: true},
		{Name: fieldTagIDs, Kind: kindStringList},
		{Name: keyTrailerSerialNumber, Kind: kindString, MaxLen: 255, Nullable: true},
	}
	if mode == bodyPatch {
		rules = append(rules, fieldRule{
			Name:     keyOdometerMeters,
			Kind:     kindInt,
			IntRange: &intRange{Min: 0, Max: 10_000_000_000},
		})
	}
	return rules
}

func applyTrailerWrite(
	tx *storeTx,
	record Record,
	selfID string,
	body Record,
	now time.Time,
) error {
	if raw, ok := body[fieldExternalIDs]; ok && raw != nil {
		if err := tx.ensureExternalIDs(externalIDClaim{
			Kind:  externalIDKindTrailers,
			Owner: externalIDOwner(ResourceAssets, selfID),
			IDs:   mapOf(raw),
		}); err != nil {
			return err
		}
	}
	for field, stored := range map[string]string{
		keyEnabledForMobile:    keyEnabledForMobile,
		fieldExternalIDs:       fieldExternalIDs,
		keyLicensePlate:        keyLicensePlate,
		keyName:                keyName,
		keyNotes:               keyNotes,
		keyTrailerSerialNumber: fieldSerialNumber,
	} {
		raw, ok := body[field]
		if !ok {
			continue
		}
		if raw == nil {
			delete(record, stored)
			continue
		}
		record[stored] = raw
	}
	if raw, ok := body[fieldAttributesKey]; ok {
		record[fieldAttributesKey] = resolveAttributeIDs(listOf(raw), attributeEntityAsset)
	}
	if meters, ok := body[keyOdometerMeters].(int64); ok {
		record["simOdometerReading"] = map[string]any{
			"meters":  meters,
			fieldTime: now.UTC().Format(time.RFC3339),
		}
	}
	record[keyType] = assetTypeTrailer
	return nil
}

func (s *Server) handleTrailerCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, trailerBodyRules(bodyCreate), bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	trailerID, err := s.store.TransactID(func(tx *storeTx) (string, error) {
		return createTrailerTx(tx, sanitized, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondTrailer(writer, request, trailerID, "|trailer-create")
}

func (s *Server) handleTrailerPatch(writer http.ResponseWriter, request *http.Request) {
	ref, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, trailerBodyRules(bodyPatch), bodyPatch)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	trailerID, err := s.store.TransactID(func(tx *storeTx) (string, error) {
		return patchTrailerTx(tx, ref, sanitized, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondTrailer(writer, request, trailerID, "|trailer-patch")
}

func createTrailerTx(tx *storeTx, body Record, now time.Time) (string, error) {
	record := Record{keyEnabledForMobile: false}
	if err := applyTrailerWrite(tx, record, "", body, now); err != nil {
		return "", err
	}
	created, err := tx.insert(ResourceAssets, record, now)
	if err != nil {
		return "", err
	}
	trailerID := recordID(created)
	tagIDs := stringListValues(body[fieldTagIDs])
	return trailerID, setEntityTagsTx(tx, tagMembersAssets, trailerID, tagIDs)
}

func patchTrailerTx(tx *storeTx, ref string, body Record, now time.Time) (string, error) {
	assets := tx.records(ResourceAssets)
	trailers := make([]Record, 0, len(assets))
	for _, asset := range assets {
		if assetType(asset) == assetTypeTrailer {
			trailers = append(trailers, asset)
		}
	}
	current, idx := findRecordByRef(trailers, ref, vehicleAutoExternalIDs)
	if idx < 0 {
		return "", notFound("trailer", ref)
	}
	trailerID := recordID(current)
	_, err := tx.update(ResourceAssets, trailerID, func(record Record) error {
		return applyTrailerWrite(tx, record, trailerID, body, now)
	}, now)
	if err != nil {
		return "", err
	}
	if raw, ok := body[fieldTagIDs]; ok {
		return trailerID, setEntityTagsTx(tx, tagMembersAssets, trailerID, stringListValues(raw))
	}
	return trailerID, nil
}

func (s *Server) respondTrailer(
	writer http.ResponseWriter,
	request *http.Request,
	trailerID string,
	signature string,
) {
	view := s.fleetView()
	trailer, ok := view.snap.assetByID[trailerID]
	if !ok {
		s.writeError(writer, notFound("trailer", trailerID))
		return
	}
	payload := map[string]any{keyData: view.trailerView(trailer, true)}
	s.respondJSON(writer, request, requestSignature(request)+signature, payload)
}

func (s *Server) handleTrailerDelete(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if err = s.deleteAsset(id, assetTypeTrailer); err != nil {
		s.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) parseTrailerStatsRequest(
	request *http.Request,
	withDecorations bool,
) (*fleetView, *statsRequest, error) {
	values := request.URL.Query()
	types, err := trailerStatRegistry.parseTypes(statTypesFromQuery(values, "types"))
	if err != nil {
		return nil, nil, err
	}
	parsed := &statsRequest{Types: types}
	if withDecorations {
		parsed.Decorations, err = trailerStatRegistry.parseDecorations(
			statTypesFromQuery(values, "decorations"),
		)
		if err != nil {
			return nil, nil, err
		}
	}
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	filter := view.snap.tags.filter(tagIDs, parentTagIDs)
	trailers := view.assetsOfType(assetTypeTrailer)
	var refs map[string]struct{}
	if raw := csvQueryValues(values, "trailerIds"); len(raw) > 0 {
		refs = resolveRecordRefs(trailers, raw, vehicleAutoExternalIDs)
	}
	parsed.AssetIDs = make(map[string]struct{}, len(trailers))
	for _, trailer := range trailers {
		id := recordID(trailer)
		if inOptionalSet(refs, id) && filter.matches(view.snap.tags, tagMembersAssets, id) {
			parsed.AssetIDs[id] = struct{}{}
		}
	}
	return view, parsed, nil
}

func (s *Server) handleTrailerStats(writer http.ResponseWriter, request *http.Request) {
	view, parsed, err := s.parseTrailerStatsRequest(request, false)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	at, err := snapshotTime(request, view.now)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	records := make([]Record, 0, len(parsed.AssetIDs))
	for _, trailer := range view.assetsOfType(assetTypeTrailer) {
		if _, ok := parsed.AssetIDs[recordID(trailer)]; !ok {
			continue
		}
		row := view.statRowBase(trailer, nil)
		snapshotStats(view.newStatContext(trailer, at, at), parsed.Types, at, row)
		records = append(records, row)
	}
	s.respondPage(writer, request, records, "|trailer-stats")
}

func (s *Server) handleTrailerStatsFeed(writer http.ResponseWriter, request *http.Request) {
	view, parsed, err := s.parseTrailerStatsRequest(request, true)
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
	if len(window.Times) == 0 {
		s.respondFeed(writer, request, []Record{}, after, false, "|trailer-stats-feed")
		return
	}
	records := make([]Record, 0, len(parsed.AssetIDs))
	for _, trailer := range view.assetsOfType(assetTypeTrailer) {
		if _, ok := parsed.AssetIDs[recordID(trailer)]; !ok {
			continue
		}
		row := view.statRowBase(trailer, nil)
		ctx := view.newStatContext(trailer, window.Times[0], window.Times[len(window.Times)-1])
		feedStats(ctx, parsed.Types, parsed.Decorations, &window.feedWindow, row)
		records = append(records, row)
	}
	s.respondFeed(writer, request, records, window.EndCursor, window.HasNext, "|trailer-stats-feed")
}

func (s *Server) handleTrailerStatsHistory(writer http.ResponseWriter, request *http.Request) {
	view, parsed, err := s.parseTrailerStatsRequest(request, true)
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
	for _, trailer := range view.assetsOfType(assetTypeTrailer) {
		if _, ok := parsed.AssetIDs[recordID(trailer)]; !ok {
			continue
		}
		row := view.statRowBase(trailer, nil)
		feedStats(
			view.newStatContext(trailer, window.EventFrom, window.EventTo),
			parsed.Types,
			parsed.Decorations,
			window,
			row,
		)
		records = append(records, row)
	}
	s.respondPage(writer, request, records, "|trailer-stats-history")
}
