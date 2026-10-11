package sim

import (
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	dvirTypePreTrip     = "preTrip"
	dvirTypePostTrip    = "postTrip"
	dvirTypeMechanic    = "mechanic"
	dvirTypeUnspecified = "unspecified"

	dvirSafetyStatusSafe     = "safe"
	dvirSafetyStatusUnsafe   = "unsafe"
	dvirSafetyStatusResolved = "resolved"

	dvirSignatureTypeDriver   = "driver"
	dvirSignatureTypeMechanic = "mechanic"

	dvirDefectAssetVehicle = "vehicle"
	dvirDefectAssetTrailer = "trailer"

	dvirUnsafeRate          = 0.10
	dvirSecondDefectRate    = 0.4
	dvirTrailerDefectRate   = 0.35
	dvirAutoResolveMinHours = 18.0
	dvirAutoResolveSpanHrs  = 24.0
	dvirUpdateLag           = 5 * 24 * time.Hour
	dvirLookupDays          = 400
	dvirTrailerDefectSeq    = 100

	fieldSimDriverID     = "simDriverId"
	fieldSimVehicleID    = "simVehicleId"
	fieldSimTrailerID    = "simTrailerId"
	fieldSimAssetID      = "simAssetId"
	fieldSimAssetKind    = "simAssetKind"
	fieldSimDay          = "simDay"
	fieldSimType         = "simType"
	fieldSimKind         = "simKind"
	fieldVehicleDefects  = "vehicleDefects"
	fieldTrailerDefects  = "trailerDefects"
	fieldAuthorSignature = "authorSignature"
	fieldSecondSignature = "secondSignature"
	fieldThirdSignature  = "thirdSignature"
	fieldSignatoryUser   = "signatoryUser"
	fieldSignedAtTime    = "signedAtTime"
	fieldSafetyStatus    = "safetyStatus"
	fieldMechanicNotes   = "mechanicNotes"
	fieldResolvedAtTime  = "resolvedAtTime"
	fieldResolvedBy      = "resolvedBy"
	fieldIsResolved      = "isResolved"
	fieldNotesUpdatedAt  = "mechanicNotesUpdatedAtTime"
	fieldDefectSeverity  = "defectSeverity"

	dvirResolutionKindDvir   = "dvir"
	dvirResolutionKindDefect = "defect"

	userRoleMaintenance = "Maintenance Technician"
)

type dvirDefectCatalogEntry struct {
	DefectType string
	Comment    string
	Repair     string
	Severity   string
}

var dvirDefectCatalog = []dvirDefectCatalogEntry{
	{
		DefectType: "Brake Hose",
		Comment:    "Brake hose chafing against frame rail",
		Repair:     "Replaced brake hose and added a chafe guard at the frame rail",
		Severity:   "major",
	},
	{
		DefectType: "Tires",
		Comment:    "Steer tire tread depth near minimum",
		Repair:     "Replaced steer tire and checked alignment",
		Severity:   "major",
	},
	{
		DefectType: "Lights",
		Comment:    "Marker lamp out on passenger side",
		Repair:     "Replaced marker lamp and cleaned the connector",
		Severity:   "minor",
	},
	{
		DefectType: "Air Compressor",
		Comment:    "Air compressor slow to build pressure",
		Repair:     "Replaced compressor governor; build-up time within spec",
		Severity:   "major",
	},
	{
		DefectType: "Wipers",
		Comment:    "Wiper blade streaking on driver side",
		Repair:     "Replaced both wiper blades",
		Severity:   "minor",
	},
	{
		DefectType: "Mirrors",
		Comment:    "Passenger mirror bracket loose",
		Repair:     "Tightened mirror bracket and replaced worn bushing",
		Severity:   "minor",
	},
}

var dvirTrailerDefectCatalog = []dvirDefectCatalogEntry{
	{
		DefectType: "Landing Gear",
		Comment:    "Landing gear crank binding on the curb side",
		Repair:     "Lubricated and adjusted landing gear gearbox",
		Severity:   "minor",
	},
	{
		DefectType: "Trailer Lights",
		Comment:    "Left rear stop lamp inoperative",
		Repair:     "Replaced stop lamp and repaired pigtail",
		Severity:   "major",
	},
	{
		DefectType: "Doors",
		Comment:    "Rear door seal torn at the lower corner",
		Repair:     "Replaced rear door seal",
		Severity:   "minor",
	},
	{
		DefectType: "Tires (Trailer)",
		Comment:    "Inner dual on axle 2 low on pressure",
		Repair:     "Repaired puncture and inflated to spec",
		Severity:   "major",
	},
}

