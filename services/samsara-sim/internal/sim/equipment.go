package sim

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	maxEquipmentStatTypes      = 4
	digitalOutputGatewayModel  = "AG53"
	digitalOutputPinCount      = 2
	maxDigitalOutputDuration   = 604_800
	fieldSimDigitalOutputs     = "simDigitalOutputs"
	equipmentIdleShare         = 0.18
	equipmentGatewayOffsetSecs = 3_600
)

var equipmentStatRegistry = buildEquipmentStatRegistry()

func buildEquipmentStatRegistry() *statRegistry {
	specs := []*statSpec{
		equipmentScalar(
			"gatewayEngineStates",
			"gatewayEngineState",
			func(ctx *statContext, at time.Time) (any, bool) {
				state, ok := equipmentEngineState(ctx, at)
				if !ok {
					return nil, false
				}
				if state == engineStateIdle {
					return engineStateOn, true
				}
				return state, true
			},
		),
		equipmentScalar("obdEngineStates", "obdEngineState", equipmentEngineStateValue),
		equipmentScalar(
			statTypeFuelPercents,
			"fuelPercent",
			func(ctx *statContext, at time.Time) (any, bool) {
				return anyValue(equipmentFuelPercent(ctx, at))
			},
		),
		equipmentScalar("engineRpm", "engineRpm", func(ctx *statContext, at time.Time) (any, bool) {
			state, ok := equipmentEngineState(ctx, at)
			if !ok {
				return nil, false
			}
			jitter := ctx.hash("equipment-rpm", at.UTC().Truncate(time.Minute).Format(time.RFC3339))
			switch state {
			case engineStateOn:
				return int64(1_450 + 350*jitter), true
			case engineStateIdle:
				return int64(900 + 150*jitter), true
			default:
				return int64(0), true
			}
		}),
		equipmentScalar(
			"gatewayEngineSeconds",
			"gatewayEngineSeconds",
			func(ctx *statContext, at time.Time) (any, bool) {
				seconds, ok := equipmentEngineSeconds(ctx, at)
				if !ok {
					return nil, false
				}
				return seconds + equipmentGatewayOffsetSecs, true
			},
		),
		equipmentScalar(
			"obdEngineSeconds",
			"obdEngineSeconds",
			func(ctx *statContext, at time.Time) (any, bool) {
				return anyValue(equipmentEngineSeconds(ctx, at))
			},
		),
		equipmentScalar(
			"gatewayJ1939EngineSeconds",
			"engineSeconds",
			func(ctx *statContext, at time.Time) (any, bool) {
				if stringValue(
					Record(nestedMap(ctx.asset, fieldInstalledGateway)),
					keyModel,
				) != "AG26" {
					return nil, false
				}
				return anyValue(equipmentEngineSeconds(ctx, at))
			},
		),
		equipmentScalar(
			"gpsOdometerMeters",
			"gpsOdometerMeters",
			manualOdometerMeters,
		),
		{
			Type:        statTypeGPS,
			SnapshotKey: statTypeGPS,
			FeedKey:     statTypeGPS,
			Sample:      equipmentGPSSample,
		},
		equipmentScalar(
			"engineTotalIdleTimeMinutes",
			"engineTotalIdleTimeMinutes",
			func(ctx *statContext, at time.Time) (any, bool) {
				seconds, ok := equipmentEngineSeconds(ctx, at)
				if !ok {
					return nil, false
				}
				return int64(float64(seconds) * equipmentIdleShare / 60), true
			},
		),
	}
	return newStatRegistry("equipment", maxEquipmentStatTypes, 0, specs, nil)
}

func equipmentScalar(
	statType string,
	snapshotKey string,
	value func(ctx *statContext, at time.Time) (any, bool),
) *statSpec {
	spec := scalarStat(statType, snapshotKey, false, value)
	return spec
}

func nestedMap(record Record, key string) map[string]any {
	mapped, ok := anyAsMap(record[key])
	if !ok {
		return map[string]any{}
	}
	return mapped
}

