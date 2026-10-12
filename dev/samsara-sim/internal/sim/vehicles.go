package sim

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxVehicleLicensePlate = 12
	maxVehicleNotes        = 255
	minVehicleVinLength    = 11
	maxVehicleVinLength    = 17
	fieldGateway           = "gateway"
	fieldAuxInputTypePref  = "auxInputType"
	defaultVehicleGateway  = "VG54NA"
	poundsPerKilogram      = 2.20462
	relayOne               = "relay1"
	relayTwo               = "relay2"
)

var (
	vehicleAuxInputTypes = []string{
		auxInputNone,
		auxInputEmergencyLights,
		"emergencyAlarm",
		"stopPaddle",
		auxInputPowerTakeOff,
		"plow",
		"sweeper",
		"salter",
		auxInputReefer,
		auxInputDoor,
		"boom",
		auxInputAuxiliaryEngine,
		auxInputGenerator,
		auxInputEightWayLights,
		"panicButton",
		"privacyButton",
		"frontAxleDrive",
		"weightSensor",
		"other",
		"secondaryFuelSource",
		auxInputEcuPowerTakeOff,
	}
	harshAccelerationSettings = []string{
		"passengerCar",
		"lightTruck",
		"heavyDuty",
		"off",
		"automatic",
	}
	vehicleRegulationModes  = []string{keyRegulated, keyUnregulated, regulationModeMixed}
	vehicleTypes            = []string{"unset", "passenger", "truck", "bus"}
	gatewaySerialPattern    = regexp.MustCompile(`^[A-Z0-9]{4}-[A-Z0-9]{3}-[A-Z0-9]{3}$`)
	vehiclePatchPassthrough = buildVehiclePatchPassthrough()
	vehicleViewPassthrough  = []string{
		"cameraSerial", "esn", keyHarshAcceleration, keyLicensePlate, keyMake, keyModel,
		keyName, keyNotes, keyVehicleType, keyVIN, "isRemotePrivacyButtonEnabled",
		"sensorConfiguration",
	}
)

func (s *Server) registerVehicleRoutes() {
	s.mux.HandleFunc("GET /fleet/vehicles", s.handleVehicleList)
	s.mux.HandleFunc("GET /fleet/vehicles/{id}", s.handleVehicleGet)
	s.mux.HandleFunc("PATCH /fleet/vehicles/{id}", s.handleVehiclePatch)
	s.mux.HandleFunc("GET /fleet/vehicles/locations", s.handleVehicleLocations)
	s.mux.HandleFunc("GET /fleet/vehicles/locations/feed", s.handleVehicleLocationsFeed)
	s.mux.HandleFunc("GET /fleet/vehicles/locations/history", s.handleVehicleLocationsHistory)
	s.mux.HandleFunc("GET /fleet/vehicles/immobilizer/stream", s.handleVehicleImmobilizerStream)
	s.registerVehicleStatsRoutes()
}

func (v *fleetView) currentDriverOf(vehicleID string) string {
	return v.live.driverByVehicleMap()[vehicleID]
}

func (v *fleetView) vehicleBaseView(vehicle Record) Record {
	id := recordID(vehicle)
	out := Record{keyID: id}
	for _, field := range vehicleViewPassthrough {
		if value, ok := vehicle[field]; ok && value != nil && value != "" {
			out[field] = cloneAny(value)
		}
	}
	for index := 1; index <= vehicleAuxInputCount; index++ {
		field := fieldAuxInputTypePref + strconv.Itoa(index)
		if value := stringValue(vehicle, field); value != "" {
			out[field] = value
		}
	}
	if year, ok := int64Value(vehicle[fieldYear]); ok && year > 0 {
		out[fieldYear] = strconv.FormatInt(year, 10)
	}
	if gateway, ok := anyAsMap(
		vehicle[fieldGateway],
	); ok &&
		stringValue(Record(gateway), keySerial) != "" {
		out[fieldGateway] = map[string]any{
			keyModel:  stringValue(Record(gateway), keyModel),
			keySerial: stringValue(Record(gateway), keySerial),
		}
		out[keySerial] = stringValue(Record(gateway), keySerial)
	}
	out[fieldExternalIDs] = renderExternalIDs(vehicle, vehicleAutoExternalIDs)
	out[keyTags] = v.snap.tags.tinyTags(tagMembersVehicles, id)
	out[fieldAttributesKey] = renderAttributes(vehicle)
	if driverID := v.currentDriverOf(id); driverID != "" {
		out["staticAssignedDriver"] = v.driverTiny(driverID)
	}
	return out
}

