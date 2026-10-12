package sim

import (
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"
)

const (
	maxAddressNameLength      = 255
	maxFormattedAddressLength = 1024
	maxAddressNotesLength     = 280
	minPolygonVertices        = 3
	maxPolygonVertices        = 40
	geocodeMinOffsetMeters    = 400.0
	geocodeOffsetSpanMeters   = 5200.0
	fieldFormattedAddress     = "formattedAddress"
	fieldAddressTypes         = "addressTypes"
	fieldContactIDs           = "contactIds"
	fieldContacts             = "contacts"
	labelAddress              = "address"
)

var addressTypeValues = []string{
	"yard",
	"shortHaul",
	"workforceSite",
	"riskZone",
	"industrialSite",
	"alertsOnly",
	"agricultureSource",
	"avoidanceZone",
	"knownGPSJammingZone",
	"authorizedZone",
	"unauthorizedZone",
	"vendor",
	"inventory",
	"customerSite",
}

func (s *Server) registerAddressRoutes() {
	s.mux.HandleFunc("GET /addresses", s.handleAddressList)
	s.mux.HandleFunc("POST /addresses", s.handleAddressCreate)
	s.mux.HandleFunc("GET /addresses/{id}", s.handleAddressGet)
	s.mux.HandleFunc("PATCH /addresses/{id}", s.handleAddressPatch)
	s.mux.HandleFunc("DELETE /addresses/{id}", s.handleAddressDelete)
	s.registerContactRoutes()
}

func addressBodyRules(mode bodyMode) []fieldRule {
	return []fieldRule{
		{
			Name:     fieldAddressTypes,
			Kind:     kindStringList,
			Nullable: true,
			Check:    enumList(fieldAddressTypes, addressTypeValues),
		},
		{Name: fieldContactIDs, Kind: kindStringList, Nullable: true},
		{Name: fieldExternalIDs, Kind: kindExternalIDs, Nullable: true},
		{
			Name:     fieldFormattedAddress,
			Kind:     kindString,
			Required: mode == bodyCreate,
			MinLen:   1,
			MaxLen:   maxFormattedAddressLength,
			Check:    nonBlank(fieldFormattedAddress),
		},
		{Name: keyLatitude, Kind: kindNumber, Check: coordinateRange(keyLatitude, 90)},
		{Name: keyLongitude, Kind: kindNumber, Check: coordinateRange(keyLongitude, 180)},
		{
			Name:     keyName,
			Kind:     kindString,
			Required: mode == bodyCreate,
			MinLen:   1,
			MaxLen:   maxAddressNameLength,
			Check:    nonBlank(keyName),
		},
		{Name: keyNotes, Kind: kindString, MaxLen: maxAddressNotesLength, Nullable: true},
		{Name: fieldTagIDs, Kind: kindStringList, Nullable: true},
	}
}

func enumList(field string, allowed []string) func(value any) error {
	return func(value any) error {
		for _, item := range listOf(value) {
			if !slices.Contains(allowed, stringOf(item)) {
				return invalidField(
					field,
					"must only contain "+strings.Join(quoteAll(allowed), ", "),
				)
			}
		}
		return nil
	}
}

func coordinateRange(field string, limit float64) func(value any) error {
	return func(value any) error {
		number, ok := value.(float64)
		if !ok || number < -limit || number > limit {
			return invalidField(field, fmt.Sprintf("must be between %g and %g", -limit, limit))
		}
		return nil
	}
}

func sanitizeAddressBody(body Record, mode bodyMode) (Record, error) {
	sanitized, err := sanitizeBody(body, addressBodyRules(mode), mode)
	if err != nil {
		return nil, err
	}
	raw, present := body[fieldGeofence]
	if !present {
		if mode == bodyCreate {
			return nil, invalidField(fieldGeofence, "is required")
		}
		return sanitized, nil
	}
	geofence, err := sanitizeGeofence(raw)
	if err != nil {
		return nil, err
	}
	sanitized[fieldGeofence] = geofence
	return sanitized, nil
}

