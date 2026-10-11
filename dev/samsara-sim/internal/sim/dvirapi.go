package sim

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const (
	paramSafetyStatus       = "safetyStatus"
	paramIncludeExternalIDs = "includeExternalIds"
	dvirStreamCursorKind    = "dvir-stream"
	dvirStreamCursorVersion = 1
	maxDvirLicensePlate     = 12
	fieldAuthorID           = "authorId"
	fieldResolvedDefectIDs  = "resolvedDefectIds"
	fieldVehicleIDBody      = "vehicleId"
	fieldTrailerIDBody      = "trailerId"
	fieldWalkaroundPhotos   = "walkaroundPhotos"
	dvirSimKindAPI          = "api"
	dvirResolutionIDPrefix  = "dvir:"
	defectResolutionPrefix  = "defect:"
)

var (
	dvirStreamSafetyStatuses = []string{
		dvirSafetyStatusSafe,
		dvirSafetyStatusUnsafe,
		dvirSafetyStatusResolved,
	}
	dvirCreateSafetyStatuses = []string{dvirSafetyStatusSafe, dvirSafetyStatusUnsafe}
)

type dvirStreamCursor struct {
	Version   int    `json:"v"`
	Kind      string `json:"k"`
	UpdatedAt string `json:"u"`
	ID        string `json:"i"`
}

func encodeDvirStreamCursor(updatedAt, id string) (string, error) {
	raw, err := sonic.Marshal(dvirStreamCursor{
		Version:   dvirStreamCursorVersion,
		Kind:      dvirStreamCursorKind,
		UpdatedAt: updatedAt,
		ID:        id,
	})
	if err != nil {
		return "", fmt.Errorf("encode DVIR stream cursor: %w", err)
	}
	return base64.URLEncoding.EncodeToString(raw), nil
}

func decodeDvirStreamCursor(value string) (dvirStreamCursor, error) {
	raw, err := base64.URLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return dvirStreamCursor{}, ErrCursorInvalid
	}
	cursor := dvirStreamCursor{}
	if err = sonic.Unmarshal(raw, &cursor); err != nil {
		return dvirStreamCursor{}, ErrCursorInvalid
	}
	if cursor.Version != dvirStreamCursorVersion || cursor.Kind != dvirStreamCursorKind {
		return dvirStreamCursor{}, ErrCursorInvalid
	}
	if _, err = parseRFC3339(cursor.UpdatedAt); err != nil {
		return dvirStreamCursor{}, ErrCursorInvalid
	}
	return cursor, nil
}

func (c dvirStreamCursor) before(record Record) bool {
	updated := stringValue(record, fieldUpdatedAtTime)
	if updated != c.UpdatedAt {
		return updated > c.UpdatedAt
	}
	return recordID(record) > c.ID
}

type dvirStreamQuery struct {
	Start            time.Time
	End              time.Time
	HasEnd           bool
	Statuses         map[string]struct{}
	IncludeExternals bool
	Limit            int
	Cursor           *dvirStreamCursor
}

func parseDvirStreamQuery(request *http.Request, now time.Time) (dvirStreamQuery, error) {
	values := request.URL.Query()
	query := dvirStreamQuery{}
	start, hasStart, err := parseOptionalTime(values, fieldStartTime)
	if err != nil {
		return query, err
	}
	if !hasStart {
		return query, invalidParameter(fieldStartTime, "is required")
	}
	end, hasEnd, err := parseOptionalTime(values, fieldEndTime)
	if err != nil {
		return query, err
	}
	if hasEnd && end.Before(start) {
		return query, invalidParameter(fieldEndTime, "must not be before startTime")
	}
	query.Start = start
	query.HasEnd = hasEnd
	query.End = now
	if hasEnd && end.Before(now) {
		query.End = end
	}
	statuses := csvQueryValues(values, paramSafetyStatus)
	for _, status := range statuses {
		if !slices.Contains(dvirStreamSafetyStatuses, status) {
			return query, invalidParameter(paramSafetyStatus, "must be `safe`, `unsafe` or `resolved`")
		}
	}
	query.Statuses = toStringSet(statuses)
	if query.IncludeExternals, err = parseBoolParam(values, paramIncludeExternalIDs); err != nil {
		return query, err
	}
	if query.Limit, err = pagePolicyFor(request).parseLimit(values); err != nil {
		return query, err
	}
	if after := strings.TrimSpace(values.Get("after")); after != "" {
		cursor, decodeErr := decodeDvirStreamCursor(after)
		if decodeErr != nil {
			return query, decodeErr
		}
		query.Cursor = &cursor
	}
	return query, nil
}