func (v *fleetView) vehicleView(vehicle Record) Record {
	out := v.vehicleBaseView(vehicle)
	if mode := stringValue(vehicle, fieldRegulationMode); mode != "" {
		out["vehicleRegulationMode"] = mode
	}
	if weight, ok := anyAsMap(vehicle["grossVehicleWeight"]); ok {
		out["grossVehicleWeight"] = cloneMap(weight)
	}
	return out
}

func (v *fleetView) vehicleListView(vehicle Record) Record {
	out := v.vehicleBaseView(vehicle)
	out[fieldCreatedAtTime] = stringValue(vehicle, fieldCreatedAtTime)
	out[fieldUpdatedAtTime] = firstNonEmptyField(vehicle, fieldUpdatedAtTime, fieldCreatedAtTime)
	switch mode := stringValue(vehicle, fieldRegulationMode); mode {
	case keyRegulated:
		out["vehicleRegulationMode"] = mode
	case keyUnregulated, regulationModeMixed:
		out["vehicleRegulationMode"] = keyUnregulated
	}
	if weight, ok := anyAsMap(vehicle["grossVehicleWeight"]); ok {
		value, _ := int64Value(weight["weight"])
		out["vehicleWeight"] = value
		if stringValue(Record(weight), "unit") == "kg" {
			out["vehicleWeightInKilograms"] = value
			out["vehicleWeightInPounds"] = int64(float64(value)*poundsPerKilogram + 0.5)
		} else {
			out["vehicleWeightInPounds"] = value
			out["vehicleWeightInKilograms"] = int64(float64(value)/poundsPerKilogram + 0.5)
		}
	}
	return out
}

func (s *Server) handleVehicleList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	attributes, err := parseAttributeFilter(values, attributeFilterVehicle)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	updatedAfter, err := parseOptionalTimePtr(values, "updatedAfterTime")
	if err != nil {
		s.writeError(writer, err)
		return
	}
	createdAfter, err := parseOptionalTimePtr(values, "createdAfterTime")
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	filter := view.snap.tags.filter(tagIDs, parentTagIDs)
	vehicles := view.vehicleRecords()
	records := make([]Record, 0, len(vehicles))
	for _, vehicle := range vehicles {
		if !filter.matches(view.snap.tags, tagMembersVehicles, recordID(vehicle)) ||
			!attributes.matches(vehicle) ||
			!recordAtOrAfter(vehicle, fieldUpdatedAtTime, updatedAfter) ||
			!recordAtOrAfter(vehicle, fieldCreatedAtTime, createdAfter) {
			continue
		}
		records = append(records, view.vehicleListView(vehicle))
	}
	s.respondPage(writer, request, records, "|vehicle-list")
}

func (v *fleetView) findVehicle(ref string) (Record, bool) {
	vehicle, idx := findRecordByRef(v.vehicleRecords(), ref, vehicleAutoExternalIDs)
	return vehicle, idx >= 0
}

func (s *Server) handleVehicleGet(writer http.ResponseWriter, request *http.Request) {
	ref, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	vehicle, ok := view.findVehicle(ref)
	if !ok {
		s.writeError(writer, notFound(keyVehicle, ref))
		return
	}
	payload := map[string]any{keyData: view.vehicleView(vehicle)}
	s.respondJSON(writer, request, requestSignature(request)+"|vehicle-get", payload)
}

func buildVehiclePatchPassthrough() []string {
	fields := []string{
		fieldExternalIDs, "grossVehicleWeight", keyHarshAcceleration, keyLicensePlate,
		keyName, keyNotes, keyVehicleType, keyVIN,
	}
	for index := 1; index <= vehicleAuxInputCount; index++ {
		fields = append(fields, fieldAuxInputTypePref+strconv.Itoa(index))
	}
	return fields
}