func equipmentHostTrailer(ctx *statContext) Record {
	hostID := stringValue(ctx.asset, fieldSimMountedOnAssetID)
	if hostID == "" {
		return nil
	}
	host, ok := ctx.view.snap.assetByID[hostID]
	if !ok || !reeferOf(host).Present {
		return nil
	}
	return host
}

func (c *statContext) hostContext(host Record) *statContext {
	key := "host|" + recordID(host)
	if cached, ok := c.scratch[key].(*statContext); ok {
		return cached
	}
	hostCtx := &statContext{
		view:      c.view,
		asset:     host,
		assetID:   recordID(host),
		track:     c.track,
		template:  c.view.snap.templates[recordID(host)],
		positions: c.positions,
		hasFix:    c.hasFix,
		scratch:   map[string]any{},
	}
	c.scratch[key] = hostCtx
	return hostCtx
}

func equipmentEngineState(ctx *statContext, at time.Time) (string, bool) {
	state, ok := ctx.position(at)
	if !ok {
		return "", false
	}
	if host := equipmentHostTrailer(ctx); host != nil {
		profile := reeferOf(host)
		if profile.RunMode == reeferRunContinuous {
			return engineStateOn, true
		}
		cycle := at.Sub(telemetryEpoch) % (40 * time.Minute)
		if cycle < time.Duration(profile.RunFraction*float64(40*time.Minute)) {
			return engineStateOn, true
		}
		return engineStateOff, true
	}
	if ctx.track.Kind == trackYardLoop {
		if !yardShiftActive(at) {
			return engineStateOff, true
		}
		if state.SpeedMPS > movingSpeedThresholdMPS {
			return engineStateOn, true
		}
		return engineStateIdle, true
	}
	return engineStateOff, true
}

func equipmentEngineStateValue(ctx *statContext, at time.Time) (any, bool) {
	return equipmentEngineState(ctx, at)
}

func equipmentEngineSeconds(ctx *statContext, at time.Time) (int64, bool) {
	if _, ok := ctx.position(at); !ok {
		return 0, false
	}
	if host := equipmentHostTrailer(ctx); host != nil {
		return reeferEngineSeconds(ctx.hostContext(host), at)
	}
	base := 4_200_000 + ctx.hash("equipment-hours")*3_000_000
	shiftShare := float64(yardShiftEndHour-yardShiftStartHour) / 24 * 6 / 7
	return int64(base + math.Max(at.Sub(telemetryEpoch).Seconds(), 0)*shiftShare), true
}

func equipmentFuelPercent(ctx *statContext, at time.Time) (int64, bool) {
	if host := equipmentHostTrailer(ctx); host != nil {
		return reeferFuelPercent(ctx.hostContext(host), at)
	}
	if _, ok := ctx.position(at); !ok {
		return 0, false
	}
	days := math.Max(at.Sub(telemetryEpoch).Hours()/24, 0)
	cycle := math.Mod(days/3.5+ctx.hash("equipment-fuel"), 1)
	return clampInt64(int64(math.Round(95-78*cycle)), 10, 100), true
}

func equipmentGPSSample(ctx *statContext, at time.Time) map[string]any {
	state, ok := ctx.position(at)
	if !ok {
		return nil
	}
	sample := map[string]any{
		keyLatitude:          round(state.Latitude, 6),
		keyLongitude:         round(state.Longitude, 6),
		keyHeadingDegrees:    round(state.Heading, 1),
		keySpeedMilesPerHour: speedMilesPerHour(state),
	}
	if formatted := ctx.view.snap.reverseGeocode(state); formatted != "" {
		sample[keyReverseGeo] = map[string]any{keyFormattedLocation: formatted}
	}
	if address := ctx.view.snap.addressRef(state); address != nil {
		sample["address"] = address
	}
	return sample
}

