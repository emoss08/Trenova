package sim

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	assetTypeVehicle       = keyVehicle
	assetTypeTrailer       = "trailer"
	assetTypeEquipment     = "equipment"
	assetTypeUnpowered     = "unpowered"
	assetTypeUncategorized = "uncategorized"

	fieldRegulationMode = "regulationMode"
	fieldSerialNumber   = "serialNumber"
	fieldYear           = "year"

	eventVehicleCreated = "VehicleCreated"
	eventVehicleUpdated = "VehicleUpdated"

	minAssetYear = 1900
	maxAssetYear = 2100
)

var (
	assetTypeValues = []string{
		assetTypeUncategorized,
		assetTypeTrailer,
		assetTypeEquipment,
		assetTypeUnpowered,
		assetTypeVehicle,
	}
	assetRegulationModes  = []string{regulationModeMixed, keyRegulated, keyUnregulated}
	assetRestrictedFields = []string{keyMake, keyModel, fieldYear}
	assetWritableFields   = []string{
		keyName, keyLicensePlate, keyMake, keyModel, keyNotes, keyReadingsIngestion,
		fieldRegulationMode, fieldSerialNumber, keyType, keyVIN, fieldYear, fieldExternalIDs,
		fieldAttributesKey,
	}
)

type assetViewOptions struct {
	ExternalIDs bool
	Tags        bool
	Attributes  bool
}

func (s *Server) registerAssetRoutes() {
	s.mux.HandleFunc("GET /assets", s.handleAssetList)
	s.mux.HandleFunc("POST /assets", s.handleAssetCreate)
	s.mux.HandleFunc("PATCH /assets", s.handleAssetPatch)
	s.mux.HandleFunc("DELETE /assets", s.handleAssetDelete)
	s.mux.HandleFunc("GET /assets/location-and-speed/stream", s.handleAssetLocationStream)
}

func assetType(asset Record) string {
	if value := stringValue(asset, keyType); value != "" {
		return value
	}
	return assetTypeUncategorized
}

func assetAutoExternalIDs(asset Record) map[string]string {
	if assetType(asset) == assetTypeVehicle {
		return vehicleAutoExternalIDs(asset)
	}
	return nil
}

func (v *fleetView) assetView(asset Record, options assetViewOptions) Record {
	out := Record{
		keyID:              recordID(asset),
		keyType:            assetType(asset),
		fieldCreatedAtTime: stringValue(asset, fieldCreatedAtTime),
		fieldUpdatedAtTime: firstNonEmptyField(
			asset,
			fieldUpdatedAtTime,
			fieldCreatedAtTime,
		),
		keyReadingsIngestion: asset[keyReadingsIngestion] == true,
	}
	for _, field := range []string{
		keyName, keyLicensePlate, keyMake, keyModel, keyNotes, fieldRegulationMode, fieldSerialNumber, keyVIN,
	} {
		if value := stringValue(asset, field); value != "" {
			out[field] = value
		}
	}
	if year, ok := int64Value(asset[fieldYear]); ok && year > 0 {
		out[fieldYear] = year
	}
	if options.ExternalIDs {
		out[fieldExternalIDs] = renderExternalIDs(asset, nil)
	}
	if options.Tags {
		out[keyTags] = v.snap.tags.tinyTags(assetTagKind(asset), recordID(asset))
	}
	if options.Attributes {
		out[fieldAttributesKey] = renderAttributes(asset)
	}
	return out
}

func parseBoolParam(values url.Values, name string) (bool, error) {
	raw := strings.TrimSpace(values.Get(name))
	switch strings.ToLower(raw) {
	case "":
		return false, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, invalidParameter(name, "must be `true` or `false`")
	}
}