type dvirHeader struct {
	ID        string
	Type      string
	DriverID  string
	VehicleID string
	TrailerID string
	Day       time.Time
	DayKey    string
	Start     time.Time
	End       time.Time
}

type dvirUser struct {
	ID   string
	Name string
}

type dvirResolution struct {
	At            time.Time
	By            map[string]any
	MechanicNotes string
	UpdatedAt     time.Time
}

type dvirContext struct {
	Now          time.Time
	Snapshot     *fleetSnapshot
	Roster       map[string]driverRoster
	Users        map[string]Record
	Mechanics    []dvirUser
	DvirOverlay  map[string]dvirResolution
	DefectByID   map[string]dvirResolution
	Overlays     []Record
	APIDvirs     []Record
	Geometries   map[string]*routeGeometry
	trailerCache map[string]string
}

func (s *Server) registerDvirRoutes() {
	s.mux.HandleFunc("GET /fleet/dvirs/history", s.handleDvirHistory)
	s.mux.HandleFunc("POST /fleet/dvirs", s.handleDvirCreate)
	s.mux.HandleFunc("PATCH /fleet/dvirs/{id}", s.handleDvirResolve)
	s.mux.HandleFunc("GET /dvirs/stream", s.handleDvirStream)
	s.mux.HandleFunc("GET /dvirs/{id}", s.handleDvirGet)
}

func (l *LiveSimulator) newDvirContext(now time.Time) *dvirContext {
	ctx := &dvirContext{
		Now:          now.UTC(),
		Snapshot:     l.fleet(),
		Roster:       l.loadDriverRoster(),
		Users:        map[string]Record{},
		DvirOverlay:  map[string]dvirResolution{},
		DefectByID:   map[string]dvirResolution{},
		Geometries:   map[string]*routeGeometry{},
		trailerCache: map[string]string{},
	}
	if l.store == nil {
		return ctx
	}
	if users, err := l.store.List(ResourceUsers); err == nil {
		for _, user := range users {
			ctx.Users[recordID(user)] = user
			if userHasRole(user, userRoleMaintenance) {
				ctx.Mechanics = append(ctx.Mechanics, dvirUser{
					ID:   recordID(user),
					Name: stringValue(user, keyName),
				})
			}
		}
		sort.Slice(ctx.Mechanics, func(i, j int) bool { return ctx.Mechanics[i].ID < ctx.Mechanics[j].ID })
	}
	if overlays, err := l.store.List(ResourceDvirResolutions); err == nil {
		ctx.Overlays = overlays
		for _, overlay := range overlays {
			resolution := dvirResolutionOf(overlay)
			if resolution.At.After(ctx.Now) {
				continue
			}
			switch stringValue(overlay, fieldSimKind) {
			case dvirResolutionKindDvir:
				ctx.DvirOverlay[stringValue(overlay, "dvirId")] = resolution
			case dvirResolutionKindDefect:
				ctx.DefectByID[stringValue(overlay, "defectId")] = resolution
			}
		}
	}
	if dvirs, err := l.store.List(ResourceDvirs); err == nil {
		ctx.APIDvirs = dvirs
	}
	return ctx
}

func userHasRole(user Record, roleName string) bool {
	for _, raw := range listOf(user["roles"]) {
		if nestedString(Record(mapOf(raw)), "role", keyName) == roleName {
			return true
		}
	}
	return false
}

func dvirResolutionOf(overlay Record) dvirResolution {
	at, _ := parseRFC3339(stringValue(overlay, fieldSignedAtTime))
	updated, err := parseRFC3339(stringValue(overlay, fieldUpdatedAtTime))
	if err != nil {
		updated = at
	}
	return dvirResolution{
		At:            at,
		By:            cloneMap(mapOf(overlay[fieldResolvedBy])),
		MechanicNotes: stringValue(overlay, fieldMechanicNotes),
		UpdatedAt:     updated,
	}
}

func (l *LiveSimulator) dvirHeaders(ctx *dvirContext, fromDay, toDay time.Time) []dvirHeader {
	fromDay = maxTime(fromDay.UTC().Truncate(24*time.Hour), telemetryEpoch)
	toDay = toDay.UTC().Truncate(24 * time.Hour)
	if toDay.Before(fromDay) {
		return nil
	}
	driverIDs := make([]string, 0, len(ctx.Roster))
	for driverID := range ctx.Roster {
		driverIDs = append(driverIDs, driverID)
	}
	sort.Strings(driverIDs)

	dayCount := int(toDay.Sub(fromDay)/(24*time.Hour)) + 1
	out := make([]dvirHeader, 0, len(driverIDs)*dayCount*2)
	for _, driverID := range driverIDs {
		vehicleID := strings.TrimSpace(ctx.Roster[driverID].VehicleID)
		if vehicleID == "" {
			continue
		}
		cutoff, deactivated := driverDeactivatedAt(ctx.Snapshot.driverByID[driverID])
		trailerID := ctx.trailerFor(vehicleID)
		for day := fromDay; !day.After(toDay); day = day.Add(24 * time.Hour) {
			for _, header := range l.driverDayDvirHeaders(driverID, vehicleID, trailerID, day) {
				if header.End.After(ctx.Now) {
					continue
				}
				if deactivated && !cutoff.IsZero() && header.End.After(cutoff) {
					continue
				}
				out = append(out, header)
			}
		}
	}
	return out
}