func (s *Server) registerEquipmentRoutes() {
	s.mux.HandleFunc("GET /fleet/equipment", s.handleEquipmentList)
	s.mux.HandleFunc("GET /fleet/equipment/locations", s.handleEquipmentLocations)
	s.mux.HandleFunc("GET /fleet/equipment/locations/feed", s.handleEquipmentLocationsFeed)
	s.mux.HandleFunc("GET /fleet/equipment/locations/history", s.handleEquipmentLocationsHistory)
	s.mux.HandleFunc("GET /fleet/equipment/stats", s.handleEquipmentStats)
	s.mux.HandleFunc("GET /fleet/equipment/stats/feed", s.handleEquipmentStatsFeed)
	s.mux.HandleFunc("GET /fleet/equipment/stats/history", s.handleEquipmentStatsHistory)
	s.mux.HandleFunc("GET /fleet/equipment/{id}", s.handleEquipmentGet)
	s.mux.HandleFunc("PATCH /fleet/equipment/{id}/digital-output", s.handleEquipmentDigitalOutput)
}

func (v *fleetView) equipmentView(equipment Record) Record {
	id := recordID(equipment)
	out := Record{
		keyID:            id,
		fieldExternalIDs: renderExternalIDs(equipment, nil),
		keyTags:          v.snap.tags.tinyTags(tagMembersAssets, id),
	}
	for field, viewField := range map[string]string{
		keyName:           keyName,
		keyNotes:          keyNotes,
		fieldSerialNumber: "assetSerial",
	} {
		if value := stringValue(equipment, field); value != "" {
			out[viewField] = value
		}
	}
	if gateway, ok := anyAsMap(equipment[fieldInstalledGateway]); ok {
		out[fieldInstalledGateway] = map[string]any{
			keyModel:  stringValue(Record(gateway), keyModel),
			keySerial: stringValue(Record(gateway), keySerial),
		}
	}
	return out
}

func (v *fleetView) selectEquipment(refs, tagIDs, parentTagIDs []string) map[string]struct{} {
	filter := v.snap.tags.filter(tagIDs, parentTagIDs)
	allowed := map[string]struct{}(nil)
	if len(refs) > 0 {
		allowed = resolvePlainIDs(refs)
	}
	out := map[string]struct{}{}
	for _, equipment := range v.assetsOfType(assetTypeEquipment) {
		id := recordID(equipment)
		if inOptionalSet(allowed, id) && filter.matches(v.snap.tags, tagMembersAssets, id) {
			out[id] = struct{}{}
		}
	}
	return out
}

func (s *Server) handleEquipmentList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	selected := view.selectEquipment(nil, tagIDs, parentTagIDs)
	records := make([]Record, 0, len(selected))
	for _, equipment := range view.assetsOfType(assetTypeEquipment) {
		if _, ok := selected[recordID(equipment)]; ok {
			records = append(records, view.equipmentView(equipment))
		}
	}
	s.respondPage(writer, request, records, "|equipment-list")
}

func (s *Server) handleEquipmentGet(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	equipment, ok := view.snap.assetByID[id]
	if !ok || assetType(equipment) != assetTypeEquipment {
		s.writeError(writer, notFound("equipment", id))
		return
	}
	payload := map[string]any{keyData: view.equipmentView(equipment)}
	s.respondJSON(writer, request, requestSignature(request)+"|equipment-get", payload)
}

func (v *fleetView) equipmentLocation(state routeState, at time.Time) map[string]any {
	return map[string]any{
		"heading":    round(state.Heading, 1),
		keyLatitude:  round(state.Latitude, 6),
		keyLongitude: round(state.Longitude, 6),
		keySpeed:     speedMilesPerHour(state),
		fieldTime:    formatSampleTime(at),
	}
}

func (v *fleetView) equipmentLocationRecords(
	selected map[string]struct{},
	times []time.Time,
) []Record {
	records := make([]Record, 0, len(selected))
	if len(times) == 0 {
		return records
	}
	for _, equipment := range v.assetsOfType(assetTypeEquipment) {
		id := recordID(equipment)
		if _, ok := selected[id]; !ok {
			continue
		}
		track := v.trackFor(id, times[0], times[len(times)-1])
		locations := make([]any, 0, len(times))
		for _, at := range times {
			if state, ok := v.live.trackStateAt(track, at); ok {
				locations = append(locations, v.equipmentLocation(state, at))
			}
		}
		if len(locations) == 0 {
			continue
		}
		records = append(records, Record{
			keyID:        id,
			keyName:      stringValue(equipment, keyName),
			keyLocations: locations,
		})
	}
	return records
}