func sanitizeGeofence(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, invalidField(fieldGeofence, "cannot be null")
	}
	geofence, ok := anyAsMap(raw)
	if !ok {
		return nil, invalidField(fieldGeofence, "must be an object")
	}
	rawCircle, hasCircle := geofence[fieldCircle]
	rawPolygon, hasPolygon := geofence[fieldPolygon]
	hasCircle = hasCircle && rawCircle != nil
	hasPolygon = hasPolygon && rawPolygon != nil
	if hasCircle == hasPolygon {
		return nil, invalidField(fieldGeofence, "must define exactly one of circle or polygon")
	}
	out := make(map[string]any, 2)
	if hasCircle {
		circle, err := sanitizeGeofenceCircle(rawCircle)
		if err != nil {
			return nil, err
		}
		out[fieldCircle] = circle
	} else {
		polygon, err := sanitizeGeofencePolygon(rawPolygon)
		if err != nil {
			return nil, err
		}
		out[fieldPolygon] = polygon
	}
	if rawSettings, ok := geofence[fieldSettings]; ok && rawSettings != nil {
		settings, isMap := anyAsMap(rawSettings)
		if !isMap {
			return nil, invalidField("geofence.settings", "must be an object")
		}
		sanitized := make(map[string]any, 1)
		if rawShow, has := settings[fieldShowAddresses]; has {
			show, isBool := rawShow.(bool)
			if !isBool {
				return nil, invalidField("geofence.settings.showAddresses", "must be a boolean")
			}
			sanitized[fieldShowAddresses] = show
		}
		out[fieldSettings] = sanitized
	}
	return out, nil
}

func sanitizeGeofenceCircle(raw any) (map[string]any, error) {
	circle, ok := anyAsMap(raw)
	if !ok {
		return nil, invalidField("geofence.circle", "must be an object")
	}
	rawRadius, present := circle[fieldRadiusMeters]
	if !present || rawRadius == nil {
		return nil, invalidField("geofence.circle.radiusMeters", "is required")
	}
	radius, err := sanitizeInt(rawRadius, &fieldRule{
		IntRange: &intRange{Min: 1, Max: math.MaxInt32},
	}, "geofence.circle.radiusMeters")
	if err != nil {
		return nil, err
	}
	out := map[string]any{fieldRadiusMeters: radius}
	rawLatitude, hasLatitude := circle[keyLatitude]
	rawLongitude, hasLongitude := circle[keyLongitude]
	hasLatitude = hasLatitude && rawLatitude != nil
	hasLongitude = hasLongitude && rawLongitude != nil
	if hasLatitude != hasLongitude {
		return nil, invalidField(
			"geofence.circle",
			"latitude and longitude must be provided together",
		)
	}
	if hasLatitude {
		latitude, err := sanitizeCoordinate(rawLatitude, "geofence.circle.latitude", 90)
		if err != nil {
			return nil, err
		}
		longitude, err := sanitizeCoordinate(rawLongitude, "geofence.circle.longitude", 180)
		if err != nil {
			return nil, err
		}
		out[keyLatitude] = latitude
		out[keyLongitude] = longitude
	}
	return out, nil
}

func sanitizeCoordinate(raw any, path string, limit float64) (float64, error) {
	value, err := sanitizeNumber(raw, path)
	if err != nil {
		return 0, err
	}
	if value < -limit || value > limit {
		return 0, invalidField(path, fmt.Sprintf("must be between %g and %g", -limit, limit))
	}
	return value, nil
}