func (ctx *dvirContext) trailerFor(vehicleID string) string {
	if trailerID, ok := ctx.trailerCache[vehicleID]; ok {
		return trailerID
	}
	trailerID := ""
	if trailer := coupledTrailer(ctx.Snapshot.assetByID, vehicleID); trailer != nil {
		trailerID = recordID(trailer)
	}
	ctx.trailerCache[vehicleID] = trailerID
	return trailerID
}

func (l *LiveSimulator) driverDayDvirHeaders(
	driverID, vehicleID, trailerID string,
	day time.Time,
) []dvirHeader {
	dayKey := day.Format(dailyLogDateLayout)
	compactDay := day.Format("20060102")
	dayCtx := l.buildDailyEventContext(driverID, vehicleID, day)
	shiftStart := l.shiftStartForDay(driverID, day).Truncate(time.Second)
	preDuration := time.Duration(
		(8 + 9*l.hashFraction("dvir-duration", driverID, dayKey, dvirTypePreTrip)) *
			float64(time.Minute),
	)
	postStart := dayCtx.DrivingEnd.Truncate(time.Second)
	postDuration := time.Duration(
		(6 + 9*l.hashFraction("dvir-duration", driverID, dayKey, dvirTypePostTrip)) *
			float64(time.Minute),
	)
	build := func(dvirType string, start, end time.Time) dvirHeader {
		return dvirHeader{
			ID:        dvirRecordID(compactDay, driverID, dvirTypeSuffix(dvirType)),
			Type:      dvirType,
			DriverID:  driverID,
			VehicleID: vehicleID,
			TrailerID: trailerID,
			Day:       day,
			DayKey:    dayKey,
			Start:     start,
			End:       end.Truncate(time.Second),
		}
	}
	return []dvirHeader{
		build(dvirTypePreTrip, shiftStart, shiftStart.Add(preDuration)),
		build(dvirTypePostTrip, postStart, postStart.Add(postDuration)),
	}
}

func dvirTypeSuffix(dvirType string) string {
	if dvirType == dvirTypePreTrip {
		return "pre"
	}
	return "post"
}

