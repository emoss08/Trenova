package sim

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	assignmentTypeDriverApp   = "driverApp"
	assignmentTypeExternal    = "external"
	assignmentTypeVoiceSignIn = "voiceSignIn"

	assignmentRestMergeGap        = 2 * time.Hour
	assignmentGenerationPadding   = 24 * time.Hour
	legacyAssignmentMaxRange      = 7 * 24 * time.Hour
	maxAssignmentSourceNameLength = 100
	assignmentSubmittedMessage    = "Driver assignment was successfully submitted"
	assignmentUpdatedMessage      = "Driver assignment was successfully updated"

	fieldDriverID       = "driverId"
	fieldVehicleID      = "vehicleId"
	fieldStartTime      = "startTime"
	fieldEndTime        = "endTime"
	fieldAssignedAtTime = "assignedAtTime"
	fieldIsPassenger    = "isPassenger"
	fieldMetadata       = "metadata"
	fieldSourceName     = "sourceName"
	fieldAssignmentType = "assignmentType"
	filterByDrivers     = "drivers"
	filterByVehicles    = "vehicles"
)

var assignmentTypeFilterValues = []string{
	"HOS",
	"idCard",
	"static",
	"faceId",
	"tachograph",
	"safetyManual",
	"RFID",
	"trailer",
	assignmentTypeExternal,
	"qrCode",
	assignmentTypeDriverApp,
	assignmentTypeVoiceSignIn,
	"smartAssign",
}

type apiAssignment struct {
	ID          string
	DriverID    string
	VehicleID   string
	Start       time.Time
	End         *time.Time
	AssignedAt  time.Time
	IsPassenger bool
	Type        string
	SourceName  string
}

type driverSignOut struct {
	DriverID string
	At       time.Time
}

type assignmentInterval struct {
	DriverID    string
	VehicleID   string
	Start       time.Time
	End         *time.Time
	AssignedAt  time.Time
	IsPassenger bool
	Type        string
	SourceName  string
}

type timeBlock struct {
	Start time.Time
	End   time.Time
}

type assignmentQuery struct {
	WindowStart *time.Time
	WindowEnd   *time.Time
	DriverIDs   map[string]struct{}
	VehicleIDs  map[string]struct{}
	Types       map[string]struct{}
	SourceName  string
	DriverApp   bool
	API         bool
}