func sanitizeGeofencePolygon(raw any) (map[string]any, error) {
	polygon, ok := anyAsMap(raw)
	if !ok {
		return nil, invalidField("geofence.polygon", "must be an object")
	}
	rawVertices, present := polygon[fieldVertices]
	if !present || rawVertices == nil {
		return nil, invalidField("geofence.polygon.vertices", "is required")
	}
	items, ok := rawVertices.([]any)
	if !ok {
		return nil, invalidField("geofence.polygon.vertices", "must be an array of vertices")
	}
	vertices := make([]geofenceVertex, 0, len(items))
	for idx, item := range items {
		path := fmt.Sprintf("geofence.polygon.vertices[%d]", idx)
		vertex, isMap := anyAsMap(item)
		if !isMap {
			return nil, invalidField(path, "must be an object")
		}
		for _, field := range []string{keyLatitude, keyLongitude} {
			if value, has := vertex[field]; !has || value == nil {
				return nil, invalidField(path+"."+field, "is required")
			}
		}
		latitude, err := sanitizeCoordinate(vertex[keyLatitude], path+".latitude", 90)
		if err != nil {
			return nil, err
		}
		longitude, err := sanitizeCoordinate(vertex[keyLongitude], path+".longitude", 180)
		if err != nil {
			return nil, err
		}
		vertices = append(vertices, geofenceVertex{Latitude: latitude, Longitude: longitude})
	}
	if len(vertices) > minPolygonVertices && vertices[0] == vertices[len(vertices)-1] {
		vertices = vertices[:len(vertices)-1]
	}
	if len(vertices) < minPolygonVertices || len(vertices) > maxPolygonVertices {
		return nil, invalidField(
			"geofence.polygon.vertices",
			fmt.Sprintf("must contain %d to %d vertices", minPolygonVertices, maxPolygonVertices),
		)
	}
	if err := validatePolygonShape(vertices); err != nil {
		return nil, err
	}
	out := make([]any, 0, len(vertices))
	for _, vertex := range vertices {
		out = append(out, map[string]any{
			keyLatitude:  vertex.Latitude,
			keyLongitude: vertex.Longitude,
		})
	}
	return map[string]any{fieldVertices: out}, nil
}

func validatePolygonShape(vertices []geofenceVertex) error {
	seen := make(map[geofenceVertex]struct{}, len(vertices))
	for _, vertex := range vertices {
		if _, dup := seen[vertex]; dup {
			return invalidField("geofence.polygon.vertices", "must not repeat a vertex")
		}
		seen[vertex] = struct{}{}
	}
	count := len(vertices)
	for first := 0; first < count; first++ {
		a1, a2 := vertices[first], vertices[(first+1)%count]
		for second := first + 1; second < count; second++ {
			if second == first+1 || (first == 0 && second == count-1) {
				continue
			}
			b1, b2 := vertices[second], vertices[(second+1)%count]
			if segmentsIntersect(a1, a2, b1, b2) {
				return invalidField("geofence.polygon.vertices", "edges must not cross each other")
			}
		}
	}
	area := 0.0
	for idx := range vertices {
		next := vertices[(idx+1)%count]
		area += vertices[idx].Longitude*next.Latitude - next.Longitude*vertices[idx].Latitude
	}
	if math.Abs(area) < 1e-12 {
		return invalidField("geofence.polygon.vertices", "must enclose an area")
	}
	return nil
}

func segmentsIntersect(a1, a2, b1, b2 geofenceVertex) bool {
	d1 := orientation(b1, b2, a1)
	d2 := orientation(b1, b2, a2)
	d3 := orientation(a1, a2, b1)
	d4 := orientation(a1, a2, b2)
	if ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) && ((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0)) {
		return true
	}
	return (d1 == 0 && onSegment(b1, b2, a1)) || (d2 == 0 && onSegment(b1, b2, a2)) ||
		(d3 == 0 && onSegment(a1, a2, b1)) || (d4 == 0 && onSegment(a1, a2, b2))
}

func orientation(a, b, c geofenceVertex) float64 {
	return (b.Longitude-a.Longitude)*(c.Latitude-a.Latitude) -
		(b.Latitude-a.Latitude)*(c.Longitude-a.Longitude)
}

func onSegment(a, b, point geofenceVertex) bool {
	return point.Longitude >= math.Min(a.Longitude, b.Longitude) &&
		point.Longitude <= math.Max(a.Longitude, b.Longitude) &&
		point.Latitude >= math.Min(a.Latitude, b.Latitude) &&
		point.Latitude <= math.Max(a.Latitude, b.Latitude)
}