func (s *Server) handleDvirStream(writer http.ResponseWriter, request *http.Request) {
	view := s.fleetView()
	query, err := parseDvirStreamQuery(request, view.now)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	ctx := s.live.newDvirContext(view.now)
	fromDay := query.Start.Add(-dvirUpdateLag)
	for _, overlay := range ctx.Overlays {
		if day, parseErr := time.Parse(dailyLogDateLayout, stringValue(overlay, fieldSimDay)); parseErr == nil {
			fromDay = minTime(fromDay, day)
		}
	}
	startStamp := query.Start.Format(time.RFC3339)
	endStamp := query.End.Format(time.RFC3339)
	selected := make([]Record, 0, 64)
	for _, record := range s.live.dvirCores(ctx, fromDay, query.End) {
		updated := stringValue(record, fieldUpdatedAtTime)
		if updated < startStamp || updated > endStamp {
			continue
		}
		if !matchesStringFilter(query.Statuses, stringValue(record, fieldSafetyStatus)) {
			continue
		}
		if query.Cursor != nil && !query.Cursor.before(record) {
			continue
		}
		selected = append(selected, record)
	}
	sortDvirs(selected, fieldUpdatedAtTime)

	page := selected[:min(query.Limit, len(selected))]
	hasNextPage := len(selected) > len(page)
	endCursor := ""
	switch {
	case len(page) > 0 && (hasNextPage || !query.HasEnd):
		last := page[len(page)-1]
		endCursor, err = encodeDvirStreamCursor(stringValue(last, fieldUpdatedAtTime), recordID(last))
	case !query.HasEnd && query.Cursor != nil:
		endCursor, err = encodeDvirStreamCursor(query.Cursor.UpdatedAt, query.Cursor.ID)
	case !query.HasEnd:
		endCursor, err = encodeDvirStreamCursor(startStamp, "")
	}
	if err != nil {
		s.writeError(writer, err)
		return
	}

	data := make([]any, 0, len(page))
	for _, record := range page {
		s.live.decorateDvir(ctx, record)
		data = append(data, renderDvirStream(view.snap, record, query.IncludeExternals))
	}
	payload := map[string]any{
		keyData: data,
		keyPagination: map[string]any{
			"endCursor":   endCursor,
			"hasNextPage": hasNextPage,
		},
	}
	s.respondJSON(writer, request, requestSignature(request)+"|dvir-stream", payload)
}

func (s *Server) handleDvirGet(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	includeExternals, err := parseBoolParam(request.URL.Query(), paramIncludeExternalIDs)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	record, ok := s.live.dvirByID(s.live.newDvirContext(view.now), id)
	if !ok {
		s.writeError(writer, notFound("DVIR", id))
		return
	}
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+"|dvir-get",
		renderDvirStream(view.snap, record, includeExternals),
	)
}

func renderDvirStream(snapshot *fleetSnapshot, record Record, includeExternals bool) map[string]any {
	out := map[string]any{
		keyID:                     recordID(record),
		keyType:                   stringValue(record, keyType),
		fieldSafetyStatus:         stringValue(record, fieldSafetyStatus),
		"dvirSubmissionBeginTime": stringValue(record, fieldStartTime),
		"dvirSubmissionTime":      stringValue(record, fieldEndTime),
		fieldUpdatedAtTime:        stringValue(record, fieldUpdatedAtTime),
		fieldAuthorSignature: streamSignature(
			snapshot,
			record[fieldAuthorSignature],
			includeExternals,
		),
	}
	for _, field := range []string{fieldSecondSignature, fieldThirdSignature} {
		if signature, ok := record[field]; ok && signature != nil {
			out[field] = streamSignature(snapshot, signature, includeExternals)
		}
	}
	defectIDs := make([]any, 0, 2)
	for _, raw := range append(listOf(record[fieldVehicleDefects]), listOf(record[fieldTrailerDefects])...) {
		defectIDs = append(defectIDs, stringOf(mapOf(raw)[keyID]))
	}
	out["defectIds"] = defectIDs
	if location := stringValue(record, keyLocation); location != "" {
		out["formattedAddress"] = location
	}
	if notes := stringValue(record, fieldMechanicNotes); notes != "" {
		out[fieldMechanicNotes] = notes
	}
	if odometer, ok := record[keyOdometerMeters]; ok && odometer != nil {
		out[keyOdometerMeters] = odometer
	}
	if vehicleID := stringValue(record, fieldSimVehicleID); vehicleID != "" {
		out[keyVehicle] = streamAssetRef(snapshot, vehicleID, includeExternals)
	}
	if trailerID := stringValue(record, fieldSimTrailerID); trailerID != "" {
		out["trailer"] = streamAssetRef(snapshot, trailerID, includeExternals)
	}
	if photos := listOf(record[fieldWalkaroundPhotos]); len(photos) > 0 {
		out[fieldWalkaroundPhotos] = cloneAny(photos)
	}
	return out
}