func (l *LiveSimulator) dvirCore(ctx *dvirContext, header *dvirHeader) Record {
	driver := ctx.Snapshot.driverByID[header.DriverID]
	driverName := stringValue(driver, keyName)
	if driverName == "" {
		driverName = ctx.Roster[header.DriverID].Name
	}
	record := Record{
		keyID:             header.ID,
		keyType:           header.Type,
		fieldStartTime:    header.Start.UTC().Format(time.RFC3339),
		fieldEndTime:      header.End.UTC().Format(time.RFC3339),
		fieldSimKind:      "generated",
		fieldSimDriverID:  header.DriverID,
		fieldSimVehicleID: header.VehicleID,
		fieldSimDay:       header.DayKey,
		fieldAuthorSignature: dvirSignature(
			header.DriverID,
			driverName,
			header.End,
			dvirSignatureTypeDriver,
		),
	}
	if header.TrailerID != "" {
		record[fieldSimTrailerID] = header.TrailerID
	}
	if plate := stringValue(ctx.Snapshot.assetByID[header.VehicleID], keyLicensePlate); plate != "" {
		record[keyLicensePlate] = plate
	}

	vehicleDefects, trailerDefects := l.dvirDefectPlan(header)
	autoAt, autoBy := l.dvirAutoResolution(ctx, header)
	overlay, patched := ctx.DvirOverlay[header.ID]
	lastResolved := time.Time{}
	var lastBy map[string]any
	unresolved := 0
	resolve := func(defects []any) {
		for _, raw := range defects {
			defect := mapOf(raw)
			resolution, ok := ctx.DefectByID[stringOf(defect[keyID])]
			switch {
			case patched && (!ok || overlay.At.Before(resolution.At)):
				resolution, ok = overlay, true
			case !ok && !autoAt.IsZero() && !autoAt.After(ctx.Now):
				resolution = dvirResolution{
					At:            autoAt,
					By:            autoBy,
					MechanicNotes: dvirRepairNote(stringOf(defect["defectType"])),
				}
				ok = true
			}
			if !ok {
				unresolved++
				continue
			}
			applyDefectResolution(defect, resolution)
			if resolution.At.After(lastResolved) {
				lastResolved = resolution.At
				lastBy = resolution.By
			}
		}
	}
	resolve(vehicleDefects)
	resolve(trailerDefects)
	record[fieldVehicleDefects] = vehicleDefects
	record[fieldTrailerDefects] = trailerDefects

	updated := header.End
	switch {
	case len(vehicleDefects)+len(trailerDefects) == 0:
		record[fieldSafetyStatus] = dvirSafetyStatusSafe
	case unresolved > 0:
		record[fieldSafetyStatus] = dvirSafetyStatusUnsafe
		updated = maxTime(updated, lastResolved)
	default:
		record[fieldSafetyStatus] = dvirSafetyStatusResolved
		updated = maxTime(updated, lastResolved)
		if lastBy != nil {
			record[fieldSecondSignature] = dvirSignature(
				stringOf(lastBy[keyID]),
				stringOf(lastBy[keyName]),
				lastResolved,
				stringOf(lastBy[keyType]),
			)
		}
		switch {
		case patched && overlay.MechanicNotes != "":
			record[fieldMechanicNotes] = overlay.MechanicNotes
		case !patched:
			if notes := dvirSummaryNotes(vehicleDefects, trailerDefects); notes != "" {
				record[fieldMechanicNotes] = notes
			}
		}
		if acknowledged, ok := l.dvirAcknowledgement(header, lastResolved); ok &&
			!acknowledged.After(ctx.Now) {
			record[fieldThirdSignature] = dvirSignature(
				header.DriverID,
				driverName,
				acknowledged,
				dvirSignatureTypeDriver,
			)
			updated = maxTime(updated, acknowledged)
		}
	}
	if patched {
		updated = maxTime(updated, overlay.UpdatedAt)
	}
	record[fieldUpdatedAtTime] = updated.UTC().Format(time.RFC3339)
	return record
}

func dvirSignature(id, name string, at time.Time, signatureType string) map[string]any {
	return map[string]any{
		fieldSignatoryUser: map[string]any{keyID: id, keyName: name},
		fieldSignedAtTime:  at.UTC().Format(time.RFC3339),
		keyType:            signatureType,
	}
}

func applyDefectResolution(defect map[string]any, resolution dvirResolution) {
	stamp := resolution.At.UTC().Format(time.RFC3339)
	defect[fieldIsResolved] = true
	defect[fieldResolvedAtTime] = stamp
	if len(resolution.By) > 0 {
		defect[fieldResolvedBy] = cloneMap(resolution.By)
	}
	if resolution.MechanicNotes != "" {
		defect[fieldMechanicNotes] = resolution.MechanicNotes
		defect[fieldNotesUpdatedAt] = stamp
	}
}

func dvirSummaryNotes(groups ...[]any) string {
	notes := make([]string, 0, 2)
	for _, defects := range groups {
		for _, raw := range defects {
			if note := stringOf(mapOf(raw)[fieldMechanicNotes]); note != "" {
				notes = append(notes, note)
			}
		}
	}
	return strings.Join(notes, "; ")
}

func (l *LiveSimulator) dvirDefectPlan(header *dvirHeader) (vehicleDefects, trailerDefects []any) {
	vehicleDefects = []any{}
	trailerDefects = []any{}
	if l.hashFraction("dvir-unsafe", header.DriverID, header.DayKey, header.Type) >= dvirUnsafeRate {
		return vehicleDefects, trailerDefects
	}
	created := header.End.UTC().Format(time.RFC3339)
	count := 1
	if l.hashFraction("dvir-defect-count", header.DriverID, header.DayKey, header.Type) <
		dvirSecondDefectRate {
		count = 2
	}
	base := catalogIndex(
		l.hashFraction("dvir-defect", header.DriverID, header.DayKey, header.Type),
		len(dvirDefectCatalog),
	)
	trailerDefect := header.TrailerID != "" &&
		l.hashFraction("dvir-trailer-defect", header.DriverID, header.DayKey, header.Type) <
			dvirTrailerDefectRate
	if trailerDefect {
		count--
	}
	for idx := 0; idx < count; idx++ {
		entry := dvirDefectCatalog[(base+idx)%len(dvirDefectCatalog)]
		vehicleDefects = append(vehicleDefects, dvirDefectRecord(
			dvirDefectID(header.ID, idx+1),
			entry,
			created,
			dvirDefectAssetVehicle,
			header.VehicleID,
		))
	}
	if trailerDefect {
		entry := dvirTrailerDefectCatalog[catalogIndex(
			l.hashFraction("dvir-trailer-defect-type", header.DriverID, header.DayKey, header.Type),
			len(dvirTrailerDefectCatalog),
		)]
		trailerDefects = append(trailerDefects, dvirDefectRecord(
			dvirDefectID(header.ID, dvirTrailerDefectSeq+1),
			entry,
			created,
			dvirDefectAssetTrailer,
			header.TrailerID,
		))
	}
	return vehicleDefects, trailerDefects
}

