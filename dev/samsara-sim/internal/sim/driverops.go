package sim

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	efficiencyMaxRange          = 31 * 24 * time.Hour
	efficiencyDefaultRange      = 24 * time.Hour
	tachographActivityMaxRange  = 30 * 24 * time.Hour
	driverAuthTokenTTL          = 10 * time.Minute
	minDriverAuthCodeLength     = 12
	tachographStateDriving      = "DRIVING"
	tachographStateWork         = "WORK"
	tachographStateBreakRest    = "BREAK/REST"
	litersPerGallon             = 3.785411784
	metersPerMile               = 1609.344
	workflowTypeFilterParameter = "workflowType"
)

var driverWorkflowTypes = []string{
	"startOfDay",
	"endOfDay",
	"assetSelection",
	"leaveAsset",
	"ridershipSafetyCheck",
	"stopArrival",
}

func (s *Server) registerDriverOperationRoutes() {
	s.mux.HandleFunc("GET /beta/fleet/drivers/efficiency", s.handleDriverEfficiency)
	s.mux.HandleFunc(
		"GET /fleet/drivers/tachograph-activity/history",
		s.handleDriverTachographActivity,
	)
	s.mux.HandleFunc("POST /fleet/drivers/auth-token", s.handleDriverAuthToken)
	s.mux.HandleFunc("POST /fleet/drivers/remote-sign-out", s.handleDriverRemoteSignOut)
	s.mux.HandleFunc(
		"POST /fleet/drivers/voice-sign-in/resolve-assignment",
		s.handleDriverVoiceSignIn,
	)
	s.mux.HandleFunc("GET /fleet/drivers/workflows", s.handleDriverWorkflowList)
	s.mux.HandleFunc("POST /fleet/drivers/workflow-assignments", s.handleDriverWorkflowAssignment)
}

type efficiencyWindow struct {
	Start time.Time
	End   time.Time
}

func parseEfficiencyWindow(request *http.Request, now time.Time) (efficiencyWindow, error) {
	values := request.URL.Query()
	end := now.Truncate(time.Hour)
	if parsed, present, err := parseOptionalTime(values, fieldEndTime); err != nil {
		return efficiencyWindow{}, err
	} else if present {
		if parsed.After(now) {
			return efficiencyWindow{}, invalidParameter(fieldEndTime, "cannot be in the future")
		}
		end = parsed.Truncate(time.Hour)
	}
	start := end.Add(-efficiencyDefaultRange)
	if parsed, present, err := parseOptionalTime(values, fieldStartTime); err != nil {
		return efficiencyWindow{}, err
	} else if present {
		if parsed.After(now) {
			return efficiencyWindow{}, invalidParameter(fieldStartTime, "cannot be in the future")
		}
		start = parsed.Truncate(time.Hour)
	}
	if end.Before(start) {
		return efficiencyWindow{}, invalidParameter(fieldEndTime, "must not be before startTime")
	}
	if end.Sub(start) > efficiencyMaxRange {
		return efficiencyWindow{}, invalidParameter(
			fieldEndTime,
			"must be within 31 days of startTime",
		)
	}
	return efficiencyWindow{Start: start, End: end}, nil
}