func parseAPIAssignments(records []Record) []apiAssignment {
	out := make([]apiAssignment, 0, len(records))
	for _, record := range records {
		start, err := parseRFC3339(stringValue(record, fieldStartTime))
		if err != nil {
			continue
		}
		assignment := apiAssignment{
			ID:          recordID(record),
			DriverID:    stringValue(record, fieldDriverID),
			VehicleID:   stringValue(record, fieldVehicleID),
			Start:       start,
			AssignedAt:  start,
			Type:        stringValue(record, fieldAssignmentType),
			SourceName:  nestedString(record, fieldMetadata, fieldSourceName),
			IsPassenger: record[fieldIsPassenger] == true,
		}
		if end, endErr := parseRFC3339(stringValue(record, fieldEndTime)); endErr == nil {
			assignment.End = &end
		}
		if assigned, assignedErr := parseRFC3339(
			stringValue(record, fieldAssignedAtTime),
		); assignedErr == nil {
			assignment.AssignedAt = assigned
		}
		if assignment.Type == "" {
			assignment.Type = assignmentTypeExternal
		}
		out = append(out, assignment)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Start.Equal(out[j].Start) {
			return out[i].Start.Before(out[j].Start)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func mergeAssignments(groups ...[]apiAssignment) []apiAssignment {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	out := make([]apiAssignment, 0, total)
	for _, group := range groups {
		out = append(out, group...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Start.Equal(out[j].Start) {
			return out[i].Start.Before(out[j].Start)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func parseSignOuts(records []Record) []driverSignOut {
	out := make([]driverSignOut, 0, len(records))
	for _, record := range records {
		at, err := parseRFC3339(stringValue(record, "signedOutAtTime"))
		if err != nil {
			continue
		}
		out = append(out, driverSignOut{DriverID: stringValue(record, fieldDriverID), At: at})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out
}

func (a *apiAssignment) activeAt(at time.Time) bool {
	return !a.Start.After(at) && (a.End == nil || at.Before(*a.End))
}

func (f *fleetSnapshot) activeAPIAssignments(at time.Time) []apiAssignment {
	out := make([]apiAssignment, 0, len(f.assignments))
	for idx := range f.assignments {
		assignment := &f.assignments[idx]
		if assignment.IsPassenger || !assignment.activeAt(at) {
			continue
		}
		if driver, ok := f.driverByID[assignment.DriverID]; !ok || driverDeactivated(driver) {
			continue
		}
		if asset, ok := f.assetByID[assignment.VehicleID]; !ok ||
			stringValue(asset, keyType) != assetTypeVehicle {
			continue
		}
		out = append(out, *assignment)
	}
	return out
}

func driverDeactivated(driver Record) bool {
	return stringValue(driver, fieldDriverActivationStatus) == driverStatusDeactivated
}

func driverDeactivatedAt(driver Record) (time.Time, bool) {
	if !driverDeactivated(driver) {
		return time.Time{}, false
	}
	for _, key := range []string{fieldSimDeactivatedAtTime, fieldUpdatedAtTime} {
		if at, err := parseRFC3339(stringValue(driver, key)); err == nil {
			return at, true
		}
	}
	return time.Time{}, true
}

func isRestStatus(status string) bool {
	return status == hosStatusOffDuty || status == hosStatusSleeperBed
}

func workBlocks(segments []timelineSegment) []timeBlock {
	blocks := make([]timeBlock, 0, len(segments)/6+1)
	for _, segment := range segments {
		if isRestStatus(segment.Status) {
			continue
		}
		if count := len(blocks); count > 0 &&
			segment.Start.Sub(blocks[count-1].End) < assignmentRestMergeGap {
			blocks[count-1].End = maxTime(blocks[count-1].End, segment.End)
			continue
		}
		blocks = append(blocks, timeBlock{Start: segment.Start, End: segment.End})
	}
	return blocks
}

func (l *LiveSimulator) derivedDriverAppAssignments(
	snapshot *fleetSnapshot,
	driverID string,
	windowStart time.Time,
	windowEnd time.Time,
	now time.Time,
) []assignmentInterval {
	driver, ok := snapshot.driverByID[driverID]
	if !ok {
		return nil
	}
	vehicleID := strings.TrimSpace(snapshot.baseRoster[driverID].VehicleID)
	vehicle, vehicleOK := snapshot.assetByID[vehicleID]
	if vehicleID == "" || !vehicleOK {
		return nil
	}
	floor := laterOf(
		recordTime(driver, fieldCreatedAtTime),
		recordTime(vehicle, fieldCreatedAtTime),
	)
	deactivatedAt, deactivated := driverDeactivatedAt(driver)

	timeline := l.driverTimelineSegments(
		driverID,
		windowStart.Add(-assignmentGenerationPadding),
		windowEnd.Add(assignmentGenerationPadding),
		now,
	)
	out := make([]assignmentInterval, 0, 8)
	for _, block := range workBlocks(timeline) {
		start := block.Start
		end := block.End
		if !floor.IsZero() && end.Before(floor) {
			continue
		}
		start = laterOf(start, floor)
		if start.After(now) {
			continue
		}
		if deactivated && !deactivatedAt.After(start) {
			continue
		}
		if deactivated && deactivatedAt.Before(end) {
			end = deactivatedAt
		}
		for _, signOut := range snapshot.signOuts {
			if signOut.DriverID == driverID && signOut.At.After(start) && signOut.At.Before(end) {
				end = signOut.At
				break
			}
		}
		interval := assignmentInterval{
			DriverID:   driverID,
			VehicleID:  vehicleID,
			Start:      start,
			AssignedAt: start,
			Type:       assignmentTypeDriverApp,
		}
		if end.After(now) {
			interval.End = nil
		} else {
			endCopy := end
			interval.End = &endCopy
		}
		out = append(out, subtractAPIAssignments(snapshot, &interval, now)...)
	}
	return out
}

func subtractAPIAssignments(
	snapshot *fleetSnapshot,
	interval *assignmentInterval,
	now time.Time,
) []assignmentInterval {
	pieces := []assignmentInterval{*interval}
	for idx := range snapshot.assignments {
		assignment := &snapshot.assignments[idx]
		if assignment.IsPassenger {
			continue
		}
		if assignment.VehicleID != interval.VehicleID && assignment.DriverID != interval.DriverID {
			continue
		}
		if assignment.DriverID == interval.DriverID && assignment.VehicleID == interval.VehicleID {
			continue
		}
		cutStart := assignment.Start
		cutEnd := now.Add(assignmentGenerationPadding)
		if assignment.End != nil {
			cutEnd = *assignment.End
		}
		next := make([]assignmentInterval, 0, len(pieces)+1)
		for pieceIdx := range pieces {
			next = append(next, cutInterval(&pieces[pieceIdx], cutStart, cutEnd, now)...)
		}
		pieces = next
	}
	return pieces
}

func cutInterval(
	piece *assignmentInterval,
	cutStart time.Time,
	cutEnd time.Time,
	now time.Time,
) []assignmentInterval {
	pieceEnd := now
	if piece.End != nil {
		pieceEnd = *piece.End
	}
	if !cutStart.Before(pieceEnd) || !cutEnd.After(piece.Start) {
		return []assignmentInterval{*piece}
	}
	out := make([]assignmentInterval, 0, 2)
	if cutStart.After(piece.Start) {
		before := *piece
		end := cutStart
		before.End = &end
		out = append(out, before)
	}
	if cutEnd.Before(pieceEnd) {
		after := *piece
		after.Start = cutEnd
		after.AssignedAt = cutEnd
		out = append(out, after)
	}
	return out
}

func recordTime(record Record, key string) time.Time {
	at, err := parseRFC3339(stringValue(record, key))
	if err != nil {
		return time.Time{}
	}
	return at
}

func laterOf(left, right time.Time) time.Time {
	if left.After(right) {
		return left
	}
	return right
}

func (v *fleetView) assignmentIntervals(query *assignmentQuery) []assignmentInterval {
	windowStart, windowEnd := v.now, v.now
	if query.WindowStart != nil {
		windowStart = *query.WindowStart
	}
	if query.WindowEnd != nil {
		windowEnd = *query.WindowEnd
	}
	out := make([]assignmentInterval, 0, 32)
	if query.DriverApp {
		out = v.appendDerivedIntervals(out, query, windowStart, windowEnd)
	}
	if query.API {
		out = v.appendAPIIntervals(out, query, windowStart, windowEnd)
	}
	if len(query.Types) > 0 {
		out = slices.DeleteFunc(out, func(interval assignmentInterval) bool {
			_, keep := query.Types[interval.Type]
			return !keep
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Start.Equal(out[j].Start) {
			return out[i].Start.Before(out[j].Start)
		}
		if out[i].DriverID != out[j].DriverID {
			return out[i].DriverID < out[j].DriverID
		}
		return out[i].VehicleID < out[j].VehicleID
	})
	return out
}

func (v *fleetView) appendDerivedIntervals(
	out []assignmentInterval,
	query *assignmentQuery,
	windowStart time.Time,
	windowEnd time.Time,
) []assignmentInterval {
	for _, driver := range v.snap.drivers {
		driverID := recordID(driver)
		if !inOptionalSet(query.DriverIDs, driverID) {
			continue
		}
		derived := v.live.derivedDriverAppAssignments(
			v.snap,
			driverID,
			windowStart,
			windowEnd,
			v.now,
		)
		for idx := range derived {
			interval := &derived[idx]
			if inOptionalSet(query.VehicleIDs, interval.VehicleID) &&
				interval.overlaps(windowStart, windowEnd, v.now) {
				out = append(out, *interval)
			}
		}
	}
	return out
}

func (v *fleetView) appendAPIIntervals(
	out []assignmentInterval,
	query *assignmentQuery,
	windowStart time.Time,
	windowEnd time.Time,
) []assignmentInterval {
	unbounded := query.SourceName != "" && query.WindowStart == nil && query.WindowEnd == nil
	for idx := range v.snap.assignments {
		assignment := &v.snap.assignments[idx]
		if !inOptionalSet(query.DriverIDs, assignment.DriverID) ||
			!inOptionalSet(query.VehicleIDs, assignment.VehicleID) ||
			(query.SourceName != "" && assignment.SourceName != query.SourceName) {
			continue
		}
		interval := assignmentInterval{
			DriverID:    assignment.DriverID,
			VehicleID:   assignment.VehicleID,
			Start:       assignment.Start,
			End:         assignment.End,
			AssignedAt:  assignment.AssignedAt,
			IsPassenger: assignment.IsPassenger,
			Type:        assignment.Type,
			SourceName:  assignment.SourceName,
		}
		if unbounded || interval.overlaps(windowStart, windowEnd, v.now) {
			out = append(out, interval)
		}
	}
	return out
}

func (a *assignmentInterval) overlaps(windowStart, windowEnd, now time.Time) bool {
	end := now
	if a.End != nil {
		end = *a.End
	}
	if a.End == nil && windowEnd.After(now) {
		end = windowEnd
	}
	return !a.Start.After(windowEnd) && !end.Before(windowStart)
}

func (s *Server) registerAssignmentRoutes() {
	s.mux.HandleFunc("GET /fleet/driver-vehicle-assignments", s.handleAssignmentList)
	s.mux.HandleFunc("POST /fleet/driver-vehicle-assignments", s.handleAssignmentCreate)
	s.mux.HandleFunc("PATCH /fleet/driver-vehicle-assignments", s.handleAssignmentPatch)
	s.mux.HandleFunc("DELETE /fleet/driver-vehicle-assignments", s.handleAssignmentDelete)
	s.mux.HandleFunc("GET /fleet/vehicles/driver-assignments", s.handleVehiclesDriverAssignments)
	s.mux.HandleFunc("GET /fleet/drivers/vehicle-assignments", s.handleDriversVehicleAssignments)
}

func parseOptionalTime(values url.Values, name string) (at time.Time, present bool, err error) {
	raw := strings.TrimSpace(values.Get(name))
	if raw == "" {
		return time.Time{}, false, nil
	}
	parsed, parseErr := parseRFC3339(raw)
	if parseErr != nil {
		return time.Time{}, false, invalidParameter(name, "must be an RFC 3339 timestamp")
	}
	return parsed, true, nil
}

func parseOptionalTimePtr(values url.Values, name string) (*time.Time, error) {
	at, present, err := parseOptionalTime(values, name)
	if err != nil || !present {
		return nil, err
	}
	return &at, nil
}

func parseAssignmentWindow(
	values url.Values,
	maxRange time.Duration,
) (start, end *time.Time, err error) {
	start, err = parseOptionalTimePtr(values, fieldStartTime)
	if err != nil {
		return nil, nil, err
	}
	end, err = parseOptionalTimePtr(values, fieldEndTime)
	if err != nil {
		return nil, nil, err
	}
	if start != nil && end != nil && end.Before(*start) {
		return nil, nil, invalidParameter(fieldEndTime, "must not be before startTime")
	}
	if maxRange > 0 && start != nil && end != nil && end.Sub(*start) > maxRange {
		return nil, nil, invalidParameter(
			fieldEndTime,
			fmt.Sprintf("must be within %d days of startTime", int(maxRange/(24*time.Hour))),
		)
	}
	return start, end, nil
}

func (s *Server) handleAssignmentList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	view := s.fleetView()
	query, err := view.parseAssignmentListQuery(values)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	intervals := view.assignmentIntervals(query)
	records := make([]Record, 0, len(intervals))
	for idx := range intervals {
		records = append(records, view.assignmentV2Record(&intervals[idx]))
	}
	s.respondPage(writer, request, records, "|driver-vehicle-assignments")
}

type assignmentListFilters struct {
	FilterBy       string
	DriverRefs     []string
	VehicleRefs    []string
	DriverTags     []string
	VehicleTags    []string
	SourceName     string
	SourceNameSet  bool
	AssignmentType string
}

func readAssignmentListFilters(values url.Values) (assignmentListFilters, error) {
	filters := assignmentListFilters{
		FilterBy:       strings.TrimSpace(values.Get("filterBy")),
		DriverRefs:     csvQueryValues(values, "driverIds"),
		VehicleRefs:    csvQueryValues(values, "vehicleIds"),
		DriverTags:     csvQueryValues(values, "driverTagIds"),
		VehicleTags:    csvQueryValues(values, "vehicleTagIds"),
		SourceName:     strings.TrimSpace(values.Get(fieldSourceName)),
		SourceNameSet:  values.Has(fieldSourceName),
		AssignmentType: strings.TrimSpace(values.Get(fieldAssignmentType)),
	}
	if filters.FilterBy != filterByDrivers && filters.FilterBy != filterByVehicles {
		return filters, invalidParameter(
			"filterBy",
			"is required and must be `drivers` or `vehicles`",
		)
	}
	driverFiltered := len(filters.DriverRefs) > 0 || len(filters.DriverTags) > 0
	vehicleFiltered := len(filters.VehicleRefs) > 0 || len(filters.VehicleTags) > 0
	if filters.FilterBy == filterByDrivers && vehicleFiltered {
		return filters, invalidParameter("vehicleIds", "can only be used with filterBy=vehicles")
	}
	if filters.FilterBy == filterByVehicles && driverFiltered {
		return filters, invalidParameter("driverIds", "can only be used with filterBy=drivers")
	}
	if filters.AssignmentType != "" &&
		!slices.Contains(assignmentTypeFilterValues, filters.AssignmentType) {
		return filters, invalidParameter(
			fieldAssignmentType,
			"must be one of "+strings.Join(quoteAll(assignmentTypeFilterValues), ", "),
		)
	}
	err := validateSourceNameFilter(&filters, driverFiltered)
	return filters, err
}

func validateSourceNameFilter(filters *assignmentListFilters, driverFiltered bool) error {
	if !filters.SourceNameSet {
		return nil
	}
	if filters.SourceName == "" || len(filters.SourceName) > maxAssignmentSourceNameLength {
		return invalidParameter(fieldSourceName, "must be 1-100 characters")
	}
	if filters.FilterBy != filterByDrivers {
		return invalidParameter(fieldSourceName, "requires filterBy=drivers")
	}
	if driverFiltered || filters.AssignmentType != "" {
		return invalidParameter(
			fieldSourceName,
			"cannot be combined with driver, vehicle, tag or assignment type filters",
		)
	}
	return nil
}

func (v *fleetView) parseAssignmentListQuery(values url.Values) (*assignmentQuery, error) {
	filters, err := readAssignmentListFilters(values)
	if err != nil {
		return nil, err
	}
	start, end, err := parseAssignmentWindow(values, 0)
	if err != nil {
		return nil, err
	}
	query := &assignmentQuery{
		WindowStart: start,
		WindowEnd:   end,
		DriverApp:   filters.SourceName == "",
		API:         true,
		SourceName:  filters.SourceName,
	}
	if filters.AssignmentType != "" {
		query.Types = map[string]struct{}{filters.AssignmentType: {}}
	}
	if len(filters.DriverRefs) > 0 || len(filters.DriverTags) > 0 {
		query.DriverIDs = v.selectDrivers(filters.DriverRefs, filters.DriverTags, nil, true)
	}
	if len(filters.VehicleRefs) > 0 || len(filters.VehicleTags) > 0 {
		query.VehicleIDs = v.selectVehicles(filters.VehicleRefs, filters.VehicleTags, nil, true)
	}
	return query, nil
}

func (v *fleetView) selectDrivers(
	refs []string,
	tagIDs []string,
	parentTagIDs []string,
	externalRefs bool,
) map[string]struct{} {
	out := map[string]struct{}{}
	filter := v.snap.tags.filter(tagIDs, parentTagIDs)
	var allowed map[string]struct{}
	if len(refs) > 0 {
		auto := autoExternalIDs(nil)
		if !externalRefs {
			allowed = resolvePlainIDs(refs)
		} else {
			allowed = resolveRecordRefs(v.snap.drivers, refs, auto)
		}
	}
	for _, driver := range v.snap.drivers {
		id := recordID(driver)
		if allowed != nil {
			if _, ok := allowed[id]; !ok {
				continue
			}
		}
		if !filter.matches(v.snap.tags, tagMembersDrivers, id) {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}

func (v *fleetView) selectVehicles(
	refs []string,
	tagIDs []string,
	parentTagIDs []string,
	externalRefs bool,
) map[string]struct{} {
	out := map[string]struct{}{}
	filter := v.snap.tags.filter(tagIDs, parentTagIDs)
	vehicles := v.vehicleRecords()
	var allowed map[string]struct{}
	if len(refs) > 0 {
		if externalRefs {
			allowed = resolveRecordRefs(vehicles, refs, vehicleAutoExternalIDs)
		} else {
			allowed = resolvePlainIDs(refs)
		}
	}
	for _, vehicle := range vehicles {
		id := recordID(vehicle)
		if allowed != nil {
			if _, ok := allowed[id]; !ok {
				continue
			}
		}
		if !filter.matches(v.snap.tags, tagMembersVehicles, id) {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}

func resolvePlainIDs(refs []string) map[string]struct{} {
	out := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		out[strings.TrimSpace(ref)] = struct{}{}
	}
	return out
}

func (v *fleetView) assignmentV2Record(interval *assignmentInterval) Record {
	record := Record{
		fieldAssignedAtTime: interval.AssignedAt.UTC().Format(time.RFC3339),
		fieldAssignmentType: interval.Type,
		keyDriver:           v.driverRef(interval.DriverID),
		fieldIsPassenger:    interval.IsPassenger,
		fieldStartTime:      interval.Start.UTC().Format(time.RFC3339),
		keyVehicle:          v.vehicleRef(interval.VehicleID),
	}
	if interval.End != nil {
		record[fieldEndTime] = interval.End.UTC().Format(time.RFC3339)
	}
	if interval.SourceName != "" {
		record[fieldMetadata] = map[string]any{fieldSourceName: interval.SourceName}
	}
	return record
}

func (s *Server) handleVehiclesDriverAssignments(
	writer http.ResponseWriter,
	request *http.Request,
) {
	values := request.URL.Query()
	start, end, err := parseAssignmentWindow(values, legacyAssignmentMaxRange)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	vehicleIDs := view.selectVehicles(
		csvQueryValues(values, "vehicleIds"),
		tagIDs,
		parentTagIDs,
		true,
	)
	intervals := view.assignmentIntervals(&assignmentQuery{
		WindowStart: start,
		WindowEnd:   end,
		VehicleIDs:  vehicleIDs,
		DriverApp:   true,
	})
	byVehicle := make(map[string][]any, len(vehicleIDs))
	for idx := range intervals {
		interval := &intervals[idx]
		entry := map[string]any{
			fieldAssignmentType: assignmentTypeDriverApp,
			keyDriver:           v2DriverRef(view, interval.DriverID),
			fieldIsPassenger:    interval.IsPassenger,
			fieldStartTime:      interval.Start.UTC().Format(time.RFC3339),
		}
		if interval.End != nil {
			entry[fieldEndTime] = interval.End.UTC().Format(time.RFC3339)
		}
		byVehicle[interval.VehicleID] = append(byVehicle[interval.VehicleID], entry)
	}
	records := make([]Record, 0, len(vehicleIDs))
	for _, vehicle := range view.vehicleRecords() {
		id := recordID(vehicle)
		if _, ok := vehicleIDs[id]; !ok {
			continue
		}
		assignments := byVehicle[id]
		if assignments == nil {
			assignments = []any{}
		}
		record := Record{
			keyID:               id,
			keyName:             stringValue(vehicle, keyName),
			fieldExternalIDs:    renderExternalIDs(vehicle, vehicleAutoExternalIDs),
			"driverAssignments": assignments,
		}
		records = append(records, record)
	}
	s.respondPage(writer, request, records, "|vehicles-driver-assignments")
}

func (s *Server) handleDriversVehicleAssignments(
	writer http.ResponseWriter,
	request *http.Request,
) {
	values := request.URL.Query()
	start, end, err := parseAssignmentWindow(values, legacyAssignmentMaxRange)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	status, err := parseActivationStatus(values)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	driverIDs := view.selectDrivers(csvQueryValues(values, "driverIds"), tagIDs, parentTagIDs, true)
	intervals := view.assignmentIntervals(&assignmentQuery{
		WindowStart: start,
		WindowEnd:   end,
		DriverIDs:   driverIDs,
		DriverApp:   true,
	})
	byDriver := make(map[string][]any, len(driverIDs))
	for idx := range intervals {
		interval := &intervals[idx]
		entry := map[string]any{
			fieldAssignmentType: assignmentTypeDriverApp,
			fieldIsPassenger:    interval.IsPassenger,
			fieldStartTime:      interval.Start.UTC().Format(time.RFC3339),
			keyVehicle:          view.vehicleRef(interval.VehicleID),
		}
		if interval.End != nil {
			entry[fieldEndTime] = interval.End.UTC().Format(time.RFC3339)
		}
		byDriver[interval.DriverID] = append(byDriver[interval.DriverID], entry)
	}
	records := make([]Record, 0, len(driverIDs))
	for _, driver := range view.snap.drivers {
		id := recordID(driver)
		if _, ok := driverIDs[id]; !ok || driverStatus(driver) != status {
			continue
		}
		assignments := byDriver[id]
		if assignments == nil {
			assignments = []any{}
		}
		records = append(records, Record{
			fieldDriverActivationStatus: driverStatus(driver),
			fieldExternalIDs:            renderExternalIDs(driver, nil),
			keyID:                       id,
			keyName:                     stringValue(driver, keyName),
			"vehicleAssignments":        assignments,
		})
	}
	s.respondPage(writer, request, records, "|drivers-vehicle-assignments")
}

func v2DriverRef(view *fleetView, driverID string) map[string]any {
	return view.driverRef(driverID)
}

func assignmentBodyRules(mode bodyMode) []fieldRule {
	required := mode == bodyCreate
	return []fieldRule{
		{Name: fieldDriverID, Kind: kindString, Required: required, MinLen: 1},
		{Name: fieldVehicleID, Kind: kindString, Required: required, MinLen: 1},
		{Name: fieldStartTime, Kind: kindTime},
		{Name: fieldEndTime, Kind: kindTime, Nullable: mode == bodyPatch},
		{Name: fieldAssignedAtTime, Kind: kindTime},
		{Name: fieldIsPassenger, Kind: kindBool},
		{
			Name: fieldMetadata,
			Kind: kindObject,
			Fields: []fieldRule{
				{
					Name:   fieldSourceName,
					Kind:   kindString,
					MinLen: 1,
					MaxLen: maxAssignmentSourceNameLength,
				},
			},
		},
	}
}

func (s *Server) handleAssignmentCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, assignmentBodyRules(bodyCreate), bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	err = s.store.Transact(func(tx *storeTx) error {
		return createAssignmentTx(tx, sanitized, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	payload := map[string]any{keyData: map[string]any{keyMessage: assignmentSubmittedMessage}}
	s.respondJSONStatus(
		writer,
		request,
		http.StatusCreated,
		requestSignature(request)+"|assignment-create",
		payload,
	)
}

func createAssignmentTx(tx *storeTx, body Record, now time.Time) error {
	driverID, vehicleID, err := resolveAssignmentSubjectsTx(
		tx,
		stringValue(body, fieldDriverID),
		stringValue(body, fieldVehicleID),
	)
	if err != nil {
		return err
	}
	record := Record{
		fieldDriverID:       driverID,
		fieldVehicleID:      vehicleID,
		fieldStartTime:      timeOrDefault(body, fieldStartTime, now),
		fieldAssignedAtTime: timeOrDefault(body, fieldAssignedAtTime, now),
		fieldIsPassenger:    body[fieldIsPassenger] == true,
		fieldAssignmentType: assignmentTypeExternal,
	}
	if end, ok := body[fieldEndTime].(string); ok {
		if end <= stringValue(record, fieldStartTime) {
			return invalidField(fieldEndTime, "must be after startTime")
		}
		record[fieldEndTime] = end
	}
	if metadata, ok := anyAsMap(body[fieldMetadata]); ok && len(metadata) > 0 {
		record[fieldMetadata] = metadata
	}
	if err = ensureAssignmentUniqueTx(tx, "", record); err != nil {
		return err
	}
	_, err = tx.insert(ResourceDriverVehicleAssignments, record, now)
	return err
}

func timeOrDefault(body Record, key string, fallback time.Time) string {
	if value, ok := body[key].(string); ok && value != "" {
		return value
	}
	return fallback.UTC().Format(time.RFC3339)
}

func resolveAssignmentSubjectsTx(
	tx *storeTx,
	driverRef string,
	vehicleRef string,
) (driverID, vehicleID string, err error) {
	driver, idx := findRecordByRef(tx.records(ResourceDrivers), driverRef, nil)
	if idx < 0 {
		return "", "", notFound(keyDriver, driverRef)
	}
	if driverDeactivated(driver) {
		return "", "", invalidField(fieldDriverID, "refers to a deactivated driver")
	}
	vehicle, vehicleIdx := findVehicleTx(tx, vehicleRef)
	if vehicleIdx < 0 {
		return "", "", notFound(keyVehicle, vehicleRef)
	}
	return recordID(driver), recordID(vehicle), nil
}

func findVehicleTx(tx *storeTx, ref string) (vehicle Record, index int) {
	vehicles := make([]Record, 0, len(tx.records(ResourceAssets)))
	for _, asset := range tx.records(ResourceAssets) {
		if stringValue(asset, keyType) == assetTypeVehicle {
			vehicles = append(vehicles, asset)
		}
	}
	return findRecordByRef(vehicles, ref, vehicleAutoExternalIDs)
}

func ensureAssignmentUniqueTx(tx *storeTx, selfID string, candidate Record) error {
	for _, existing := range tx.records(ResourceDriverVehicleAssignments) {
		if recordID(existing) == selfID {
			continue
		}
		if stringValue(existing, fieldDriverID) == stringValue(candidate, fieldDriverID) &&
			stringValue(existing, fieldVehicleID) == stringValue(candidate, fieldVehicleID) &&
			stringValue(existing, fieldStartTime) == stringValue(candidate, fieldStartTime) {
			return fmt.Errorf(
				"%w: an assignment of driver %s to vehicle %s starting at %s already exists",
				ErrUniqueConflict,
				stringValue(candidate, fieldDriverID),
				stringValue(candidate, fieldVehicleID),
				stringValue(candidate, fieldStartTime),
			)
		}
	}
	return nil
}

func (s *Server) handleAssignmentPatch(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, assignmentBodyRules(bodyPatch), bodyPatch)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	err = s.store.Transact(func(tx *storeTx) error {
		return patchAssignmentTx(tx, sanitized, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	payload := map[string]any{keyData: map[string]any{keyMessage: assignmentUpdatedMessage}}
	s.respondJSONStatus(
		writer,
		request,
		http.StatusAccepted,
		requestSignature(request)+"|assignment-patch",
		payload,
	)
}

func patchAssignmentTx(tx *storeTx, body Record, now time.Time) error {
	target, err := findAssignmentForPatchTx(tx, body)
	if err != nil {
		return err
	}
	_, err = tx.update(
		ResourceDriverVehicleAssignments,
		recordID(target),
		func(record Record) error {
			return applyAssignmentPatch(record, body)
		},
		now,
	)
	return err
}

func applyAssignmentPatch(record, body Record) error {
	if raw, ok := body[fieldEndTime]; ok {
		if raw == nil {
			delete(record, fieldEndTime)
		} else {
			if stringOf(raw) <= stringValue(record, fieldStartTime) {
				return invalidField(fieldEndTime, "must be after startTime")
			}
			record[fieldEndTime] = raw
		}
	}
	if value, ok := body[fieldIsPassenger].(bool); ok {
		record[fieldIsPassenger] = value
	}
	if value, ok := body[fieldAssignedAtTime].(string); ok {
		record[fieldAssignedAtTime] = value
	}
	if identityProvided(body) {
		if metadata, ok := anyAsMap(body[fieldMetadata]); ok && len(metadata) > 0 {
			record[fieldMetadata] = metadata
		}
	}
	return nil
}

func identityProvided(body Record) bool {
	_, hasDriver := body[fieldDriverID]
	_, hasVehicle := body[fieldVehicleID]
	_, hasStart := body[fieldStartTime]
	return hasDriver || hasVehicle || hasStart
}

func findAssignmentForPatchTx(tx *storeTx, body Record) (Record, error) {
	records := tx.records(ResourceDriverVehicleAssignments)
	if identityProvided(body) {
		driverRef := stringValue(body, fieldDriverID)
		vehicleRef := stringValue(body, fieldVehicleID)
		startTime := stringValue(body, fieldStartTime)
		if driverRef == "" || vehicleRef == "" || startTime == "" {
			return nil, invalidField(
				"vehicleId, driverId and startTime",
				"are required together to identify the assignment",
			)
		}
		driver, idx := findRecordByRef(tx.records(ResourceDrivers), driverRef, nil)
		if idx < 0 {
			return nil, notFound(keyDriver, driverRef)
		}
		vehicle, vehicleIdx := findVehicleTx(tx, vehicleRef)
		if vehicleIdx < 0 {
			return nil, notFound(keyVehicle, vehicleRef)
		}
		for _, record := range records {
			if stringValue(record, fieldDriverID) == recordID(driver) &&
				stringValue(record, fieldVehicleID) == recordID(vehicle) &&
				stringValue(record, fieldStartTime) == startTime {
				return record, nil
			}
		}
		return nil, notFound("driver assignment", driverRef+"/"+vehicleRef+"@"+startTime)
	}
	sourceName := nestedString(body, fieldMetadata, fieldSourceName)
	if sourceName == "" {
		return nil, invalidField(
			fieldMetadata+"."+fieldSourceName,
			"is required when vehicleId, driverId and startTime are omitted",
		)
	}
	var match Record
	for _, record := range records {
		if nestedString(record, fieldMetadata, fieldSourceName) != sourceName {
			continue
		}
		if match != nil {
			return nil, invalidField(
				fieldMetadata+"."+fieldSourceName,
				"matches more than one assignment; identify it with vehicleId, driverId and startTime",
			)
		}
		match = record
	}
	if match == nil {
		return nil, notFound("driver assignment with source name", sourceName)
	}
	return match, nil
}

func (s *Server) handleAssignmentDelete(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	rules := []fieldRule{
		{Name: fieldVehicleID, Kind: kindString, Required: true, MinLen: 1},
		{Name: fieldStartTime, Kind: kindTime},
		{Name: fieldEndTime, Kind: kindTime},
		{Name: fieldAssignedAtTime, Kind: kindTime},
		{Name: fieldIsPassenger, Kind: kindBool},
	}
	sanitized, err := sanitizeBody(body, rules, bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	start := stringValue(sanitized, fieldStartTime)
	end := stringValue(sanitized, fieldEndTime)
	if start != "" && end != "" && end < start {
		s.writeError(writer, invalidField(fieldEndTime, "must not be before startTime"))
		return
	}
	err = s.store.Transact(func(tx *storeTx) error {
		vehicleRef := stringValue(sanitized, fieldVehicleID)
		vehicle, idx := findVehicleTx(tx, vehicleRef)
		if idx < 0 {
			return notFound(keyVehicle, vehicleRef)
		}
		records := tx.records(ResourceDriverVehicleAssignments)
		kept := make([]Record, 0, len(records))
		for _, record := range records {
			if !assignmentMatchesDelete(record, recordID(vehicle), sanitized) {
				kept = append(kept, record)
			}
		}
		if len(kept) == len(records) {
			return nil
		}
		return tx.replace(ResourceDriverVehicleAssignments, kept)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func assignmentMatchesDelete(record Record, vehicleID string, filter Record) bool {
	if stringValue(record, fieldVehicleID) != vehicleID {
		return false
	}
	if value, ok := filter[fieldIsPassenger].(bool); ok &&
		(record[fieldIsPassenger] == true) != value {
		return false
	}
	if value := stringValue(filter, fieldAssignedAtTime); value != "" &&
		stringValue(record, fieldAssignedAtTime) != value {
		return false
	}
	recordStart := stringValue(record, fieldStartTime)
	recordEnd := stringValue(record, fieldEndTime)
	if windowEnd := stringValue(filter, fieldEndTime); windowEnd != "" && recordStart > windowEnd {
		return false
	}
	if windowStart := stringValue(filter, fieldStartTime); windowStart != "" &&
		recordEnd != "" && recordEnd < windowStart {
		return false
	}
	return true
}