func (s *Server) handleAssetList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	typeFilter := strings.TrimSpace(values.Get(keyType))
	if typeFilter != "" && !slices.Contains(assetTypeValues, typeFilter) {
		s.writeError(writer, ErrAssetTypeInvalid)
		return
	}
	updatedAfter, err := parseOptionalTimePtr(values, "updatedAfterTime")
	if err != nil {
		s.writeError(writer, err)
		return
	}
	attributes, err := parseAttributeFilter(values, attributeFilterAsset)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	options := assetViewOptions{}
	for name, target := range map[string]*bool{
		"includeExternalIds": &options.ExternalIDs,
		"includeTags":        &options.Tags,
		"includeAttributes":  &options.Attributes,
	} {
		if *target, err = parseBoolParam(values, name); err != nil {
			s.writeError(writer, err)
			return
		}
	}
	externalRefs := csvQueryValues(values, "externalIds")
	for _, ref := range externalRefs {
		if parsed := parseRecordRef(ref); !parsed.isExternal() || parsed.Value == "" {
			s.writeError(
				writer,
				invalidParameter(
					"externalIds",
					fmt.Sprintf("%q must use the key:value format", ref),
				),
			)
			return
		}
	}
	view := s.fleetView()
	var idFilter map[string]struct{}
	if refs := csvQueryValues(values, "ids"); len(refs) > 0 {
		idFilter = resolveRecordRefs(view.snap.assets, refs, assetAutoExternalIDs)
	}
	var externalFilter map[string]struct{}
	if len(externalRefs) > 0 {
		externalFilter = resolveRecordRefs(view.snap.assets, externalRefs, assetAutoExternalIDs)
	}
	tagIDs, parentTagIDs := standardTagParams(values)
	tagFilter := view.snap.tags.filter(tagIDs, parentTagIDs)
	records := make([]Record, 0, len(view.snap.assets))
	for _, asset := range view.snap.assets {
		id := recordID(asset)
		if typeFilter != "" && assetType(asset) != typeFilter {
			continue
		}
		if !inOptionalSet(idFilter, id) || !inOptionalSet(externalFilter, id) {
			continue
		}
		if !tagFilter.matches(view.snap.tags, assetTagKind(asset), id) ||
			!attributes.matches(asset) ||
			!recordAtOrAfter(asset, fieldUpdatedAtTime, updatedAfter) {
			continue
		}
		records = append(records, view.assetView(asset, options))
	}
	s.respondPage(writer, request, records, "|asset-list")
}

func inOptionalSet(set map[string]struct{}, id string) bool {
	if set == nil {
		return true
	}
	_, ok := set[id]
	return ok
}

func assetBodyRules(mode bodyMode) []fieldRule {
	return []fieldRule{
		{Name: fieldAttributesKey, Kind: kindAttributes},
		{Name: fieldExternalIDs, Kind: kindExternalIDs, Nullable: true},
		{Name: keyLicensePlate, Kind: kindString, MaxLen: 255, Nullable: true},
		{Name: keyMake, Kind: kindString, MaxLen: 255, Nullable: true},
		{Name: keyModel, Kind: kindString, MaxLen: 255, Nullable: true},
		{Name: keyName, Kind: kindString, MinLen: 1, MaxLen: 255, Check: nonBlank(keyName)},
		{Name: keyNotes, Kind: kindString, MaxLen: 4096, Nullable: true},
		{Name: keyReadingsIngestion, Kind: kindBool},
		{Name: fieldRegulationMode, Kind: kindString, Enum: assetRegulationModes},
		{Name: fieldSerialNumber, Kind: kindString, MaxLen: 255, Nullable: true},
		{Name: fieldTagIDs, Kind: kindStringList},
		{Name: keyType, Kind: kindString, Enum: assetTypeValues},
		{Name: keyVIN, Kind: kindString, MinLen: 1, MaxLen: 64, Nullable: true, Check: validateVIN},
		{
			Name:     fieldYear,
			Kind:     kindInt,
			IntRange: &intRange{Min: minAssetYear, Max: maxAssetYear},
			Nullable: mode == bodyPatch,
		},
	}
}

func validateVIN(value any) error {
	vin := strings.TrimSpace(stringOf(value))
	for _, char := range vin {
		if (char < '0' || char > '9') && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') {
			return invalidField(keyVIN, "may only contain letters and digits")
		}
	}
	return nil
}