func (s *Server) handleEquipmentLocations(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	selected := view.selectEquipment(csvQueryValues(values, "equipmentIds"), tagIDs, parentTagIDs)
	at := view.now.Truncate(time.Second)
	records := make([]Record, 0, len(selected))
	for _, record := range view.equipmentLocationRecords(selected, []time.Time{at}) {
		locations := listOf(record[keyLocations])
		records = append(records, Record{
			keyID:       record[keyID],
			keyName:     record[keyName],
			keyLocation: locations[len(locations)-1],
		})
	}
	s.respondPage(writer, request, records, "|equipment-locations")
}

func (s *Server) handleEquipmentLocationsFeed(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	selected := view.selectEquipment(csvQueryValues(values, "equipmentIds"), tagIDs, parentTagIDs)
	window, after, err := parseFeedWindow(
		request,
		view.now,
		len(selected),
		pagePolicyFor(request).Size,
	)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if len(window.Times) == 0 {
		s.respondFeed(writer, request, []Record{}, after, false, "|equipment-locations-feed")
		return
	}
	records := view.equipmentLocationRecords(selected, window.Times)
	s.respondFeed(
		writer,
		request,
		records,
		window.EndCursor,
		window.HasNext,
		"|equipment-locations-feed",
	)
}

func (s *Server) handleEquipmentLocationsHistory(
	writer http.ResponseWriter,
	request *http.Request,
) {
	values := request.URL.Query()
	view := s.fleetView()
	window, err := parseHistoryWindow(request, view.now)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	tagIDs, parentTagIDs := standardTagParams(values)
	selected := view.selectEquipment(csvQueryValues(values, "equipmentIds"), tagIDs, parentTagIDs)
	s.respondPage(
		writer,
		request,
		view.equipmentLocationRecords(selected, window.Times),
		"|equipment-locations-history",
	)
}

func (s *Server) parseEquipmentStatsRequest(
	request *http.Request,
) (*fleetView, *statsRequest, error) {
	values := request.URL.Query()
	types, err := equipmentStatRegistry.parseTypes(statTypesFromQuery(values, "types"))
	if err != nil {
		return nil, nil, err
	}
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	return view, &statsRequest{
		Types: types,
		AssetIDs: view.selectEquipment(
			csvQueryValues(values, "equipmentIds"),
			tagIDs,
			parentTagIDs,
		),
	}, nil
}

func (s *Server) handleEquipmentStats(writer http.ResponseWriter, request *http.Request) {
	view, parsed, err := s.parseEquipmentStatsRequest(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	at := view.now.Truncate(time.Second)
	records := make([]Record, 0, len(parsed.AssetIDs))
	for _, equipment := range view.assetsOfType(assetTypeEquipment) {
		if _, ok := parsed.AssetIDs[recordID(equipment)]; !ok {
			continue
		}
		row := Record{keyID: recordID(equipment), keyName: stringValue(equipment, keyName)}
		snapshotStats(view.newStatContext(equipment, at, at), parsed.Types, at, row)
		records = append(records, row)
	}
	s.respondPage(writer, request, records, "|equipment-stats")
}

func (s *Server) handleEquipmentStatsFeed(writer http.ResponseWriter, request *http.Request) {
	view, parsed, err := s.parseEquipmentStatsRequest(request)
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
		s.respondFeed(writer, request, []Record{}, after, false, "|equipment-stats-feed")
		return
	}
	records := view.equipmentStatRows(parsed, &window.feedWindow)
	s.respondFeed(
		writer,
		request,
		records,
		window.EndCursor,
		window.HasNext,
		"|equipment-stats-feed",
	)
}