func streamAssetRef(snapshot *fleetSnapshot, assetID string, includeExternals bool) map[string]any {
	out := map[string]any{keyID: assetID}
	if includeExternals {
		out[fieldExternalIDs] = renderExternalIDs(snapshot.assetByID[assetID], vehicleAutoExternalIDs)
	}
	return out
}

func streamSignature(snapshot *fleetSnapshot, raw any, includeExternals bool) map[string]any {
	signature := mapOf(raw)
	userID := stringOf(mapOf(signature[fieldSignatoryUser])[keyID])
	user := map[string]any{keyID: userID}
	if includeExternals && stringOf(signature[keyType]) == dvirSignatureTypeDriver {
		if driver, ok := snapshot.driverByID[userID]; ok && len(externalIDsOf(driver)) > 0 {
			user[fieldExternalIDs] = renderExternalIDs(driver, nil)
		}
	}
	return map[string]any{
		fieldSignatoryUser: user,
		fieldSignedAtTime:  stringOf(signature[fieldSignedAtTime]),
		keyType:            stringOf(signature[keyType]),
	}
}

func dvirCreateRules() []fieldRule {
	return []fieldRule{
		{Name: fieldAuthorID, Kind: kindString, Required: true, MinLen: 1, Check: nonBlank(fieldAuthorID)},
		{Name: keyLicensePlate, Kind: kindString, MaxLen: maxDvirLicensePlate},
		{Name: keyLocation, Kind: kindString},
		{Name: fieldMechanicNotes, Kind: kindString},
		{
			Name:     keyOdometerMeters,
			Kind:     kindInt,
			IntRange: &intRange{Min: 0, Max: 1 << 53},
		},
		{Name: fieldResolvedDefectIDs, Kind: kindStringList},
		{
			Name:     fieldSafetyStatus,
			Kind:     kindString,
			Required: true,
			Enum:     dvirCreateSafetyStatuses,
		},
		{Name: fieldTrailerIDBody, Kind: kindString, MinLen: 1},
		{Name: keyType, Kind: kindString, Required: true, Enum: []string{dvirTypeMechanic}},
		{Name: fieldVehicleIDBody, Kind: kindString, MinLen: 1},
	}
}

func dvirResolveRules() []fieldRule {
	return []fieldRule{
		{Name: fieldAuthorID, Kind: kindString, Required: true, MinLen: 1, Check: nonBlank(fieldAuthorID)},
		{
			Name:     fieldIsResolved,
			Kind:     kindBool,
			Required: true,
			Check: func(value any) error {
				if value != true {
					return invalidField(fieldIsResolved, "must be true")
				}
				return nil
			},
		},
		{Name: fieldMechanicNotes, Kind: kindString},
		{Name: fieldSignedAtTime, Kind: kindTime},
	}
}

func dvirAuthor(ctx *dvirContext, authorID string) (map[string]any, error) {
	user, ok := ctx.Users[strings.TrimSpace(authorID)]
	if !ok {
		return nil, invalidField(fieldAuthorID, "must be the ID of a user in the organization")
	}
	return map[string]any{
		keyID:   recordID(user),
		keyName: stringValue(user, keyName),
		keyType: dvirSignatureTypeMechanic,
	}, nil
}