func catalogIndex(fraction float64, size int) int {
	return min(int(math.Floor(float64(size)*fraction)), size-1)
}

func dvirDefectRecord(
	id string,
	entry dvirDefectCatalogEntry,
	created string,
	assetKind string,
	assetID string,
) map[string]any {
	return map[string]any{
		keyID:               id,
		"defectType":        entry.DefectType,
		"comment":           entry.Comment,
		fieldDefectSeverity: entry.Severity,
		fieldCreatedAtTime:  created,
		fieldIsResolved:     false,
		fieldSimAssetKind:   assetKind,
		fieldSimAssetID:     assetID,
	}
}

func (l *LiveSimulator) dvirAutoResolution(
	ctx *dvirContext,
	header *dvirHeader,
) (at time.Time, by map[string]any) {
	delay := dvirAutoResolveMinHours +
		dvirAutoResolveSpanHrs*l.hashFraction("dvir-auto-resolve", header.ID)
	at = header.End.Add(time.Duration(delay * float64(time.Hour))).Truncate(time.Second)
	if len(ctx.Mechanics) > 0 {
		mechanic := ctx.Mechanics[catalogIndex(
			l.hashFraction("dvir-mechanic", header.VehicleID),
			len(ctx.Mechanics),
		)]
		by = map[string]any{
			keyID:   mechanic.ID,
			keyName: mechanic.Name,
			keyType: dvirSignatureTypeMechanic,
		}
	}
	return at, by
}

func dvirRepairNote(defectType string) string {
	for _, catalog := range [][]dvirDefectCatalogEntry{dvirDefectCatalog, dvirTrailerDefectCatalog} {
		for _, entry := range catalog {
			if entry.DefectType == defectType {
				return entry.Repair
			}
		}
	}
	return ""
}

func (l *LiveSimulator) dvirAcknowledgement(header *dvirHeader, resolvedAt time.Time) (time.Time, bool) {
	if resolvedAt.IsZero() {
		return time.Time{}, false
	}
	day := resolvedAt.UTC().Truncate(24 * time.Hour)
	for offset := 0; offset < 4; offset++ {
		candidate := l.shiftStartForDay(header.DriverID, day.Add(time.Duration(offset)*24*time.Hour)).
			Truncate(time.Second)
		if candidate.After(resolvedAt) {
			return candidate, true
		}
	}
	return time.Time{}, false
}

func (l *LiveSimulator) decorateDvir(ctx *dvirContext, record Record) {
	if stringValue(record, fieldSimKind) != "generated" {
		return
	}
	vehicleID := stringValue(record, fieldSimVehicleID)
	end, err := parseRFC3339(stringValue(record, fieldEndTime))
	if err != nil {
		return
	}
	start, startErr := parseRFC3339(stringValue(record, fieldStartTime))
	if startErr != nil {
		start = end
	}
	record[keyOdometerMeters] = l.vehicleOdometerMeters(ctx.Snapshot, vehicleID, end)
	record[keyLocation] = l.dvirLocation(ctx, record, vehicleID, start)
	if stringValue(record, keyType) == dvirTypePreTrip {
		record["walkaroundPhotos"] = dvirWalkaroundPhotos(recordID(record), start, end, ctx.Now)
	}
}

func (l *LiveSimulator) dvirLocation(
	ctx *dvirContext,
	record Record,
	vehicleID string,
	at time.Time,
) string {
	geometry := l.cachedRouteGeometry(ctx.Geometries, ctx.Snapshot.waypoints, vehicleID)
	if geometry != nil && len(geometry.Points) > 0 {
		state := l.routeStateForGeometry(vehicleID, geometry, at, at.Add(-time.Minute), ctx.Now)
		if formatted := formattedLocationFromAddress(state.Address); formatted != "" {
			return formatted
		}
		if formatted := ctx.Snapshot.reverseGeocode(state); formatted != "" {
			return formatted
		}
	}
	_, address := l.driverHomeTerminal(ctx.Snapshot.driverByID[stringValue(record, fieldSimDriverID)])
	return address
}