func geocodeFormattedAddress(
	formatted string,
	addresses []Record,
	selfID string,
) (geofenceVertex, bool) {
	clean := strings.TrimSpace(formatted)
	for _, address := range addresses {
		if recordID(address) == selfID ||
			!strings.EqualFold(stringValue(address, fieldFormattedAddress), clean) {
			continue
		}
		latitude, hasLatitude := address[keyLatitude].(float64)
		longitude, hasLongitude := address[keyLongitude].(float64)
		if hasLatitude && hasLongitude {
			return geofenceVertex{Latitude: latitude, Longitude: longitude}, true
		}
	}
	city, ok := gazetteerCityIn(clean)
	if !ok {
		return geofenceVertex{}, false
	}
	key := strings.ToLower(clean)
	bearing := fractionFromHash("geocode-bearing|"+key) * 2 * math.Pi
	distance := geocodeMinOffsetMeters + fractionFromHash(
		"geocode-distance|"+key,
	)*geocodeOffsetSpanMeters
	latitude := city.Latitude + distance*math.Cos(bearing)/metersPerDegreeLatitude
	longitude := city.Longitude + distance*math.Sin(bearing)/metersPerDegreeLongitude(city.Latitude)
	return geofenceVertex{Latitude: round(latitude, 6), Longitude: round(longitude, 6)}, true
}

func gazetteerCityIn(formatted string) (gazetteerCity, bool) {
	parts := strings.Split(formatted, ",")
	for idx := len(parts) - 1; idx >= 0; idx-- {
		part := strings.TrimSpace(parts[idx])
		for cityIdx := range texasGazetteer {
			if strings.EqualFold(part, texasGazetteer[cityIdx].Name) {
				return texasGazetteer[cityIdx], true
			}
		}
	}
	return gazetteerCity{}, false
}

func applyAddressWrite(tx *storeTx, record Record, selfID string, body Record) error {
	if raw, ok := body[fieldExternalIDs]; ok && raw != nil {
		if err := tx.ensureExternalIDs(externalIDClaim{
			Kind:  externalIDKindAddresses,
			Owner: externalIDOwner(ResourceAddresses, selfID),
			IDs:   mapOf(raw),
		}); err != nil {
			return err
		}
	}
	if raw, ok := body[fieldContactIDs]; ok && raw != nil {
		exists := recordExists(tx.records(ResourceContacts))
		for _, contactID := range stringListValues(raw) {
			if !exists(contactID) {
				return missingReference("contact", contactID)
			}
		}
	}
	formattedChanged := false
	for _, field := range []string{
		keyName,
		fieldFormattedAddress,
		keyNotes,
		fieldAddressTypes,
		fieldContactIDs,
		fieldExternalIDs,
		keyLatitude,
		keyLongitude,
		fieldGeofence,
	} {
		raw, ok := body[field]
		if !ok {
			continue
		}
		if field == fieldFormattedAddress && stringOf(raw) != stringValue(record, field) {
			formattedChanged = true
		}
		if raw == nil || isEmptyList(raw) {
			delete(record, field)
			continue
		}
		record[field] = cloneAny(raw)
	}
	return resolveAddressCoordinates(tx, record, selfID, body, formattedChanged)
}

func isEmptyList(raw any) bool {
	items, ok := raw.([]any)
	return ok && len(items) == 0
}

func resolveAddressCoordinates(
	tx *storeTx,
	record Record,
	selfID string,
	body Record,
	formattedChanged bool,
) error {
	_, latitudeSent := body[keyLatitude]
	_, longitudeSent := body[keyLongitude]
	if latitudeSent != longitudeSent {
		return invalidField(keyLatitude, "and longitude must be provided together")
	}
	addresses := tx.records(ResourceAddresses)
	if !latitudeSent && (formattedChanged || !hasCoordinates(record)) {
		if point, ok := geocodeFormattedAddress(
			stringValue(record, fieldFormattedAddress),
			addresses,
			selfID,
		); ok {
			record[keyLatitude] = point.Latitude
			record[keyLongitude] = point.Longitude
		}
	}
	geofence := mapOf(record[fieldGeofence])
	if geofence == nil {
		return invalidField(fieldGeofence, "is required")
	}
	if circle := mapOf(geofence[fieldCircle]); circle != nil {
		if _, ok := circle[keyLatitude]; !ok {
			if !hasCoordinates(record) {
				return invalidField(
					fieldFormattedAddress,
					"could not be geocoded; provide geofence.circle latitude and longitude",
				)
			}
			circle[keyLatitude] = record[keyLatitude]
			circle[keyLongitude] = record[keyLongitude]
		}
		if !hasCoordinates(record) {
			record[keyLatitude] = circle[keyLatitude]
			record[keyLongitude] = circle[keyLongitude]
		}
		return nil
	}
	vertices := geofenceVerticesOf(mapOf(geofence[fieldPolygon])[fieldVertices])
	latitude, _ := record[keyLatitude].(float64)
	longitude, _ := record[keyLongitude].(float64)
	_, geofenceSent := body[fieldGeofence]
	relocated := !latitudeSent && (formattedChanged || geofenceSent)
	if !hasCoordinates(record) ||
		(relocated && !polygonContains(vertices, latitude, longitude)) {
		center := polygonCentroid(vertices)
		record[keyLatitude] = round(center.Latitude, 6)
		record[keyLongitude] = round(center.Longitude, 6)
	}
	return nil
}