func (s *Server) handleDvirCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, dvirCreateRules(), bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow().Truncate(time.Second)
	ctx := s.live.newDvirContext(now)
	record, overlays, err := s.live.buildMechanicDvir(ctx, sanitized, now)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	dvirID, err := s.store.TransactID(func(tx *storeTx) (string, error) {
		created, insertErr := tx.insert(ResourceDvirs, record, now)
		if insertErr != nil {
			return "", insertErr
		}
		return recordID(created), tx.upsertDvirOverlays(overlays, recordID(created))
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	created, ok := s.live.dvirByID(s.live.newDvirContext(now), dvirID)
	if !ok {
		s.writeError(writer, notFound("DVIR", dvirID))
		return
	}
	s.dispatchEventOnce(
		request,
		"DvirSubmitted|"+dvirID,
		"DvirSubmitted",
		dvirWebhookData(view.snap, created),
	)
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+"|dvir-create",
		map[string]any{keyData: renderDvirHistory(view, created)},
	)
}

func (l *LiveSimulator) buildMechanicDvir(
	ctx *dvirContext,
	body Record,
	now time.Time,
) (Record, []Record, error) {
	author, err := dvirAuthor(ctx, stringValue(body, fieldAuthorID))
	if err != nil {
		return nil, nil, err
	}
	vehicleID := stringValue(body, fieldVehicleIDBody)
	trailerID := stringValue(body, fieldTrailerIDBody)
	if vehicleID == "" && trailerID == "" {
		return nil, nil, invalidField(fieldVehicleIDBody, "or trailerId must be provided")
	}
	if vehicleID != "" && assetType(ctx.Snapshot.assetByID[vehicleID]) != assetTypeVehicle {
		return nil, nil, missingReference("vehicle", vehicleID)
	}
	if trailerID != "" && assetType(ctx.Snapshot.assetByID[trailerID]) != assetTypeTrailer {
		return nil, nil, missingReference("trailer", trailerID)
	}

	stamp := now.UTC().Format(time.RFC3339)
	record := Record{
		keyType:              dvirTypeMechanic,
		fieldSafetyStatus:    stringValue(body, fieldSafetyStatus),
		fieldStartTime:       stamp,
		fieldEndTime:         stamp,
		fieldSimKind:         dvirSimKindAPI,
		fieldAuthorSignature: dvirSignature(stringOf(author[keyID]), stringOf(author[keyName]), now, dvirSignatureTypeMechanic),
		fieldVehicleDefects:  []any{},
		fieldTrailerDefects:  []any{},
	}
	if vehicleID != "" {
		record[fieldSimVehicleID] = vehicleID
		record[keyOdometerMeters] = l.vehicleOdometerMeters(ctx.Snapshot, vehicleID, now)
		if plate := stringValue(ctx.Snapshot.assetByID[vehicleID], keyLicensePlate); plate != "" {
			record[keyLicensePlate] = plate
		}
	}
	if trailerID != "" {
		record[fieldSimTrailerID] = trailerID
	}
	for _, field := range []string{keyLicensePlate, keyLocation, fieldMechanicNotes} {
		if value, ok := body[field].(string); ok && strings.TrimSpace(value) != "" {
			record[field] = value
		}
	}
	if odometer, ok := body[keyOdometerMeters]; ok {
		record[keyOdometerMeters] = odometer
	}

	defectIDs := stringListValues(body[fieldResolvedDefectIDs])
	overlays := make([]Record, 0, len(defectIDs))
	for _, defectID := range defectIDs {
		parent, defect, found := l.findDvirDefect(ctx, defectID)
		if !found {
			return nil, nil, missingReference("DVIR defect", defectID)
		}
		assetID := stringOf(defect[fieldSimAssetID])
		if assetID != vehicleID && assetID != trailerID {
			return nil, nil, invalidField(
				fieldResolvedDefectIDs,
				fmt.Sprintf("contains defect %s, which belongs to a different vehicle or trailer", defectID),
			)
		}
		if defect[fieldIsResolved] == true {
			return nil, nil, invalidField(
				fieldResolvedDefectIDs,
				fmt.Sprintf("contains defect %s, which is already resolved", defectID),
			)
		}
		overlays = append(overlays, Record{
			keyID:              defectResolutionPrefix + defectID,
			fieldSimKind:       dvirResolutionKindDefect,
			"defectId":         defectID,
			"dvirId":           recordID(parent),
			fieldSimDriverID:   stringValue(parent, fieldSimDriverID),
			fieldSimDay:        stringValue(parent, fieldSimDay),
			fieldSimType:       stringValue(parent, keyType),
			fieldSignedAtTime:  stamp,
			fieldUpdatedAtTime: stamp,
			fieldResolvedBy:    author,
			fieldMechanicNotes: stringValue(body, fieldMechanicNotes),
		})
	}
	return record, overlays, nil
}