func dvirWalkaroundPhotos(dvirID string, start, end, now time.Time) []any {
	names := []string{"Front", "Driver Side", "Rear", "Passenger Side"}
	span := end.Sub(start)
	photos := make([]any, 0, len(names))
	for idx, name := range names {
		taken := start.Add(span * time.Duration(idx+1) / time.Duration(len(names)+1))
		photoID := deterministicUUID("dvir-walkaround", dvirID, strconv.Itoa(idx))
		photos = append(photos, map[string]any{
			keyName:            name,
			fieldCreatedAtTime: taken.UTC().Format(time.RFC3339),
			"url": dvirMediaURLPrefix + dvirID + "/" + photoID + ".jpg?expires=" +
				strconv.FormatInt(now.Add(24*time.Hour).Unix(), 10),
		})
	}
	return photos
}

const dvirMediaURLPrefix = "https://samsara-driver-media-upload.s3.us-west-2.amazonaws.com/dvirs/"

func (l *LiveSimulator) dvirCores(ctx *dvirContext, fromDay, toDay time.Time) []Record {
	headers := l.dvirHeaders(ctx, fromDay, toDay)
	out := make([]Record, 0, len(headers)+len(ctx.APIDvirs))
	for idx := range headers {
		out = append(out, l.dvirCore(ctx, &headers[idx]))
	}
	for _, record := range ctx.APIDvirs {
		out = append(out, cloneRecord(record))
	}
	return out
}

func (l *LiveSimulator) dvirByID(ctx *dvirContext, id string) (Record, bool) {
	clean := strings.TrimSpace(id)
	for _, record := range ctx.APIDvirs {
		if recordID(record) == clean {
			return cloneRecord(record), true
		}
	}
	header, ok := l.findDvirHeader(ctx, func(candidate *dvirHeader) bool { return candidate.ID == clean })
	if !ok {
		return nil, false
	}
	record := l.dvirCore(ctx, &header)
	l.decorateDvir(ctx, record)
	return record, true
}

func (l *LiveSimulator) findDvirHeader(
	ctx *dvirContext,
	match func(header *dvirHeader) bool,
) (dvirHeader, bool) {
	toDay := ctx.Now.Truncate(24 * time.Hour)
	fromDay := toDay.Add(-dvirLookupDays * 24 * time.Hour)
	headers := l.dvirHeaders(ctx, fromDay, toDay)
	for idx := len(headers) - 1; idx >= 0; idx-- {
		if match(&headers[idx]) {
			return headers[idx], true
		}
	}
	return dvirHeader{}, false
}

func (l *LiveSimulator) findDvirDefect(ctx *dvirContext, defectID string) (Record, map[string]any, bool) {
	var parent Record
	var found map[string]any
	_, ok := l.findDvirHeader(ctx, func(header *dvirHeader) bool {
		if l.hashFraction("dvir-unsafe", header.DriverID, header.DayKey, header.Type) >= dvirUnsafeRate {
			return false
		}
		vehicleDefects, trailerDefects := l.dvirDefectPlan(header)
		for _, raw := range append(vehicleDefects, trailerDefects...) {
			if stringOf(mapOf(raw)[keyID]) != defectID {
				continue
			}
			parent = l.dvirCore(ctx, header)
			for _, current := range append(listOf(parent[fieldVehicleDefects]), listOf(parent[fieldTrailerDefects])...) {
				if stringOf(mapOf(current)[keyID]) == defectID {
					found = mapOf(current)
				}
			}
			return true
		}
		return false
	})
	return parent, found, ok && found != nil
}