func (s *Server) handleDriverEfficiency(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	now := s.simNow()
	window, err := parseEfficiencyWindow(request, now)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	driverIDs := csvQueryValues(values, "driverIds")
	tagIDs, parentTagIDs := tagFilterParams(values, "driverTagIds", "driverParentTagIds")
	if len(driverIDs) > 0 &&
		(len(tagIDs) > 0 || len(parentTagIDs) > 0 || values.Has(fieldDriverActivationStatus)) {
		s.writeError(writer, invalidParameter(
			"driverIds",
			"cannot be combined with driverTagIds, driverParentTagIds or driverActivationStatus",
		))
		return
	}
	status, err := parseActivationStatus(values)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	selected := view.selectDrivers(driverIDs, tagIDs, parentTagIDs, false)
	summaries := make([]Record, 0, len(selected))
	if window.Start.Before(now.Add(-time.Hour)) {
		for _, driver := range view.snap.drivers {
			id := recordID(driver)
			if _, ok := selected[id]; !ok {
				continue
			}
			if len(driverIDs) == 0 && driverStatus(driver) != status {
				continue
			}
			if summary := view.driverEfficiencySummary(driver, window); summary != nil {
				summaries = append(summaries, summary)
			}
		}
	}
	page, pagination, err := paginate(summaries, request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	payload := map[string]any{
		keyData: map[string]any{
			"driverSummaries":  recordsAsAny(page),
			"summaryStartTime": window.Start.UTC().Format(time.RFC3339),
			"summaryEndTime":   window.End.UTC().Format(time.RFC3339),
		},
		keyPagination: pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|driver-efficiency", payload)
}

type efficiencyTotals struct {
	DriveMs    float64
	IdleMs     float64
	Meters     float64
	FuelMl     float64
	Brakes     float64
	Anticipate float64
	Coasting   float64
	Cruise     float64
	GreenBand  float64
	HighTorque float64
	OverSpeed  float64
	PowerTake  float64
}

func (v *fleetView) driverEfficiencySummary(driver Record, window efficiencyWindow) Record {
	driverID := recordID(driver)
	intervals := v.assignmentIntervals(&assignmentQuery{
		WindowStart: &window.Start,
		WindowEnd:   &window.End,
		DriverIDs:   map[string]struct{}{driverID: {}},
		DriverApp:   true,
		API:         true,
	})
	timeline := v.live.driverTimelineSegments(driverID, window.Start, window.End, v.now)
	byVehicle := map[string]*efficiencyTotals{}
	order := make([]string, 0, 2)
	for idx := range timeline {
		segment := &timeline[idx]
		start := maxTime(segment.Start, window.Start)
		end := minTime(segment.End, minTime(window.End, v.now))
		if !end.After(start) || isRestStatus(segment.Status) {
			continue
		}
		vehicleID := vehicleForInterval(intervals, start, v.snap.baseRoster[driverID].VehicleID)
		if vehicleID == "" {
			continue
		}
		totals, ok := byVehicle[vehicleID]
		if !ok {
			totals = &efficiencyTotals{}
			byVehicle[vehicleID] = totals
			order = append(order, vehicleID)
		}
		v.accumulateEfficiency(totals, driverID, vehicleID, segment.Status, start, end)
	}
	if len(order) == 0 {
		return nil
	}
	sort.Strings(order)
	overall := efficiencyTotals{}
	vehicleSummaries := make([]any, 0, len(order))
	for _, vehicleID := range order {
		totals := byVehicle[vehicleID]
		overall.add(totals)
		summary := totals.vehicleSummary()
		summary[keyVehicle] = v.vehicleTinyCapitalized(vehicleID)
		vehicleSummaries = append(vehicleSummaries, summary)
	}
	if overall.DriveMs <= 0 {
		return nil
	}
	record := overall.driverSummary()
	record[keyDriver] = v.extendedDriverTiny(driver)
	record["vehicleSummaries"] = vehicleSummaries
	return record
}

func vehicleForInterval(intervals []assignmentInterval, at time.Time, fallback string) string {
	for idx := len(intervals) - 1; idx >= 0; idx-- {
		interval := &intervals[idx]
		if interval.IsPassenger || interval.Start.After(at) {
			continue
		}
		if interval.End == nil || interval.End.After(at) {
			return interval.VehicleID
		}
	}
	return fallback
}

func (v *fleetView) accumulateEfficiency(
	totals *efficiencyTotals,
	driverID string,
	vehicleID string,
	status string,
	start time.Time,
	end time.Time,
) {
	durationMs := float64(end.Sub(start)) / float64(time.Millisecond)
	key := driverID + "|" + vehicleID + "|" + start.UTC().Format(time.RFC3339)
	if status != hosStatusDriving {
		idleShare := 0.25 + 0.2*v.live.hashFraction("efficiency-idle", key)
		totals.IdleMs += durationMs * idleShare
		if vehicleHasAuxInput(v.snap.assetByID[vehicleID], auxInputPowerTakeOff) {
			totals.PowerTake += durationMs * idleShare * (0.3 + 0.2*v.live.hashFraction("pto", key))
		}
		return
	}
	speed := v.vehicleCruiseSpeedMPS(
		vehicleID,
	) * (0.9 + 0.12*v.live.hashFraction("efficiency-speed", key))
	meters := speed * durationMs / 1000
	mpg := 6.1 + 1.3*v.live.hashFraction("efficiency-mpg", vehicleID)
	totals.DriveMs += durationMs
	totals.Meters += meters
	totals.FuelMl += meters / metersPerMile / mpg * litersPerGallon * 1000
	brakes := meters / 1000 * (0.12 + 0.08*v.live.hashFraction("efficiency-brakes", key))
	totals.Brakes += math.Round(brakes)
	totals.Anticipate += math.Round(brakes * (0.1 + 0.1*v.live.hashFraction("anticipation", key)))
	totals.Coasting += durationMs * (0.08 + 0.07*v.live.hashFraction("coasting", key))
	totals.Cruise += durationMs * (0.35 + 0.25*v.live.hashFraction("cruise", key))
	totals.GreenBand += durationMs * (0.7 + 0.15*v.live.hashFraction("green-band", key))
	totals.HighTorque += durationMs * (0.03 + 0.05*v.live.hashFraction("torque", key))
	totals.OverSpeed += durationMs * (0.01 + 0.03*v.live.hashFraction("over-speed", key))
}

func (v *fleetView) vehicleCruiseSpeedMPS(vehicleID string) float64 {
	geometry, ok := v.snap.geometries[vehicleID]
	if !ok || geometry.Period <= 0 {
		return defaultAssetSpeedMPS * 1.6
	}
	total := 0.0
	for idx := range geometry.Segments {
		total += geometry.Segments[idx].DistanceMeters
	}
	return clampFloat64(total/geometry.Period.Seconds(), minRouteSpeedMPS, maxRouteSpeedMPS)
}

func vehicleHasAuxInput(vehicle Record, inputType string) bool {
	for index := 1; index <= vehicleAuxInputCount; index++ {
		if stringValue(vehicle, "auxInputType"+strconv.Itoa(index)) == inputType {
			return true
		}
	}
	return false
}

func (t *efficiencyTotals) add(other *efficiencyTotals) {
	t.DriveMs += other.DriveMs
	t.IdleMs += other.IdleMs
	t.Meters += other.Meters
	t.FuelMl += other.FuelMl
	t.Brakes += other.Brakes
	t.Anticipate += other.Anticipate
	t.Coasting += other.Coasting
	t.Cruise += other.Cruise
	t.GreenBand += other.GreenBand
	t.HighTorque += other.HighTorque
	t.OverSpeed += other.OverSpeed
	t.PowerTake += other.PowerTake
}

func (t *efficiencyTotals) vehicleSummary() Record {
	return Record{
		"anticipationBrakeEventCount": t.Anticipate,
		"coastingDurationMs":          math.Round(t.Coasting),
		"cruiseControlDurationMs":     math.Round(t.Cruise),
		"distanceDrivenMeters":        math.Round(t.Meters),
		"driveTimeDurationMs":         math.Round(t.DriveMs),
		"fuelConsumedMl":              math.Round(t.FuelMl),
		"greenBandDrivingDurationMs":  math.Round(t.GreenBand),
		"highTorqueMs":                math.Round(t.HighTorque),
		"idleTimeDurationMs":          math.Round(t.IdleMs),
		"overSpeedMs":                 math.Round(t.OverSpeed),
		"powerTakeOffDurationMs":      math.Round(t.PowerTake),
		"totalBrakeEventCount":        t.Brakes,
	}
}

func (t *efficiencyTotals) driverSummary() Record {
	return Record{
		"anticipationBrakeEventCount": t.Anticipate,
		"coastingDurationMs":          math.Round(t.Coasting),
		"cruiseControlDurationMs":     math.Round(t.Cruise),
		"greenBandDrivingDurationMs":  math.Round(t.GreenBand),
		"highTorqueMs":                math.Round(t.HighTorque),
		"overSpeedMs":                 math.Round(t.OverSpeed),
		"totalBrakeEventCount":        t.Brakes,
		"totalDistanceDrivenMeters":   math.Round(t.Meters),
		"totalDriveTimeDurationMs":    math.Round(t.DriveMs),
		"totalFuelConsumedMl":         math.Round(t.FuelMl),
		"totalIdleTimeDurationMs":     math.Round(t.IdleMs),
		"totalPowerTakeOffDurationMs": math.Round(t.PowerTake),
	}
}

func (v *fleetView) extendedDriverTiny(driver Record) map[string]any {
	out := map[string]any{
		keyID:      recordID(driver),
		keyName:    stringValue(driver, keyName),
		"username": stringValue(driver, fieldUsername),
	}
	if externalIDs := externalIDsOf(driver); len(externalIDs) > 0 {
		out[fieldExternalIDs] = renderExternalIDs(driver, nil)
	}
	return out
}

func (v *fleetView) vehicleTinyCapitalized(vehicleID string) map[string]any {
	out := map[string]any{keyID: vehicleID}
	if vehicle, ok := v.snap.assetByID[vehicleID]; ok {
		out[keyName] = stringValue(vehicle, keyName)
		out["ExternalIds"] = renderExternalIDs(vehicle, vehicleAutoExternalIDs)
	}
	return out
}

func (s *Server) handleDriverTachographActivity(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	start, hasStart, err := parseOptionalTime(values, fieldStartTime)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	end, hasEnd, err := parseOptionalTime(values, fieldEndTime)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if !hasStart || !hasEnd {
		s.writeError(writer, ErrTimeRangeRequired)
		return
	}
	if end.Before(start) {
		s.writeError(writer, invalidParameter(fieldEndTime, "must not be before startTime"))
		return
	}
	if end.Sub(start) > tachographActivityMaxRange {
		s.writeError(
			writer,
			invalidParameter(fieldEndTime, "can't be more than 30 days past startTime"),
		)
		return
	}
	view := s.fleetView()
	tagIDs, parentTagIDs := standardTagParams(values)
	selected := view.selectDrivers(csvQueryValues(values, "driverIds"), tagIDs, parentTagIDs, false)
	windowEnd := minTime(end, view.now)
	records := make([]Record, 0, len(selected))
	for _, driver := range view.snap.drivers {
		id := recordID(driver)
		if _, ok := selected[id]; !ok || stringValue(driver, keyTachographCard) == "" {
			continue
		}
		records = append(records, Record{
			keyDriver:  view.driverTiny(id),
			"activity": view.tachographActivity(id, start, windowEnd),
		})
	}
	s.respondPage(writer, request, records, "|driver-tachograph-activity")
}

func (v *fleetView) tachographActivity(driverID string, start, end time.Time) []any {
	out := make([]any, 0, 16)
	if !end.After(start) {
		return out
	}
	timeline := v.live.driverTimelineSegments(driverID, start, end, v.now)
	var current map[string]any
	for idx := range timeline {
		segment := &timeline[idx]
		segmentStart := maxTime(segment.Start, start)
		segmentEnd := minTime(segment.End, end)
		if !segmentEnd.After(segmentStart) {
			continue
		}
		state := tachographState(segment.Status)
		manual := state == tachographStateWork &&
			v.live.hashFraction(
				"tacho-manual",
				driverID,
				segment.Start.UTC().Format(time.RFC3339),
			) < 0.05
		if current != nil && current[keyState] == state && current["isManualEntry"] == manual {
			current[fieldEndTime] = segmentEnd.UTC().Format(time.RFC3339)
			continue
		}
		current = map[string]any{
			fieldStartTime:  segmentStart.UTC().Format(time.RFC3339),
			fieldEndTime:    segmentEnd.UTC().Format(time.RFC3339),
			keyState:        state,
			"isManualEntry": manual,
		}
		out = append(out, current)
	}
	return out
}

func tachographState(status string) string {
	switch status {
	case hosStatusDriving:
		return tachographStateDriving
	case hosStatusOnDuty:
		return tachographStateWork
	default:
		return tachographStateBreakRest
	}
}

func (s *Server) handleDriverAuthToken(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	rules := []fieldRule{
		{Name: "code", Kind: kindString, Required: true, MinLen: minDriverAuthCodeLength},
		{Name: fieldDriverID, Kind: kindInt, IntRange: &intRange{Min: 1, Max: math.MaxInt64 >> 10}},
		{Name: "externalId", Kind: kindString, MinLen: 3},
		{Name: fieldUsername, Kind: kindString, MinLen: 1},
	}
	sanitized, err := sanitizeBody(body, rules, bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	driver, err := view.resolveAuthTokenDriver(sanitized)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := view.now
	expiration := now.Add(driverAuthTokenTTL)
	digest := sha256.Sum256([]byte(strings.Join([]string{
		"driver-auth-token",
		recordID(driver),
		stringValue(sanitized, "code"),
		strconv.FormatInt(now.UnixNano(), 10),
		strconv.FormatUint(s.requestSeq.Add(1), 10),
	}, "|")))
	payload := map[string]any{
		keyData: map[string]any{
			"token":          base64.RawURLEncoding.EncodeToString(digest[:]),
			"expirationTime": expiration.UnixMilli(),
		},
	}
	s.respondJSON(writer, request, requestSignature(request)+"|driver-auth-token", payload)
}

func (v *fleetView) resolveAuthTokenDriver(body Record) (Record, error) {
	refs := make([]string, 0, 3)
	if id, ok := body[fieldDriverID].(int64); ok {
		refs = append(refs, strconv.FormatInt(id, 10))
	}
	if externalID, ok := body["externalId"].(string); ok {
		ref := parseRecordRef(externalID)
		if !ref.isExternal() || ref.Value == "" {
			return nil, invalidField("externalId", "must use the key:value format")
		}
		refs = append(refs, externalID)
	}
	username, hasUsername := body[fieldUsername].(string)
	if len(refs) == 0 && !hasUsername {
		return nil, invalidField("driverId, externalId or username", "one is required")
	}
	var resolved Record
	for _, ref := range refs {
		driver, idx := findRecordByRef(v.snap.drivers, ref, nil)
		if idx < 0 {
			return nil, notFound(keyDriver, ref)
		}
		if resolved != nil && recordID(resolved) != recordID(driver) {
			return nil, invalidField(
				"driverId, externalId and username",
				"must identify the same driver",
			)
		}
		resolved = driver
	}
	if hasUsername {
		var match Record
		for _, driver := range v.snap.drivers {
			if strings.EqualFold(stringValue(driver, fieldUsername), username) {
				match = driver
				break
			}
		}
		if match == nil {
			return nil, notFound("driver with username", username)
		}
		if resolved != nil && recordID(resolved) != recordID(match) {
			return nil, invalidField(
				"driverId, externalId and username",
				"must identify the same driver",
			)
		}
		resolved = match
	}
	if driverDeactivated(resolved) {
		return nil, invalidField(keyDriver, "is deactivated and cannot sign in")
	}
	return resolved, nil
}

func (s *Server) handleDriverRemoteSignOut(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, []fieldRule{
		{Name: fieldDriverID, Kind: kindString, Required: true, MinLen: 1},
	}, bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	driverName, err := s.store.TransactID(func(tx *storeTx) (string, error) {
		return recordSignOutTx(tx, stringValue(sanitized, fieldDriverID), now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+"|driver-remote-sign-out",
		map[string]any{keyDriverName: driverName},
	)
}

func (s *Server) handleDriverVoiceSignIn(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, []fieldRule{
		{
			Name:     keyDriverName,
			Kind:     kindString,
			Required: true,
			MinLen:   1,
			Check:    nonBlank(keyDriverName),
		},
		{Name: fieldVehicleID, Kind: kindString, Required: true, MinLen: 1},
	}, bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	var driver Record
	err = s.store.Transact(func(tx *storeTx) error {
		var applyErr error
		driver, applyErr = voiceSignInTx(
			tx,
			stringValue(sanitized, keyDriverName),
			stringValue(sanitized, fieldVehicleID),
			now,
		)
		return applyErr
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	driverID, driverName := recordID(driver), stringValue(driver, keyName)
	payload := map[string]any{
		keyData: map[string]any{fieldDriverID: driverID, keyDriverName: driverName},
	}
	s.respondJSONStatus(
		writer,
		request,
		http.StatusCreated,
		requestSignature(request)+"|driver-voice-sign-in",
		payload,
	)
}

func recordSignOutTx(tx *storeTx, ref string, now time.Time) (string, error) {
	driver, idx := findRecordByRef(tx.records(ResourceDrivers), ref, nil)
	if idx < 0 {
		return "", notFound(keyDriver, ref)
	}
	_, err := tx.insert(ResourceDriverSignOuts, Record{
		fieldDriverID:     recordID(driver),
		"signedOutAtTime": now.UTC().Format(time.RFC3339),
	}, now)
	return stringValue(driver, keyName), err
}

func voiceSignInTx(tx *storeTx, spokenName, vehicleRef string, now time.Time) (Record, error) {
	vehicle, idx := findVehicleTx(tx, vehicleRef)
	if idx < 0 {
		return nil, notFound(keyVehicle, vehicleRef)
	}
	driver, err := matchDriverByName(tx.records(ResourceDrivers), spokenName)
	if err != nil {
		return nil, err
	}
	stamp := now.UTC().Format(time.RFC3339)
	endOngoingAssignmentsTx(tx, recordID(driver), recordID(vehicle), stamp)
	record := Record{
		fieldDriverID:       recordID(driver),
		fieldVehicleID:      recordID(vehicle),
		fieldStartTime:      stamp,
		fieldAssignedAtTime: stamp,
		fieldIsPassenger:    false,
		fieldAssignmentType: assignmentTypeVoiceSignIn,
	}
	if err = ensureAssignmentUniqueTx(tx, "", record); err != nil {
		return nil, err
	}
	if _, err = tx.insert(ResourceDriverVehicleAssignments, record, now); err != nil {
		return nil, err
	}
	return driver, nil
}

func matchDriverByName(drivers []Record, spoken string) (Record, error) {
	normalized := strings.Join(strings.Fields(strings.ToLower(spoken)), " ")
	var match Record
	for _, driver := range drivers {
		if driverDeactivated(driver) {
			continue
		}
		name := strings.Join(strings.Fields(strings.ToLower(stringValue(driver, keyName))), " ")
		if name != normalized {
			continue
		}
		if match != nil {
			return nil, invalidField(keyDriverName, "matches more than one active driver")
		}
		match = driver
	}
	if match == nil {
		return nil, notFound("active driver named", spoken)
	}
	return match, nil
}

func endOngoingAssignmentsTx(tx *storeTx, driverID, vehicleID, at string) {
	records := tx.records(ResourceDriverVehicleAssignments)
	for idx, record := range records {
		if record[fieldIsPassenger] == true || stringValue(record, fieldEndTime) != "" {
			continue
		}
		if stringValue(record, fieldDriverID) != driverID &&
			stringValue(record, fieldVehicleID) != vehicleID {
			continue
		}
		if stringValue(record, fieldStartTime) >= at {
			continue
		}
		next := cloneRecord(record)
		next[fieldEndTime] = at
		records[idx] = next
	}
}

func (s *Server) handleDriverWorkflowList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	workflowType := strings.TrimSpace(values.Get(workflowTypeFilterParameter))
	if workflowType != "" && !slices.Contains(driverWorkflowTypes, workflowType) {
		s.writeError(writer, invalidParameter(
			workflowTypeFilterParameter,
			"must be one of "+strings.Join(quoteAll(driverWorkflowTypes), ", "),
		))
		return
	}
	view := s.fleetView()
	records := make([]Record, 0, len(view.snap.workflows))
	for _, workflow := range view.snap.workflows {
		if workflowType != "" && stringValue(workflow, "workflowType") != workflowType {
			continue
		}
		records = append(records, Record{
			keyID:          recordID(workflow),
			keyName:        stringValue(workflow, keyName),
			"workflowType": stringValue(workflow, "workflowType"),
		})
	}
	s.respondPage(writer, request, records, "|driver-workflows")
}

func (s *Server) handleDriverWorkflowAssignment(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, []fieldRule{
		{Name: "workflowId", Kind: kindString, Required: true, MinLen: 1},
		{Name: "driverIdsToPublish", Kind: kindStringList},
		{Name: "driverIdsToUnpublish", Kind: kindStringList},
	}, bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	publish := stringListValues(sanitized["driverIdsToPublish"])
	unpublish := stringListValues(sanitized["driverIdsToUnpublish"])
	if len(publish)+len(unpublish) == 0 {
		s.writeError(writer, invalidField(
			"driverIdsToPublish or driverIdsToUnpublish",
			"must list at least one driver",
		))
		return
	}
	for _, id := range publish {
		if slices.Contains(unpublish, id) {
			s.writeError(writer, invalidField(
				"driverIdsToUnpublish",
				fmt.Sprintf("cannot also contain driver %s from driverIdsToPublish", id),
			))
			return
		}
	}
	workflowID := stringValue(sanitized, "workflowId")
	now := s.simNow()
	err = s.store.Transact(func(tx *storeTx) error {
		return publishWorkflowTx(tx, workflowID, publish, unpublish, now)
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	payload := map[string]any{keyData: map[string]any{"workflowId": workflowID}}
	s.respondJSON(writer, request, requestSignature(request)+"|driver-workflow-assignment", payload)
}

func publishWorkflowTx(
	tx *storeTx,
	workflowID string,
	publish []string,
	unpublish []string,
	now time.Time,
) error {
	if _, idx := tx.find(ResourceDriverWorkflows, workflowID); idx < 0 {
		return notFound("workflow", workflowID)
	}
	exists := recordExists(tx.records(ResourceDrivers))
	for _, id := range append(slices.Clone(publish), unpublish...) {
		if !exists(id) {
			return missingReference(keyDriver, id)
		}
	}
	_, err := tx.update(ResourceDriverWorkflows, workflowID, func(record Record) error {
		published := stringListValues(record["simPublishedDriverIds"])
		for _, id := range publish {
			if !slices.Contains(published, id) {
				published = append(published, id)
			}
		}
		published = slices.DeleteFunc(published, func(id string) bool {
			return slices.Contains(unpublish, id)
		})
		sort.Strings(published)
		values := make([]any, 0, len(published))
		for _, id := range published {
			values = append(values, id)
		}
		record["simPublishedDriverIds"] = values
		return nil
	}, now)
	return err
}