func (s *Server) handleEquipmentStatsHistory(writer http.ResponseWriter, request *http.Request) {
	view, parsed, err := s.parseEquipmentStatsRequest(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	window, err := parseHistoryWindow(request, view.now)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondPage(
		writer,
		request,
		view.equipmentStatRows(parsed, window),
		"|equipment-stats-history",
	)
}

func (v *fleetView) equipmentStatRows(parsed *statsRequest, window *feedWindow) []Record {
	records := make([]Record, 0, len(parsed.AssetIDs))
	for _, equipment := range v.assetsOfType(assetTypeEquipment) {
		if _, ok := parsed.AssetIDs[recordID(equipment)]; !ok {
			continue
		}
		row := Record{keyID: recordID(equipment), keyName: stringValue(equipment, keyName)}
		feedStats(
			v.newStatContext(equipment, window.EventFrom, window.EventTo),
			parsed.Types,
			nil,
			window,
			row,
		)
		records = append(records, row)
	}
	return records
}

func (s *Server) handleEquipmentDigitalOutput(writer http.ResponseWriter, request *http.Request) {
	rawID, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	gatewayID, parseErr := strconv.ParseInt(rawID, 10, 64)
	if parseErr != nil || gatewayID <= 0 {
		s.writeError(writer, invalidParameter(keyID, "must be an integer Samsara ID"))
		return
	}
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, []fieldRule{
		{
			Name:     "durationSeconds",
			Kind:     kindInt,
			IntRange: &intRange{Min: 0, Max: maxDigitalOutputDuration},
		},
		{Name: "pinId", Kind: kindInt, Required: true, IntRange: &intRange{Min: 1, Max: 1 << 31}},
		{Name: keyState, Kind: kindBool, Required: true},
	}, bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	pinID, _ := int64Value(sanitized["pinId"])
	if pinID > digitalOutputPinCount {
		s.writeError(writer, invalidField("pinId", "must be 1 or 2 on an AG53 gateway"))
		return
	}
	duration, _ := sanitized["durationSeconds"].(int64)
	state, _ := boolValue(sanitized, keyState)
	now := s.simNow()
	id := strconv.FormatInt(gatewayID, 10)
	command := digitalOutputCommand{
		EquipmentID: id,
		PinID:       pinID,
		State:       state,
		Duration:    time.Duration(duration) * time.Second,
		At:          now,
	}
	err = s.store.Transact(func(tx *storeTx) error {
		return setDigitalOutputTx(tx, &command)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	payload := map[string]any{keyData: map[string]any{
		"durationSeconds": duration,
		keyID:             gatewayID,
		"pinId":           pinID,
		keyState:          state,
	}}
	s.respondJSON(writer, request, requestSignature(request)+"|equipment-digital-output", payload)
}

type digitalOutputCommand struct {
	EquipmentID string
	PinID       int64
	State       bool
	Duration    time.Duration
	At          time.Time
}

func setDigitalOutputTx(tx *storeTx, command *digitalOutputCommand) error {
	equipment, idx := tx.find(ResourceAssets, command.EquipmentID)
	if idx < 0 || assetType(equipment) != assetTypeEquipment {
		return notFound("equipment gateway", command.EquipmentID)
	}
	model := stringValue(Record(nestedMap(equipment, fieldInstalledGateway)), keyModel)
	if !strings.EqualFold(model, digitalOutputGatewayModel) {
		return invalidField(keyID, "must identify an AG53 gateway connected through CBL-AG-BEQP")
	}
	_, err := tx.update(ResourceAssets, command.EquipmentID, func(record Record) error {
		outputs := cloneMap(nestedMap(record, fieldSimDigitalOutputs))
		entry := map[string]any{
			keyState:    command.State,
			"setAtTime": command.At.UTC().Format(time.RFC3339),
		}
		if command.Duration > 0 {
			entry["expiresAtTime"] = command.At.Add(command.Duration).UTC().Format(time.RFC3339)
		}
		outputs[strconv.FormatInt(command.PinID, 10)] = entry
		record[fieldSimDigitalOutputs] = outputs
		return nil
	}, command.At)
	return err
}