func (s *Server) handleDvirHistory(writer http.ResponseWriter, request *http.Request) {
	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if startTime == nil || endTime == nil {
		s.writeError(writer, ErrTimeRangeRequired)
		return
	}
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(request.URL.Query())
	filter := view.snap.tags.filter(tagIDs, parentTagIDs)

	ctx := s.live.newDvirContext(view.now)
	cores := s.live.dvirCores(ctx, startTime.Add(-24*time.Hour), *endTime)
	selected := make([]Record, 0, len(cores))
	for _, record := range cores {
		end, parseErr := parseRFC3339(stringValue(record, fieldEndTime))
		if parseErr != nil || !end.After(*startTime) || end.After(*endTime) {
			continue
		}
		if !dvirMatchesTags(view, filter, record) {
			continue
		}
		selected = append(selected, record)
	}
	sortDvirs(selected, fieldEndTime)
	page, pagination, err := paginate(selected, request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	data := make([]any, 0, len(page))
	for _, record := range page {
		s.live.decorateDvir(ctx, record)
		data = append(data, renderDvirHistory(view, record))
	}
	payload := map[string]any{keyData: data, keyPagination: pagination}
	s.respondJSON(writer, request, requestSignature(request)+"|dvir-history", payload)
}

func dvirMatchesTags(view *fleetView, filter tagFilter, record Record) bool {
	if !filter.active {
		return true
	}
	if vehicleID := stringValue(record, fieldSimVehicleID); vehicleID != "" &&
		filter.matches(view.snap.tags, tagMembersVehicles, vehicleID) {
		return true
	}
	trailerID := stringValue(record, fieldSimTrailerID)
	return trailerID != "" && filter.matches(view.snap.tags, tagMembersAssets, trailerID)
}

func sortDvirs(records []Record, field string) {
	sort.SliceStable(records, func(i, j int) bool {
		left := stringValue(records[i], field)
		right := stringValue(records[j], field)
		if left == right {
			return recordID(records[i]) < recordID(records[j])
		}
		return left < right
	})
}

func renderDvirHistory(view *fleetView, record Record) map[string]any {
	out := map[string]any{
		keyID:                recordID(record),
		keyType:              stringValue(record, keyType),
		fieldSafetyStatus:    stringValue(record, fieldSafetyStatus),
		fieldStartTime:       stringValue(record, fieldStartTime),
		fieldEndTime:         stringValue(record, fieldEndTime),
		fieldAuthorSignature: cloneAny(record[fieldAuthorSignature]),
	}
	for _, field := range []string{
		fieldSecondSignature,
		fieldThirdSignature,
		keyOdometerMeters,
	} {
		if value, ok := record[field]; ok && value != nil {
			out[field] = cloneAny(value)
		}
	}
	for _, field := range []string{keyLicensePlate, keyLocation, fieldMechanicNotes} {
		if value := stringValue(record, field); value != "" {
			out[field] = value
		}
	}
	vehicleID := stringValue(record, fieldSimVehicleID)
	trailerID := stringValue(record, fieldSimTrailerID)
	if vehicleID != "" {
		out[keyVehicle] = vehicleTinyWithExternalIDs(vehicleID, view.snap.assetByID)
	}
	if trailerID != "" {
		out["trailer"] = map[string]any{
			keyID:   trailerID,
			keyName: vehicleIDToName(trailerID, view.snap.assetByID),
		}
		if vehicleID != "" {
			out["trailerName"] = vehicleIDToName(trailerID, view.snap.assetByID)
		}
	}
	if defects := renderDvirDefects(view, listOf(record[fieldVehicleDefects])); len(defects) > 0 {
		out[fieldVehicleDefects] = defects
	}
	if defects := renderDvirDefects(view, listOf(record[fieldTrailerDefects])); len(defects) > 0 {
		out[fieldTrailerDefects] = defects
	}
	return out
}

func renderDvirDefects(view *fleetView, defects []any) []any {
	out := make([]any, 0, len(defects))
	for _, raw := range defects {
		defect := mapOf(raw)
		rendered := map[string]any{}
		for key, value := range defect {
			if key == fieldSimAssetID || key == fieldSimAssetKind || key == fieldDefectSeverity {
				continue
			}
			rendered[key] = cloneAny(value)
		}
		assetID := stringOf(defect[fieldSimAssetID])
		if stringOf(defect[fieldSimAssetKind]) == dvirDefectAssetTrailer {
			rendered["trailer"] = map[string]any{
				keyID:   assetID,
				keyName: vehicleIDToName(assetID, view.snap.assetByID),
			}
		} else {
			rendered[keyVehicle] = vehicleTinyWithExternalIDs(assetID, view.snap.assetByID)
		}
		out = append(out, rendered)
	}
	return out
}

func (l *LiveSimulator) DvirWebhookEmissions(
	now time.Time,
	windowStart time.Time,
	windowEnd time.Time,
) []WebhookEmission {
	ctx := l.newDvirContext(now)
	headers := l.dvirHeaders(ctx, windowStart.Add(-24*time.Hour), windowEnd)
	out := make([]WebhookEmission, 0, len(headers)/8+1)
	for idx := range headers {
		header := &headers[idx]
		if !header.End.After(windowStart) || header.End.After(windowEnd) {
			continue
		}
		record := l.dvirCore(ctx, header)
		l.decorateDvir(ctx, record)
		out = append(out, WebhookEmission{
			EventType: "DvirSubmitted",
			UniqueKey: "DvirSubmitted|" + header.ID,
			Data:      dvirWebhookData(ctx.Snapshot, record),
		})
	}
	return out
}

func dvirWebhookData(snapshot *fleetSnapshot, record Record) map[string]any {
	dvir := map[string]any{
		keyID:                recordID(record),
		keyType:              stringValue(record, keyType),
		fieldSafetyStatus:    stringValue(record, fieldSafetyStatus),
		fieldStartTime:       stringValue(record, fieldStartTime),
		fieldEndTime:         stringValue(record, fieldEndTime),
		fieldAuthorSignature: webhookSignature(record[fieldAuthorSignature]),
		"needsCorrection":    stringValue(record, fieldSafetyStatus) == dvirSafetyStatusUnsafe,
	}
	for _, field := range []string{fieldSecondSignature, fieldThirdSignature} {
		if signature, ok := record[field]; ok {
			dvir[field] = webhookSignature(signature)
		}
	}
	if value, ok := record[keyOdometerMeters]; ok {
		dvir[keyOdometerMeters] = value
	}
	if location := stringValue(record, keyLocation); location != "" {
		dvir[keyFormattedLocation] = location
	}
	if notes := stringValue(record, fieldMechanicNotes); notes != "" {
		dvir[fieldMechanicNotes] = notes
	}
	trailerID := stringValue(record, fieldSimTrailerID)
	if trailerID != "" {
		dvir["trailer"] = trailerWebhookTiny(snapshot, trailerID)
	}
	defects := make([]any, 0, 2)
	for _, raw := range append(listOf(record[fieldVehicleDefects]), listOf(record[fieldTrailerDefects])...) {
		defect := mapOf(raw)
		rendered := map[string]any{}
		for key, value := range defect {
			if key == fieldSimAssetID || key == fieldSimAssetKind {
				continue
			}
			rendered[key] = cloneAny(value)
		}
		assetID := stringOf(defect[fieldSimAssetID])
		if stringOf(defect[fieldSimAssetKind]) == dvirDefectAssetTrailer {
			rendered["trailer"] = trailerWebhookTiny(snapshot, assetID)
		} else {
			rendered[keyVehicle] = vehicleWithGatewayTiny(snapshot, assetID)
		}
		defects = append(defects, rendered)
	}
	if len(defects) > 0 {
		dvir["defects"] = defects
	}

	data := map[string]any{"dvir": dvir}
	if vehicleID := stringValue(record, fieldSimVehicleID); vehicleID != "" {
		data[keyVehicle] = vehicleWithGatewayTiny(snapshot, vehicleID)
	}
	if driverID := stringValue(record, fieldSimDriverID); driverID != "" {
		driver := map[string]any{keyID: driverID}
		if stored, ok := snapshot.driverByID[driverID]; ok {
			driver[keyName] = stringValue(stored, keyName)
			driver[fieldExternalIDs] = renderExternalIDs(stored, nil)
		}
		data[keyDriver] = driver
	}
	return data
}

func webhookSignature(raw any) map[string]any {
	signature := mapOf(raw)
	user := mapOf(signature[fieldSignatoryUser])
	return map[string]any{
		fieldSignatoryUser: map[string]any{
			keyID:   stringOf(user[keyID]),
			keyName: stringOf(user[keyName]),
		},
		fieldSignedAtTime: stringOf(signature[fieldSignedAtTime]),
		keyType:           stringOf(signature[keyType]),
	}
}

func trailerWebhookTiny(snapshot *fleetSnapshot, trailerID string) map[string]any {
	trailer := snapshot.assetByID[trailerID]
	return map[string]any{
		keyID:            trailerID,
		keyName:          vehicleIDToName(trailerID, snapshot.assetByID),
		fieldExternalIDs: renderExternalIDs(trailer, vehicleAutoExternalIDs),
	}
}

func vehicleWithGatewayTiny(snapshot *fleetSnapshot, vehicleID string) map[string]any {
	asset := snapshot.assetByID[vehicleID]
	out := map[string]any{
		keyID:            vehicleID,
		keyName:          vehicleIDToName(vehicleID, snapshot.assetByID),
		"assetType":      assetType(asset),
		fieldExternalIDs: renderExternalIDs(asset, vehicleAutoExternalIDs),
	}
	if plate := stringValue(asset, keyLicensePlate); plate != "" {
		out[keyLicensePlate] = plate
	}
	if vin := stringValue(asset, keyVIN); vin != "" {
		out[keyVIN] = vin
	}
	for _, field := range []string{fieldGateway, fieldInstalledGateway} {
		if gateway, ok := anyAsMap(asset[field]); ok {
			out[fieldGateway] = map[string]any{
				keyModel:  stringValue(Record(gateway), keyModel),
				keySerial: stringValue(Record(gateway), keySerial),
			}
			break
		}
	}
	return out
}