func (tx *storeTx) upsertDvirOverlays(overlays []Record, resolvingDvirID string) error {
	if len(overlays) == 0 {
		return nil
	}
	list := tx.records(ResourceDvirResolutions)
	next := make([]Record, 0, len(list)+len(overlays))
	replaced := make(map[string]struct{}, len(overlays))
	for _, overlay := range overlays {
		replaced[recordID(overlay)] = struct{}{}
	}
	for _, existing := range list {
		if _, ok := replaced[recordID(existing)]; !ok {
			next = append(next, existing)
		}
	}
	for _, overlay := range overlays {
		stored := cloneRecord(overlay)
		if resolvingDvirID != "" {
			stored["resolvingDvirId"] = resolvingDvirID
		}
		next = append(next, stored)
	}
	return tx.replace(ResourceDvirResolutions, next)
}

func (s *Server) handleDvirResolve(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, dvirResolveRules(), bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow().Truncate(time.Second)
	ctx := s.live.newDvirContext(now)
	current, ok := s.live.dvirByID(ctx, id)
	if !ok {
		s.writeError(writer, notFound("DVIR", id))
		return
	}
	author, err := dvirAuthor(ctx, stringValue(sanitized, fieldAuthorID))
	if err != nil {
		s.writeError(writer, err)
		return
	}
	signedAt, err := dvirSignedAt(sanitized, current, now)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if status := stringValue(current, fieldSafetyStatus); status != dvirSafetyStatusUnsafe {
		s.writeError(writer, invalidField(
			fieldIsResolved,
			fmt.Sprintf("cannot resolve a DVIR whose safetyStatus is %s", status),
		))
		return
	}

	notes := stringValue(sanitized, fieldMechanicNotes)
	err = s.store.Transact(func(tx *storeTx) error {
		if stringValue(current, fieldSimKind) == dvirSimKindAPI {
			_, updateErr := tx.update(ResourceDvirs, recordID(current), func(record Record) error {
				record[fieldSafetyStatus] = dvirSafetyStatusResolved
				record[fieldSecondSignature] = dvirSignature(
					stringOf(author[keyID]),
					stringOf(author[keyName]),
					signedAt,
					dvirSignatureTypeMechanic,
				)
				if notes != "" {
					record[fieldMechanicNotes] = notes
				}
				return nil
			}, now)
			return updateErr
		}
		return tx.upsertDvirOverlays([]Record{{
			keyID:              dvirResolutionIDPrefix + recordID(current),
			fieldSimKind:       dvirResolutionKindDvir,
			"dvirId":           recordID(current),
			fieldSimDriverID:   stringValue(current, fieldSimDriverID),
			fieldSimDay:        stringValue(current, fieldSimDay),
			fieldSimType:       stringValue(current, keyType),
			fieldSignedAtTime:  signedAt.UTC().Format(time.RFC3339),
			fieldUpdatedAtTime: now.UTC().Format(time.RFC3339),
			fieldResolvedBy:    author,
			fieldMechanicNotes: notes,
		}}, "")
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	updated, ok := s.live.dvirByID(s.live.newDvirContext(now), recordID(current))
	if !ok {
		s.writeError(writer, notFound("DVIR", id))
		return
	}
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+"|dvir-resolve",
		map[string]any{keyData: renderDvirHistory(view, updated)},
	)
}

func dvirSignedAt(body, current Record, now time.Time) (time.Time, error) {
	raw := stringValue(body, fieldSignedAtTime)
	if raw == "" {
		return now, nil
	}
	signedAt, err := parseRFC3339(raw)
	if err != nil {
		return time.Time{}, invalidField(fieldSignedAtTime, "must be an RFC 3339 timestamp")
	}
	if signedAt.After(now) {
		return time.Time{}, invalidField(fieldSignedAtTime, "cannot be in the future")
	}
	if submitted, parseErr := parseRFC3339(stringValue(current, fieldEndTime)); parseErr == nil &&
		signedAt.Before(submitted) {
		return time.Time{}, invalidField(fieldSignedAtTime, "cannot be before the DVIR was submitted")
	}
	return signedAt.Truncate(time.Second), nil
}