func vehiclePatchRules() []fieldRule {
	rules := []fieldRule{
		{Name: fieldAttributesKey, Kind: kindAttributes},
		{Name: "engineHours", Kind: kindInt, IntRange: &intRange{Min: 0, Max: 10_000_000}},
		{Name: fieldExternalIDs, Kind: kindExternalIDs, Nullable: true},
		{Name: "gatewaySerial", Kind: kindString, Check: validateGatewaySerial},
		{
			Name:     "grossVehicleWeight",
			Kind:     kindObject,
			Nullable: true,
			Fields: []fieldRule{
				{Name: "unit", Kind: kindString, Required: true, Enum: []string{"lb", "kg"}},
				{
					Name:     "weight",
					Kind:     kindInt,
					Required: true,
					IntRange: &intRange{Min: 1, Max: 500_000},
				},
			},
		},
		{Name: keyHarshAcceleration, Kind: kindString, Enum: harshAccelerationSettings},
		{Name: keyLicensePlate, Kind: kindString, MaxLen: maxVehicleLicensePlate, Nullable: true},
		{Name: keyName, Kind: kindString, MinLen: 1, MaxLen: 255, Check: nonBlank(keyName)},
		{Name: keyNotes, Kind: kindString, MaxLen: maxVehicleNotes, Nullable: true},
		{Name: keyOdometerMeters, Kind: kindInt, IntRange: &intRange{Min: 0, Max: 10_000_000_000}},
		{Name: "staticAssignedDriverId", Kind: kindString, Nullable: true},
		{Name: fieldTagIDs, Kind: kindStringList},
		{Name: "vehicleRegulationMode", Kind: kindString, Enum: vehicleRegulationModes},
		{Name: keyVehicleType, Kind: kindString, Enum: vehicleTypes},
		{
			Name:   keyVIN,
			Kind:   kindString,
			MinLen: minVehicleVinLength,
			MaxLen: maxVehicleVinLength,
			Check:  validateVIN,
		},
	}
	for index := 1; index <= vehicleAuxInputCount; index++ {
		rules = append(rules, fieldRule{
			Name: fieldAuxInputTypePref + strconv.Itoa(index),
			Kind: kindString,
			Enum: vehicleAuxInputTypes,
		})
	}
	return rules
}

func validateGatewaySerial(value any) error {
	serial := strings.ToUpper(strings.TrimSpace(stringOf(value)))
	if !gatewaySerialPattern.MatchString(serial) {
		return invalidField("gatewaySerial", "must look like XXXX-XXX-XXX")
	}
	return nil
}