func hasCoordinates(record Record) bool {
	_, hasLatitude := record[keyLatitude].(float64)
	_, hasLongitude := record[keyLongitude].(float64)
	return hasLatitude && hasLongitude
}

func createAddressTx(tx *storeTx, body Record, now time.Time) (string, error) {
	record := Record{}
	if err := applyAddressWrite(tx, record, "", body); err != nil {
		return "", err
	}
	created, err := tx.insert(ResourceAddresses, record, now)
	if err != nil {
		return "", err
	}
	addressID := recordID(created)
	return addressID, setEntityTagsTx(
		tx,
		tagMembersAddresses,
		addressID,
		stringListValues(body[fieldTagIDs]),
	)
}

func patchAddressTx(tx *storeTx, ref string, body Record, now time.Time) (string, error) {
	current, idx := findRecordByRef(tx.records(ResourceAddresses), ref, nil)
	if idx < 0 {
		return "", notFound(labelAddress, ref)
	}
	addressID := recordID(current)
	_, err := tx.update(ResourceAddresses, addressID, func(record Record) error {
		return applyAddressWrite(tx, record, addressID, body)
	}, now)
	if err != nil {
		return "", err
	}
	if raw, ok := body[fieldTagIDs]; ok {
		return addressID, setEntityTagsTx(
			tx,
			tagMembersAddresses,
			addressID,
			stringListValues(raw),
		)
	}
	return addressID, nil
}

func deleteAddressTx(tx *storeTx, ref string) (Record, error) {
	current, idx := findRecordByRef(tx.records(ResourceAddresses), ref, nil)
	if idx < 0 {
		return nil, notFound(labelAddress, ref)
	}
	addressID := recordID(current)
	if err := tx.remove(ResourceAddresses, addressID); err != nil {
		return nil, err
	}
	removeEntityFromTagsTx(tx, tagMembersAddresses, addressID)
	return current, nil
}

type addressRenderer struct {
	tags     *tagIndex
	contacts map[string]Record
}

func (s *Server) addressRenderer(view *fleetView) (addressRenderer, error) {
	contacts, err := s.store.List(ResourceContacts)
	if err != nil {
		return addressRenderer{}, err
	}
	return addressRenderer{tags: view.snap.tags, contacts: indexRecordsByID(contacts)}, nil
}

func (r addressRenderer) view(address Record) Record {
	id := recordID(address)
	out := Record{
		keyID:                 id,
		keyName:               stringValue(address, keyName),
		fieldFormattedAddress: stringValue(address, fieldFormattedAddress),
		keyTags:               r.tags.tinyTags(tagMembersAddresses, id),
	}
	if geofence := mapOf(address[fieldGeofence]); geofence != nil {
		out[fieldGeofence] = cloneMap(geofence)
	}
	for _, field := range []string{keyLatitude, keyLongitude} {
		if value, ok := address[field].(float64); ok {
			out[field] = value
		}
	}
	if notes := stringValue(address, keyNotes); notes != "" {
		out[keyNotes] = notes
	}
	if types := stringListValues(address[fieldAddressTypes]); len(types) > 0 {
		out[fieldAddressTypes] = stringsAsAny(types)
	}
	contacts := make([]any, 0, len(listOf(address[fieldContactIDs])))
	for _, contactID := range stringListValues(address[fieldContactIDs]) {
		contact, ok := r.contacts[contactID]
		if !ok {
			continue
		}
		contacts = append(contacts, map[string]any{
			keyID:          contactID,
			fieldFirstName: stringValue(contact, fieldFirstName),
			fieldLastName:  stringValue(contact, fieldLastName),
		})
	}
	out[fieldContacts] = contacts
	if externalIDs := externalIDsOf(address); len(externalIDs) > 0 {
		out[fieldExternalIDs] = renderExternalIDs(address, nil)
	}
	if created := stringValue(address, fieldCreatedAtTime); created != "" {
		out[fieldCreatedAtTime] = created
	}
	return out
}