func applyAssetWrite(tx *storeTx, record Record, selfID string, body Record) error {
	assets := tx.records(ResourceAssets)
	if raw, ok := body[fieldExternalIDs]; ok && raw != nil {
		kindRecord := Record{keyType: assetType(record)}
		if bodyType, isString := body[keyType].(string); isString {
			kindRecord[keyType] = bodyType
		}
		if err := tx.ensureExternalIDs(externalIDClaim{
			Kind:  assetExternalIDKind(kindRecord),
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

	if selfID != "" && assetGatewaySerial(record) != "" {
		for _, field := range assetRestrictedFields {
			raw, ok := body[field]
			if !ok || fmt.Sprint(raw) == fmt.Sprint(record[field]) {
				continue
			}
			return invalidField(field, "is read from the installed gateway and cannot be updated")
		}
	}
	for _, field := range assetWritableFields {
		raw, ok := body[field]
		if !ok {
			continue
		}
		if raw == nil {
			delete(record, field)
			continue
		}
		if field == fieldAttributesKey {
			record[field] = resolveAttributeIDs(listOf(raw), attributeEntityAsset)
			continue
		}
		record[field] = raw
	}
	if stringValue(record, keyType) == "" {
		record[keyType] = assetTypeUncategorized
	}
	return nil
}

func (s *Server) handleAssetCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, assetBodyRules(bodyCreate), bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	assetID, err := s.store.TransactID(func(tx *storeTx) (string, error) {
		return createAssetTx(tx, sanitized, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondAssetWrite(writer, request, assetID, eventVehicleCreated, "|asset-create")
}

func (s *Server) handleAssetPatch(writer http.ResponseWriter, request *http.Request) {
	ref, err := queryID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, assetBodyRules(bodyPatch), bodyPatch)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	assetID, err := s.store.TransactID(func(tx *storeTx) (string, error) {
		return patchAssetTx(tx, ref, sanitized, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondAssetWrite(writer, request, assetID, eventVehicleUpdated, "|asset-patch")
}

func createAssetTx(tx *storeTx, body Record, now time.Time) (string, error) {
	record := Record{}
	if err := applyAssetWrite(tx, record, "", body); err != nil {
		return "", err
	}
	created, err := tx.insert(ResourceAssets, record, now)
	if err != nil {
		return "", err
	}
	assetID := recordID(created)
	tagIDs := stringListValues(body[fieldTagIDs])
	return assetID, setEntityTagsTx(tx, assetTagKind(created), assetID, tagIDs)
}

func patchAssetTx(tx *storeTx, ref string, body Record, now time.Time) (string, error) {
	current, idx := findRecordByRef(tx.records(ResourceAssets), ref, assetAutoExternalIDs)
	if idx < 0 {
		return "", notFound(keyAsset, ref)
	}
	assetID := recordID(current)
	previousKind := assetTagKind(current)
	updated, err := tx.update(ResourceAssets, assetID, func(record Record) error {
		return applyAssetWrite(tx, record, assetID, body)
	}, now)
	if err != nil {
		return "", err
	}
	moveEntityTagKindTx(tx, previousKind, assetTagKind(updated), assetID)
	if raw, ok := body[fieldTagIDs]; ok {
		return assetID, setEntityTagsTx(tx, assetTagKind(updated), assetID, stringListValues(raw))
	}
	return assetID, nil
}

func (s *Server) respondAssetWrite(
	writer http.ResponseWriter,
	request *http.Request,
	assetID string,
	vehicleEvent string,
	signature string,
) {
	view := s.fleetView()
	asset, ok := view.snap.assetByID[assetID]
	if !ok {
		s.writeError(writer, notFound(keyAsset, assetID))
		return
	}
	if assetType(asset) == assetTypeVehicle {
		s.dispatchEvent(
			request,
			vehicleEvent,
			map[string]any{keyVehicle: map[string]any(view.vehicleView(asset))},
		)
	}
	data := view.assetView(asset, assetViewOptions{ExternalIDs: true, Tags: true, Attributes: true})
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+signature,
		map[string]any{keyData: data},
	)
}

func (s *Server) handleAssetDelete(writer http.ResponseWriter, request *http.Request) {
	id, err := queryID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if err = s.deleteAsset(id, ""); err != nil {
		s.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteAsset(id, requiredType string) error {
	view := s.fleetView()
	asset, ok := view.snap.assetByID[strings.TrimSpace(id)]
	if !ok || (requiredType != "" && assetType(asset) != requiredType) {
		return notFound(ternary(requiredType == "", keyAsset, requiredType), id)
	}
	parkedAt := view.dependentParkingSpots(recordID(asset))
	return s.store.Transact(func(tx *storeTx) error {
		if err := tx.remove(ResourceAssets, recordID(asset)); err != nil {
			return notFound(keyAsset, id)
		}
		removeEntityFromTagsTx(tx, assetTagKind(asset), recordID(asset))
		detachDependentsTx(tx, recordID(asset), parkedAt)
		clearStaticVehicleTx(tx, recordID(asset))
		return nil
	})
}

func (v *fleetView) dependentParkingSpots(hostID string) map[string]routeState {
	out := map[string]routeState{}
	for _, asset := range v.snap.assets {
		if firstNonEmptyField(asset, fieldSimCoupledVehicleID, fieldSimMountedOnAssetID) != hostID {
			continue
		}
		track := v.trackFor(recordID(asset), v.now, v.now)
		if state, ok := v.live.trackStateAt(track, v.now); ok {
			out[recordID(asset)] = state
		}
	}
	return out
}

func detachDependentsTx(tx *storeTx, hostID string, parkedAt map[string]routeState) {
	assets := tx.records(ResourceAssets)
	for idx, asset := range assets {
		if firstNonEmptyField(asset, fieldSimCoupledVehicleID, fieldSimMountedOnAssetID) != hostID {
			continue
		}
		next := cloneRecord(asset)
		delete(next, fieldSimCoupledVehicleID)
		delete(next, fieldSimMountedOnAssetID)
		if state, ok := parkedAt[recordID(asset)]; ok {
			next[fieldSimParkedLocation] = map[string]any{
				keyLatitude:       round(state.Latitude, 6),
				keyLongitude:      round(state.Longitude, 6),
				keyHeadingDegrees: round(state.Heading, 1),
			}
		}
		assets[idx] = next
	}
}

func clearStaticVehicleTx(tx *storeTx, vehicleID string) {
	drivers := tx.records(ResourceDrivers)
	for idx, driver := range drivers {
		if stringValue(driver, fieldStaticAssignedVehicle) != vehicleID {
			continue
		}
		next := cloneRecord(driver)
		delete(next, fieldStaticAssignedVehicle)
		delete(next, fieldSimStaticAssignedAt)
		drivers[idx] = next
	}
}

func (s *Server) handleAssetLocationStream(writer http.ResponseWriter, request *http.Request) {
	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	view := s.fleetView()
	assetIDs := idsFromQuery(request.URL.Query(), "ids")
	records := view.assetLocationStream(assetIDs, startTime, endTime)
	page, pagination, err := paginate(records, request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if pagination["endCursor"] == "" {
		delete(pagination, "endCursor")
	}
	payload := map[string]any{keyData: recordsAsAny(page), keyPagination: pagination}
	s.respondJSON(writer, request, requestSignature(request)+"|asset-stream", payload)
}

func (v *fleetView) assetLocationStream(assetIDs []string, startTime, endTime *time.Time) []Record {
	filter := toStringSet(assetIDs)
	windowStart, windowEnd := resolveWindow(v.now, startTime, endTime, defaultAssetLookback)
	times := sampleTimes(windowStart, windowEnd, defaultAssetSampleStep, maxAssetSamplesPerAsset)
	stream := make([]Record, 0, len(v.snap.assets)*len(times))
	for _, asset := range v.snap.assets {
		id := recordID(asset)
		if !matchesStringFilter(filter, id) {
			continue
		}
		track := v.trackFor(id, windowStart, windowEnd)
		if track.Kind == trackNone {
			continue
		}
		for _, sampleTime := range times {
			state, ok := v.live.trackStateAt(track, sampleTime)
			if !ok {
				continue
			}
			location := map[string]any{
				keyLatitude:       round(state.Latitude, 6),
				keyLongitude:      round(state.Longitude, 6),
				keyHeadingDegrees: int64(round(state.Heading, 0)),
			}
			if len(state.Address) > 0 {
				location["address"] = cloneAny(state.Address)
			}
			assetRef := map[string]any{keyID: id}
			if externalIDs := externalIDsOf(asset); len(externalIDs) > 0 {
				assetRef[fieldExternalIDs] = renderExternalIDs(asset, nil)
			}
			record := Record{
				keyAsset:          assetRef,
				keyHappenedAtTime: formatSampleTime(sampleTime),
				keyLocation:       location,
				keySpeed: map[string]any{
					"gpsSpeedMetersPerSecond": round(state.SpeedMPS, 2),
					"ecuSpeedMetersPerSecond": round(state.SpeedMPS*0.985, 2),
				},
			}
			stream = append(stream, record)
		}
	}
	sortStreamRecords(stream)
	return stream
}

func sortStreamRecords(stream []Record) {
	slices.SortStableFunc(stream, func(left, right Record) int {
		if cmp := strings.Compare(
			stringValue(left, keyHappenedAtTime),
			stringValue(right, keyHappenedAtTime),
		); cmp != 0 {
			return cmp
		}
		return strings.Compare(
			nestedString(left, keyAsset, keyID),
			nestedString(right, keyAsset, keyID),
		)
	})
}