func (s *Server) handleVehiclePatch(writer http.ResponseWriter, request *http.Request) {
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
	sanitized, err := sanitizeBody(body, vehiclePatchRules(), bodyPatch)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	vehicleID, err := s.store.TransactID(func(tx *storeTx) (string, error) {
		return patchVehicleTx(tx, ref, sanitized, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	vehicle, ok := view.snap.assetByID[vehicleID]
	if !ok {
		s.writeError(writer, notFound(keyVehicle, vehicleID))
		return
	}
	data := view.vehicleView(vehicle)
	s.dispatchEvent(request, eventVehicleUpdated, map[string]any{keyVehicle: cloneRecord(data)})
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+"|vehicle-patch",
		map[string]any{keyData: data},
	)
}

func patchVehicleTx(tx *storeTx, ref string, body Record, now time.Time) (string, error) {
	current, idx := findVehicleTx(tx, ref)
	if idx < 0 {
		return "", notFound(keyVehicle, ref)
	}
	vehicleID := recordID(current)
	_, err := tx.update(ResourceAssets, vehicleID, func(record Record) error {
		return applyVehiclePatch(tx, record, body, now)
	}, now)
	if err != nil {
		return "", err
	}
	if raw, ok := body["staticAssignedDriverId"]; ok {
		if err = setStaticDriverTx(tx, vehicleID, raw, now); err != nil {
			return "", err
		}
	}
	if raw, ok := body[fieldTagIDs]; ok {
		return vehicleID, setEntityTagsTx(tx, tagMembersVehicles, vehicleID, stringListValues(raw))
	}
	return vehicleID, nil
}

func applyVehiclePatch(tx *storeTx, record, body Record, now time.Time) error {
	if err := ensureVehicleIdentifiersUnique(tx, recordID(record), body); err != nil {
		return err
	}
	if serial, ok := body["gatewaySerial"].(string); ok {
		model := defaultVehicleGateway
		if gateway, exists := anyAsMap(record[fieldGateway]); exists {
			model = stringutils.FirstNonEmptyTrimmed(
				stringValue(Record(gateway), keyModel),
				defaultVehicleGateway,
			)
		}
		serial = strings.ToUpper(strings.TrimSpace(serial))
		record[fieldGateway] = map[string]any{keyModel: model, keySerial: serial}
		record[fieldSerialNumber] = serial
	}
	applyManualReadings(record, body, now)
	if mode, ok := body["vehicleRegulationMode"].(string); ok {
		record[fieldRegulationMode] = mode
	}
	if raw, ok := body[fieldAttributesKey]; ok {
		record[fieldAttributesKey] = resolveAttributeIDs(listOf(raw), attributeEntityAsset)
	}
	for _, field := range vehiclePatchPassthrough {
		raw, ok := body[field]
		if !ok {
			continue
		}
		if raw == nil {
			delete(record, field)
			continue
		}
		record[field] = raw
	}
	return nil
}

func ensureVehicleIdentifiersUnique(tx *storeTx, selfID string, body Record) error {
	assets := tx.records(ResourceAssets)
	if raw, ok := body[fieldExternalIDs]; ok && raw != nil {
		if err := tx.ensureExternalIDs(externalIDClaim{
			Kind:  externalIDKindVehicles,
			Owner: externalIDOwner(ResourceAssets, selfID),
			IDs:   mapOf(raw),
		}); err != nil {
			return err
		}
	}
	if vin, ok := body[keyVIN].(string); ok {
		if err := ensureVINUnique(assets, selfID, vin); err != nil {
			return err
		}
	}
	serial, ok := body["gatewaySerial"].(string)
	if !ok {
		return nil
	}
	serial = strings.ToUpper(strings.TrimSpace(serial))
	for _, asset := range assets {
		if recordID(asset) != selfID && strings.EqualFold(assetGatewaySerial(asset), serial) {
			return fmt.Errorf(
				"%w: gateway %s is installed on another asset",
				ErrUniqueConflict,
				serial,
			)
		}
	}
	return nil
}

func ensureVINUnique(assets []Record, selfID, vin string) error {
	if strings.TrimSpace(vin) == "" {
		return nil
	}
	for _, asset := range assets {
		if recordID(asset) != selfID && strings.EqualFold(stringValue(asset, keyVIN), vin) {
			return fmt.Errorf(
				"%w: vin %s is already assigned to another asset",
				ErrUniqueConflict,
				vin,
			)
		}
	}
	return nil
}

func applyManualReadings(record, body Record, now time.Time) {
	stamp := now.UTC().Format(time.RFC3339)
	if hours, ok := body["engineHours"].(int64); ok {
		record["simEngineHoursReading"] = map[string]any{"hours": hours, fieldTime: stamp}
	}
	if meters, ok := body[keyOdometerMeters].(int64); ok {
		record["simOdometerReading"] = map[string]any{"meters": meters, fieldTime: stamp}
	}
}

func setStaticDriverTx(tx *storeTx, vehicleID string, raw any, now time.Time) error {
	drivers := tx.records(ResourceDrivers)
	targetID := ""
	if ref, ok := raw.(string); ok && strings.TrimSpace(ref) != "" {
		driver, idx := findRecordByRef(drivers, ref, nil)
		if idx < 0 {
			return missingReference(keyDriver, ref)
		}
		if driverDeactivated(driver) {
			return invalidField("staticAssignedDriverId", "refers to a deactivated driver")
		}
		targetID = recordID(driver)
	}
	stamp := now.UTC().Format(time.RFC3339)
	for idx, driver := range drivers {
		id := recordID(driver)
		current := stringValue(driver, fieldStaticAssignedVehicle)
		switch {
		case id == targetID && current != vehicleID:
			next := cloneRecord(driver)
			next[fieldStaticAssignedVehicle] = vehicleID
			next[fieldSimStaticAssignedAt] = stamp
			drivers[idx] = next
		case id != targetID && current == vehicleID:
			next := cloneRecord(driver)
			delete(next, fieldStaticAssignedVehicle)
			delete(next, fieldSimStaticAssignedAt)
			drivers[idx] = next
		}
	}
	return nil
}

func (v *fleetView) locationRecord(state routeState, at time.Time) map[string]any {
	location := map[string]any{
		"heading":    round(state.Heading, 1),
		keyLatitude:  round(state.Latitude, 6),
		keyLongitude: round(state.Longitude, 6),
		keySpeed:     speedMilesPerHour(state),
		fieldTime:    formatSampleTime(at),
	}
	if formatted := v.snap.reverseGeocode(state); formatted != "" {
		location[keyReverseGeo] = map[string]any{keyFormattedLocation: formatted}
	}
	return location
}

func (s *Server) handleVehicleLocations(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	view := s.fleetView()
	at, err := snapshotTime(request, view.now)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	tagIDs, parentTagIDs := standardTagParams(values)
	selected := view.selectVehicles(
		csvQueryValues(values, "vehicleIds"),
		tagIDs,
		parentTagIDs,
		false,
	)
	records := make([]Record, 0, len(selected))
	for _, vehicle := range view.vehicleRecords() {
		id := recordID(vehicle)
		if _, ok := selected[id]; !ok {
			continue
		}
		track := view.trackFor(id, at, at)
		state, ok := view.live.trackStateAt(track, at)
		if !ok {
			continue
		}
		records = append(records, Record{
			keyID:       id,
			keyName:     stringValue(vehicle, keyName),
			keyLocation: view.locationRecord(state, at),
		})
	}
	s.respondPage(writer, request, records, "|vehicle-locations")
}

func (v *fleetView) locationSeries(assetID string, times []time.Time) []any {
	out := make([]any, 0, len(times))
	if len(times) == 0 {
		return out
	}
	track := v.trackFor(assetID, times[0], times[len(times)-1])
	for _, at := range times {
		if state, ok := v.live.trackStateAt(track, at); ok {
			out = append(out, v.locationRecord(state, at))
		}
	}
	return out
}

func (s *Server) handleVehicleLocationsFeed(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	selected := view.selectVehicles(
		csvQueryValues(values, "vehicleIds"),
		tagIDs,
		parentTagIDs,
		false,
	)
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
		s.respondFeed(writer, request, []Record{}, after, false, "|vehicle-locations-feed")
		return
	}
	records := view.locationSeriesRecords(view.vehicleRecords(), selected, window.Times)
	s.respondFeed(
		writer,
		request,
		records,
		window.EndCursor,
		window.HasNext,
		"|vehicle-locations-feed",
	)
}

func (v *fleetView) locationSeriesRecords(
	assets []Record,
	selected map[string]struct{},
	times []time.Time,
) []Record {
	records := make([]Record, 0, len(selected))
	for _, asset := range assets {
		id := recordID(asset)
		if _, ok := selected[id]; !ok {
			continue
		}
		locations := v.locationSeries(id, times)
		if len(locations) == 0 {
			continue
		}
		records = append(records, Record{
			keyID:        id,
			keyName:      stringValue(asset, keyName),
			keyLocations: locations,
		})
	}
	return records
}

func (s *Server) handleVehicleLocationsHistory(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	view := s.fleetView()
	window, err := parseHistoryWindow(request, view.now)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	tagIDs, parentTagIDs := standardTagParams(values)
	selected := view.selectVehicles(
		csvQueryValues(values, "vehicleIds"),
		tagIDs,
		parentTagIDs,
		false,
	)
	records := view.locationSeriesRecords(view.vehicleRecords(), selected, window.Times)
	s.respondPage(writer, request, records, "|vehicle-locations-history")
}

func (s *Server) handleVehicleImmobilizerStream(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	refs := csvQueryValues(values, "vehicleIds")
	if len(refs) == 0 {
		s.writeError(writer, invalidParameter("vehicleIds", "is required"))
		return
	}
	start, hasStart, err := parseOptionalTime(values, fieldStartTime)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if !hasStart {
		s.writeError(writer, invalidParameter(fieldStartTime, "is required"))
		return
	}
	end, hasEnd, err := parseOptionalTime(values, fieldEndTime)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if hasEnd && end.Before(start) {
		s.writeError(writer, invalidParameter(fieldEndTime, "must not be before startTime"))
		return
	}
	view := s.fleetView()
	windowEnd := view.now
	if hasEnd {
		windowEnd = minTime(end, view.now)
	}
	selected := view.selectVehicles(refs, nil, nil, true)
	records := make([]Record, 0, 8)
	for _, vehicle := range view.vehicleRecords() {
		id := recordID(vehicle)
		if _, ok := selected[id]; !ok {
			continue
		}
		ctx := view.newStatContext(vehicle, start, windowEnd)
		for _, event := range immobilizerEvents(ctx, start, windowEnd) {
			records = append(records, Record{
				keyHappenedAtTime:      formatSampleTime(event.At),
				"isConnectedToVehicle": event.Connected,
				"relayStates": []any{
					map[string]any{keyID: relayOne, "isOpen": false},
					map[string]any{keyID: relayTwo, "isOpen": false},
				},
				"vehicleId": id,
			})
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		left := stringValue(records[i], keyHappenedAtTime)
		right := stringValue(records[j], keyHappenedAtTime)
		if left != right {
			return left < right
		}
		return stringValue(records[i], "vehicleId") < stringValue(records[j], "vehicleId")
	})
	s.respondPage(writer, request, records, "|vehicle-immobilizer-stream")
}