func stringsAsAny(values []string) []any {
	out := make([]any, len(values))
	for idx, value := range values {
		out[idx] = value
	}
	return out
}

func addressDeletedPayload(address Record) map[string]any {
	return map[string]any{
		labelAddress: map[string]any{
			keyID:            recordID(address),
			keyName:          stringValue(address, keyName),
			fieldExternalIDs: renderExternalIDs(address, nil),
		},
	}
}

func (s *Server) handleAddressList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	createdAfter, err := parseOptionalTimePtr(values, "createdAfterTime")
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	renderer, err := s.addressRenderer(view)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	tagIDs, parentTagIDs := standardTagParams(values)
	filter := view.snap.tags.filter(tagIDs, parentTagIDs)
	records := make([]Record, 0, len(view.snap.addresses))
	for _, address := range view.snap.addresses {
		if !filter.matches(view.snap.tags, tagMembersAddresses, recordID(address)) ||
			!recordAtOrAfter(address, fieldCreatedAtTime, createdAfter) {
			continue
		}
		records = append(records, renderer.view(address))
	}
	s.respondPage(writer, request, records, "|address-list")
}

func (s *Server) handleAddressGet(writer http.ResponseWriter, request *http.Request) {
	ref, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	address, idx := findRecordByRef(view.snap.addresses, ref, nil)
	if idx < 0 {
		s.writeError(writer, notFound(labelAddress, ref))
		return
	}
	s.respondAddress(writer, request, view, recordID(address), "|address-get", "")
}

func (s *Server) handleAddressCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeAddressBody(body, bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	addressID, err := s.store.TransactID(func(tx *storeTx) (string, error) {
		return createAddressTx(tx, sanitized, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondAddress(
		writer,
		request,
		s.fleetView(),
		addressID,
		"|address-create",
		webhookEventAddressCreated,
	)
}

func (s *Server) handleAddressPatch(writer http.ResponseWriter, request *http.Request) {
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
	sanitized, err := sanitizeAddressBody(body, bodyPatch)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	addressID, err := s.store.TransactID(func(tx *storeTx) (string, error) {
		return patchAddressTx(tx, ref, sanitized, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondAddress(
		writer,
		request,
		s.fleetView(),
		addressID,
		"|address-patch",
		webhookEventAddressUpdated,
	)
}

func (s *Server) handleAddressDelete(writer http.ResponseWriter, request *http.Request) {
	ref, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	var deleted Record
	err = s.store.Transact(func(tx *storeTx) error {
		var deleteErr error
		deleted, deleteErr = deleteAddressTx(tx, ref)
		return deleteErr
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.dispatchEvent(request, webhookEventAddressDeleted, addressDeletedPayload(deleted))
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) respondAddress(
	writer http.ResponseWriter,
	request *http.Request,
	view *fleetView,
	addressID string,
	signature string,
	eventType string,
) {
	address, ok := view.snap.addressByID[addressID]
	if !ok {
		s.writeError(writer, notFound(labelAddress, addressID))
		return
	}
	renderer, err := s.addressRenderer(view)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	rendered := renderer.view(address)
	if eventType != "" {
		s.dispatchEvent(request, eventType, map[string]any{labelAddress: cloneRecord(rendered)})
	}
	payload := map[string]any{keyData: rendered}
	s.respondJSON(writer, request, requestSignature(request)+signature, payload)
}
