package sim

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
)

const (
	fieldDriverActivationStatus = "driverActivationStatus"
	fieldSimDeactivatedAtTime   = "simDeactivatedAtTime"
	fieldSimStaticAssignedAt    = "simStaticAssignedAtTime"
	fieldStaticAssignedVehicle  = "staticAssignedVehicleId"
	fieldTagIDs                 = "tagIds"
	fieldUsername               = "username"
	fieldDriverLogin            = "password"
	fieldProfileImageBase64     = "profileImageBase64"
	fieldProfileImageURL        = "profileImageUrl"
	fieldUsDriverRuleset        = "usDriverRulesetOverride"
	fieldEldSettings            = "eldSettings"
	fieldLicenseNumber          = "licenseNumber"
	fieldLicenseState           = "licenseState"

	driverStatusActive      = "active"
	driverStatusDeactivated = "deactivated"
	defaultDriverTimezone   = "America/Los_Angeles"

	maxDriverNameLength     = 255
	maxDriverUsernameLength = 189
	maxDriverNotesLength    = 4096
	maxDriverShortText      = 255
	maxProfileImageURL      = 1024
	maxProfileImageBytes    = 1 << 20

	eventDriverCreated = "DriverCreated"
	eventDriverUpdated = "DriverUpdated"
)

var (
	driverActivationStatuses = []string{driverStatusActive, driverStatusDeactivated}
	driverLocales            = []string{
		"us", "at", "be", "ca", "gb", "fr", "de", "ie", "it", "lu", "mx", "nl", "es", "ch", "pr",
	}
	usRulesetCycles = []string{
		"USA Property (8/70)", "USA Property (7/60)", "USA Passenger (8/70)",
		"USA Passenger (7/60)", "Alaska Property (8/80)", "Alaska Property (7/70)",
		"Alaska Passenger (8/80)", "Alaska Passenger (7/70)", "California School/FLV (8/80)",
		"California Farm (8/112)", "California Property (8/80)",
		"California Flammable Liquid (8/80)", "California Passenger (8/80)",
		"California Motion Picture (8/80)", "Florida (8/80)", "Florida (7/70)",
		"Nebraska (8/80)", "Nebraska (7/70)", "North Carolina (8/80)", "North Carolina (7/70)",
		"Oklahoma (8/70)", "Oklahoma (7/60)", "Oregon (8/80)", "Oregon (7/70)",
		"South Carolina (8/80)", "South Carolina (7/70)", "Texas (7/70)", "Wisconsin (8/80)",
		"Wisconsin (7/70)",
	}
	usRulesetRestarts = []string{
		"34-hour Restart", "24-hour Restart", "36-hour Restart", "72-hour Restart", rulesetNone,
	}
	usRulesetRestbreaks = []string{
		rulesetPropertyBreak, "California Mealbreak (off-duty/sleeper)", rulesetNone,
	}
	usRulesetStates = []string{"", "AK", "CA", "FL", "NE", "NC", "OK", "OR", "SC", "TX", "WI"}
	licenseRegions  = []string{
		"AL", "AK", "AZ", "AR", "CA", "CO", "CT", "DE", "DC", "FL", "GA", "HI", "ID", "IL", "IN",
		"IA", "KS", "KY", "LA", "ME", "MD", "MA", "MI", "MN", "MS", "MO", "MT", "NE", "NV", "NH",
		"NJ", "NM", "NY", "NC", "ND", "OH", "OK", "OR", "PA", "RI", "SC", "SD", "TN", "TX", "UT",
		"VT", "VA", "WA", "WV", "WI", "WY", "AS", "GU", "MP", "PR", "VI", "UM", "AB", "BC", "MB",
		"NB", "NL", "NS", "NT", "NU", "ON", "PE", "QC", "SK", "YT",
	}
	driverStoredFields = []string{
		keyName, fieldUsername, "phone", keyEmail, fieldLicenseNumber, fieldLicenseState,
		keyDateOfBirth, keyTimezone, keyLocale, keyNotes, keyCarrierSettings, "currentIdCardCode",
		keyAdverseWeather, keyBigDayExemption, "eldDayStartHour",
		keyEldExempt, keyEldExemptReason, keyEldPcEnabled, "eldYmEnabled",
		keyDrivingHidden, keyVehicleUnpinning, "hosSetting",
		fieldProfileImageURL, keyTachographCard, fieldUsDriverRuleset,
		keyWaitingTime, fieldExternalIDs, fieldAttributesKey,
		keyPeerGroupTagID, keyVehicleGroupTagID, keyTrailerGroupTagID, fieldStaticAssignedVehicle,
	}
	driverDefaultBooleans = map[string]bool{
		keyAdverseWeather:   false,
		keyBigDayExemption:  false,
		keyEldExempt:        false,
		keyEldPcEnabled:     false,
		"eldYmEnabled":      false,
		keyDrivingHidden:    false,
		keyVehicleUnpinning: true,
		keyWaitingTime:      false,
	}
	driverPassthroughFields = []string{
		"attributes", keyCarrierSettings, fieldCreatedAtTime, "currentIdCardCode", keyDateOfBirth,
		keyEldExemptReason, keyEmail, fieldEldSettings, fieldLicenseNumber, fieldLicenseState,
		keyLocale, keyNotes, "phone", fieldProfileImageURL, keyTachographCard,
		fieldUpdatedAtTime, fieldUsDriverRuleset, fieldUsername,
	}
	rulesetRegionCodes = map[string]string{
		"USA":            "USA",
		"Alaska":         "AK",
		"California":     "CA",
		"Florida":        "FL",
		"Nebraska":       "NE",
		"North Carolina": "NC",
		"Oklahoma":       "OK",
		"Oregon":         "OR",
		"South Carolina": "SC",
		"Texas":          "TX",
		"Wisconsin":      "WI",
	}
	rulesetCyclePattern = regexp.MustCompile(`^(.+) \((\d+)/(\d+)\)$`)
)

func (s *Server) registerDriverRoutes() {
	s.mux.HandleFunc("GET /fleet/drivers", s.handleDriverList)
	s.mux.HandleFunc("POST /fleet/drivers", s.handleDriverCreate)
	s.mux.HandleFunc("GET /fleet/drivers/{id}", s.handleDriverGet)
	s.mux.HandleFunc("PATCH /fleet/drivers/{id}", s.handleDriverPatch)
	s.registerDriverOperationRoutes()
}

func driverStatus(driver Record) string {
	if status := stringValue(driver, fieldDriverActivationStatus); status != "" {
		return status
	}
	return driverStatusActive
}

func parseActivationStatus(values url.Values) (string, error) {
	raw := strings.TrimSpace(values.Get(fieldDriverActivationStatus))
	if raw == "" {
		return driverStatusActive, nil
	}
	if !slices.Contains(driverActivationStatuses, raw) {
		return "", invalidParameter(
			fieldDriverActivationStatus,
			"must be `active` or `deactivated`",
		)
	}
	return raw, nil
}

func driverBodyRules(mode bodyMode) []fieldRule {
	create := mode == bodyCreate
	rules := make([]fieldRule, 0, 48)
	rules = append(rules, driverIdentityRules(create)...)
	rules = append(rules, driverProfileRules()...)
	rules = append(rules, driverEldRules()...)
	if mode == bodyPatch {
		rules = append(
			rules,
			fieldRule{
				Name: fieldDriverActivationStatus,
				Kind: kindString,
				Enum: driverActivationStatuses,
			},
			fieldRule{Name: "deactivatedAtTime", Kind: kindTime},
		)
	}
	return rules
}

func driverIdentityRules(create bool) []fieldRule {
	return []fieldRule{
		{
			Name:     keyName,
			Kind:     kindString,
			Required: create,
			MinLen:   1,
			MaxLen:   maxDriverNameLength,
			Check:    nonBlank(keyName),
		},
		{
			Name:     fieldUsername,
			Kind:     kindString,
			Required: create,
			MinLen:   1,
			MaxLen:   maxDriverUsernameLength,
			Check:    validateUsername,
		},
		{
			Name:     fieldDriverLogin,
			Kind:     kindString,
			Required: create,
			MinLen:   1,
			Check:    nonBlank(fieldDriverLogin),
		},
	}
}

func driverProfileRules() []fieldRule {
	nullableText := func(name string, maxLength int) fieldRule {
		return fieldRule{Name: name, Kind: kindString, MaxLen: maxLength, Nullable: true}
	}
	return []fieldRule{
		{Name: fieldAttributesKey, Kind: kindAttributes},
		{
			Name:     keyCarrierSettings,
			Kind:     kindObject,
			Nullable: true,
			Fields: []fieldRule{
				{Name: "carrierName", Kind: kindString, MaxLen: maxDriverShortText},
				{
					Name:     "dotNumber",
					Kind:     kindInt,
					IntRange: &intRange{Min: 1, Max: 99_999_999},
				},
				{Name: "homeTerminalAddress", Kind: kindString, MaxLen: maxDriverShortText},
				{Name: "homeTerminalName", Kind: kindString, MaxLen: maxDriverShortText},
				{Name: "mainOfficeAddress", Kind: kindString, MaxLen: maxDriverShortText},
			},
		},
		nullableText("currentIdCardCode", maxDriverShortText),
		{
			Name:     keyDateOfBirth,
			Kind:     kindDate,
			Nullable: true,
			Check:    notFutureDate(keyDateOfBirth),
		},
		{Name: keyEmail, Kind: kindString, Nullable: true, Check: validateEmail},
		{Name: fieldExternalIDs, Kind: kindExternalIDs, Nullable: true},
		nullableText(fieldLicenseNumber, maxDriverShortText),
		{Name: fieldLicenseState, Kind: kindString, Enum: licenseRegions, Nullable: true},
		{Name: keyLocale, Kind: kindString, Enum: driverLocales, Nullable: true},
		nullableText(keyNotes, maxDriverNotesLength),
		{Name: keyPeerGroupTagID, Kind: kindString, Nullable: true},
		nullableText("phone", maxDriverShortText),
		{Name: fieldProfileImageBase64, Kind: kindString, MinLen: 1},
		{
			Name:     fieldProfileImageURL,
			Kind:     kindString,
			MaxLen:   maxProfileImageURL,
			Nullable: true,
			Check:    validateHTTPURL(fieldProfileImageURL),
		},
		{Name: fieldStaticAssignedVehicle, Kind: kindString, Nullable: true},
		nullableText(keyTachographCard, maxDriverShortText),
		{Name: fieldTagIDs, Kind: kindStringList},
		{Name: keyTimezone, Kind: kindString, Check: validateTimezone},
		{Name: keyTrailerGroupTagID, Kind: kindString, Nullable: true},
		{Name: keyVehicleGroupTagID, Kind: kindString, Nullable: true},
	}
}

func driverEldRules() []fieldRule {
	return []fieldRule{
		{Name: keyAdverseWeather, Kind: kindBool},
		{Name: keyBigDayExemption, Kind: kindBool},
		{Name: "eldDayStartHour", Kind: kindInt, Check: oneOfInts("eldDayStartHour", 0, 12)},
		{Name: keyEldExempt, Kind: kindBool},
		{Name: keyEldExemptReason, Kind: kindString, MaxLen: maxDriverShortText, Nullable: true},
		{Name: keyEldPcEnabled, Kind: kindBool},
		{Name: "eldYmEnabled", Kind: kindBool},
		{Name: keyDrivingHidden, Kind: kindBool},
		{Name: keyVehicleUnpinning, Kind: kindBool},
		{
			Name: "hosSetting",
			Kind: kindObject,
			Fields: []fieldRule{
				{Name: "heavyHaulExemptionToggleEnabled", Kind: kindBool},
			},
		},
		{
			Name:     fieldUsDriverRuleset,
			Kind:     kindObject,
			Nullable: true,
			Fields: []fieldRule{
				{Name: keyCycle, Kind: kindString, Required: true, Enum: usRulesetCycles},
				{Name: "restart", Kind: kindString, Required: true, Enum: usRulesetRestarts},
				{Name: "restbreak", Kind: kindString, Required: true, Enum: usRulesetRestbreaks},
				{
					Name:     "usStateToOverride",
					Kind:     kindString,
					Required: true,
					Enum:     usRulesetStates,
				},
			},
		},
		{Name: keyWaitingTime, Kind: kindBool},
	}
}

func nonBlank(field string) func(value any) error {
	return func(value any) error {
		if strings.TrimSpace(stringOf(value)) == "" {
			return invalidField(field, "cannot be blank")
		}
		return nil
	}
}

func validateUsername(value any) error {
	username := stringOf(value)
	if strings.ContainsFunc(username, unicode.IsSpace) || strings.Contains(username, "@") {
		return invalidField(fieldUsername, "may not contain spaces or the '@' symbol")
	}
	return nil
}

func validateEmail(value any) error {
	text := strings.TrimSpace(stringOf(value))
	parsed, err := mail.ParseAddress(text)
	if err != nil || parsed.Address != text {
		return invalidField(keyEmail, "must be a valid email address")
	}
	return nil
}

func validateTimezone(value any) error {
	name := strings.TrimSpace(stringOf(value))
	if name == "" || strings.EqualFold(name, "local") {
		return invalidField(keyTimezone, "must be an IANA time zone such as America/Chicago")
	}
	if _, err := time.LoadLocation(name); err != nil {
		return invalidField(keyTimezone, "must be an IANA time zone such as America/Chicago")
	}
	return nil
}

func validateHTTPURL(field string) func(value any) error {
	return func(value any) error {
		parsed, err := url.Parse(strings.TrimSpace(stringOf(value)))
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.Host == "" {
			return invalidField(field, "must be an absolute http or https URL")
		}
		return nil
	}
}

func notFutureDate(field string) func(value any) error {
	return func(value any) error {
		day, err := time.Parse(dateLayout, stringOf(value))
		if err != nil || day.After(time.Now().UTC()) {
			return invalidField(field, "must be a past YYYY-MM-DD date")
		}
		return nil
	}
}

func oneOfInts(field string, allowed ...int64) func(value any) error {
	return func(value any) error {
		if number, ok := value.(int64); !ok || !slices.Contains(allowed, number) {
			parts := make([]string, 0, len(allowed))
			for _, option := range allowed {
				parts = append(parts, fmt.Sprintf("`%d`", option))
			}
			return invalidField(field, "must be one of "+strings.Join(parts, ", "))
		}
		return nil
	}
}

func (s *Server) handleDriverList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	status, err := parseActivationStatus(values)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	attributes, err := parseAttributeFilter(values, attributeFilterDriver)
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
	records := make([]Record, 0, len(view.snap.drivers))
	for _, driver := range view.snap.drivers {
		id := recordID(driver)
		if driverStatus(driver) != status ||
			!filter.matches(view.snap.tags, tagMembersDrivers, id) ||
			!attributes.matches(driver) ||
			!recordAtOrAfter(driver, fieldUpdatedAtTime, updatedAfter) ||
			!recordAtOrAfter(driver, fieldCreatedAtTime, createdAfter) {
			continue
		}
		records = append(records, view.driverView(driver))
	}
	s.respondPage(writer, request, records, "|driver-list")
}

func recordAtOrAfter(record Record, key string, threshold *time.Time) bool {
	if threshold == nil {
		return true
	}
	at := recordTime(record, key)
	return !at.IsZero() && !at.Before(*threshold)
}

func (s *Server) handleDriverGet(writer http.ResponseWriter, request *http.Request) {
	ref, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	view := s.fleetView()
	driver, idx := findRecordByRef(view.snap.drivers, ref, nil)
	if idx < 0 {
		s.writeError(writer, notFound(keyDriver, ref))
		return
	}
	payload := map[string]any{keyData: view.driverView(driver)}
	s.respondJSON(writer, request, requestSignature(request)+"|driver-get", payload)
}

func (s *Server) handleDriverCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeBody(body, driverBodyRules(bodyCreate), bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	driverID, err := s.createDriver(sanitized)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondDriverWrite(writer, request, driverID, eventDriverCreated, "|driver-create")
}

func (s *Server) handleDriverPatch(writer http.ResponseWriter, request *http.Request) {
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
	sanitized, err := sanitizeBody(body, driverBodyRules(bodyPatch), bodyPatch)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	driverID, err := s.patchDriver(ref, sanitized)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondDriverWrite(writer, request, driverID, eventDriverUpdated, "|driver-patch")
}

func (s *Server) createDriver(body Record) (string, error) {
	now := s.simNow()
	var driverID string
	err := s.store.Transact(func(tx *storeTx) error {
		record := Record{fieldDriverActivationStatus: driverStatusActive}
		if err := applyDriverFields(tx, record, "", body, now); err != nil {
			return err
		}
		created, err := tx.insert(ResourceDrivers, record, now)
		if err != nil {
			return err
		}
		driverID = recordID(created)
		if image, ok := body[fieldProfileImageBase64].(string); ok {
			if err = attachProfileImageTx(tx, driverID, image, now); err != nil {
				return err
			}
		}
		return setEntityTagsTx(tx, tagMembersDrivers, driverID, stringListValues(body[fieldTagIDs]))
	})
	return driverID, err
}

func (s *Server) patchDriver(ref string, body Record) (string, error) {
	now := s.simNow()
	var driverID string
	err := s.store.Transact(func(tx *storeTx) error {
		current, idx := findRecordByRef(tx.records(ResourceDrivers), ref, nil)
		if idx < 0 {
			return notFound(keyDriver, ref)
		}
		driverID = recordID(current)
		_, err := tx.update(ResourceDrivers, driverID, func(record Record) error {
			if applyErr := applyDriverFields(tx, record, driverID, body, now); applyErr != nil {
				return applyErr
			}
			return applyDriverActivation(record, body, now)
		}, now)
		if err != nil {
			return err
		}
		if image, ok := body[fieldProfileImageBase64].(string); ok {
			if err = attachProfileImageTx(tx, driverID, image, now); err != nil {
				return err
			}
		}
		if raw, ok := body[fieldTagIDs]; ok {
			return setEntityTagsTx(tx, tagMembersDrivers, driverID, stringListValues(raw))
		}
		return nil
	})
	return driverID, err
}

func (s *Server) respondDriverWrite(
	writer http.ResponseWriter,
	request *http.Request,
	driverID string,
	eventType string,
	signature string,
) {
	view := s.fleetView()
	driver, ok := view.snap.driverByID[driverID]
	if !ok {
		s.writeError(writer, notFound(keyDriver, driverID))
		return
	}
	data := view.driverView(driver)
	s.dispatchEvent(request, eventType, map[string]any{keyDriver: cloneRecord(data)})
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+signature,
		map[string]any{keyData: data},
	)
}

func applyDriverFields(
	tx *storeTx,
	record Record,
	selfID string,
	body Record,
	now time.Time,
) error {
	drivers := tx.records(ResourceDrivers)
	if err := ensureDriverIdentifiersUnique(tx, drivers, selfID, body); err != nil {
		return err
	}
	if err := ensureDriverTagsExist(tx.records(ResourceTags), body); err != nil {
		return err
	}
	if err := resolveStaticVehicle(tx, record, body, now); err != nil {
		return err
	}
	assignDriverFields(record, body)
	if _, ok := body[fieldUsDriverRuleset]; ok || stringValue(record, keyID) == "" {
		record[fieldEldSettings] = eldSettingsFor(record)
	}
	return ensureLicenseUnique(drivers, selfID, record)
}

func ensureDriverIdentifiersUnique(
	tx *storeTx,
	drivers []Record,
	selfID string,
	body Record,
) error {
	if username, ok := body[fieldUsername].(string); ok {
		for _, driver := range drivers {
			if recordID(driver) != selfID &&
				strings.EqualFold(stringValue(driver, fieldUsername), username) {
				return fmt.Errorf("%w: username %q is already taken", ErrUniqueConflict, username)
			}
		}
	}
	if raw, ok := body[fieldExternalIDs]; ok && raw != nil {
		return tx.ensureExternalIDs(externalIDClaim{
			Kind:  externalIDKindDrivers,
			Owner: externalIDOwner(ResourceDrivers, selfID),
			IDs:   mapOf(raw),
		})
	}
	return nil
}

func ensureDriverTagsExist(tags []Record, body Record) error {
	exists := recordExists(tags)
	for _, field := range driverGroupTagFields {
		tagID := strings.TrimSpace(stringOf(body[field]))
		if tagID != "" && !exists(tagID) {
			return missingReference("tag", tagID)
		}
	}
	return nil
}

func resolveStaticVehicle(tx *storeTx, record, body Record, now time.Time) error {
	vehicleRef := strings.TrimSpace(stringOf(body[fieldStaticAssignedVehicle]))
	if vehicleRef == "" {
		return nil
	}
	vehicle, idx := findVehicleTx(tx, vehicleRef)
	if idx < 0 {
		return missingReference(keyVehicle, vehicleRef)
	}
	if recordID(vehicle) != stringValue(record, fieldStaticAssignedVehicle) {
		record[fieldSimStaticAssignedAt] = now.UTC().Format(time.RFC3339)
	}
	body[fieldStaticAssignedVehicle] = recordID(vehicle)
	return nil
}

func assignDriverFields(record, body Record) {
	for _, field := range driverStoredFields {
		raw, ok := body[field]
		if !ok {
			continue
		}
		clearsGroupTag := raw == "" && slices.Contains(driverGroupTagFields, field)
		switch {
		case raw == nil || clearsGroupTag:
			delete(record, field)
			if field == fieldStaticAssignedVehicle {
				delete(record, fieldSimStaticAssignedAt)
			}
		case field == fieldAttributesKey:
			record[field] = resolveAttributeIDs(listOf(raw), attributeEntityDriver)
		default:
			record[field] = raw
		}
	}
}

func ensureLicenseUnique(drivers []Record, selfID string, record Record) error {
	number := stringValue(record, fieldLicenseNumber)
	state := stringValue(record, fieldLicenseState)
	if number == "" || state == "" {
		return nil
	}
	for _, driver := range drivers {
		if recordID(driver) == selfID {
			continue
		}
		if strings.EqualFold(stringValue(driver, fieldLicenseNumber), number) &&
			stringValue(driver, fieldLicenseState) == state {
			return fmt.Errorf(
				"%w: license %s/%s is already assigned to another driver",
				ErrUniqueConflict,
				state,
				number,
			)
		}
	}
	return nil
}

func applyDriverActivation(record, body Record, now time.Time) error {
	status, hasStatus := body[fieldDriverActivationStatus].(string)
	deactivatedAt, hasDeactivatedAt := body["deactivatedAtTime"].(string)
	if hasDeactivatedAt {
		at, err := parseRFC3339(deactivatedAt)
		if err != nil || at.After(now) {
			return invalidField(
				"deactivatedAtTime",
				"must be an RFC 3339 time that is not in the future",
			)
		}
		target := driverStatus(record)
		if hasStatus {
			target = status
		}
		if target != driverStatusDeactivated {
			return invalidField(
				"deactivatedAtTime",
				"can only be set when the driver is deactivated",
			)
		}
	}
	if hasStatus {
		switch status {
		case driverStatusDeactivated:
			if driverStatus(record) != driverStatusDeactivated || hasDeactivatedAt {
				at := now.UTC().Format(time.RFC3339)
				if hasDeactivatedAt {
					at = deactivatedAt
				}
				record[fieldSimDeactivatedAtTime] = at
			}
		case driverStatusActive:
			delete(record, fieldSimDeactivatedAtTime)
		}
		record[fieldDriverActivationStatus] = status
		return nil
	}
	if hasDeactivatedAt {
		record[fieldSimDeactivatedAtTime] = deactivatedAt
	}
	return nil
}

func attachProfileImageTx(tx *storeTx, driverID, encoded string, now time.Time) error {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(encoded))
	}
	if err != nil {
		return invalidField(fieldProfileImageBase64, "must be valid base64")
	}
	if len(raw) > maxProfileImageBytes {
		return invalidField(fieldProfileImageBase64, "must decode to at most 1 MiB")
	}
	var extension string
	switch {
	case bytes.HasPrefix(raw, []byte{0xFF, 0xD8, 0xFF}):
		extension = "jpg"
	case bytes.HasPrefix(raw, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}):
		extension = "png"
	default:
		return invalidField(fieldProfileImageBase64, "must be a JPEG or PNG image")
	}
	digest := sha256.Sum256(raw)
	location := fmt.Sprintf(
		"https://example.invalid/driver-profile-images/%s/%s.%s",
		driverID,
		hex.EncodeToString(digest[:8]),
		extension,
	)
	_, err = tx.update(ResourceDrivers, driverID, func(record Record) error {
		record[fieldProfileImageURL] = location
		return nil
	}, now)
	return err
}

func eldSettingsFor(record Record) map[string]any {
	ruleset := map[string]any{
		keyCycle:  cycleUSA70Hour8Day,
		"shift":   "US Interstate Property",
		"restart": "34-hour Restart",
		"break":   rulesetPropertyBreak,
	}
	if state := stringValue(record, fieldLicenseState); state != "" {
		ruleset["jurisdiction"] = state
	}
	if override, ok := anyAsMap(record[fieldUsDriverRuleset]); ok {
		applyRulesetOverride(ruleset, Record(override))
	}
	return map[string]any{"rulesets": []any{ruleset}}
}

func eldCycleForOverride(cycle string) (string, bool) {
	match := rulesetCyclePattern.FindStringSubmatch(cycle)
	if match == nil {
		return "", false
	}
	region := match[1]
	code, known := rulesetRegionCodes[region]
	for !known {
		cut := strings.LastIndex(region, " ")
		if cut < 0 {
			return "", false
		}
		region = region[:cut]
		code, known = rulesetRegionCodes[region]
	}
	return code + " " + match[3] + " hour / " + match[2] + " day", true
}

func applyRulesetOverride(ruleset map[string]any, override Record) {
	cycle := stringValue(override, keyCycle)
	if mapped, known := eldCycleForOverride(cycle); known {
		ruleset[keyCycle] = mapped
	}
	if strings.Contains(cycle, "Passenger") {
		ruleset["shift"] = "US Interstate Passenger"
	}
	if restart := stringValue(override, "restart"); restart != "" {
		ruleset["restart"] = restart
	}
	state := stringValue(override, "usStateToOverride")
	if state == "" {
		return
	}
	ruleset["jurisdiction"] = state
	if state == "TX" {
		ruleset["shift"] = "Texas Intrastate"
	}
}

func (v *fleetView) driverView(driver Record) Record {
	id := recordID(driver)
	out := Record{keyID: id, keyName: stringValue(driver, keyName)}
	for _, field := range driverPassthroughFields {
		if value, ok := driver[field]; ok && value != nil {
			out[field] = cloneAny(value)
		}
	}
	for field, fallback := range driverDefaultBooleans {
		if value, ok := boolValue(driver, field); ok {
			out[field] = value
			continue
		}
		out[field] = fallback
	}
	hour, ok := int64Value(driver["eldDayStartHour"])
	if !ok {
		hour = 0
	}
	out["eldDayStartHour"] = hour
	out[keyTimezone] = defaultDriverTimezone
	if timezone := stringValue(driver, keyTimezone); timezone != "" {
		out[keyTimezone] = timezone
	}
	status := driverStatus(driver)
	out[fieldDriverActivationStatus] = status
	out["isDeactivated"] = status == driverStatusDeactivated
	if _, has := out[fieldEldSettings]; !has {
		out[fieldEldSettings] = eldSettingsFor(driver)
	}
	if externalIDs := externalIDsOf(driver); len(externalIDs) > 0 {
		out[fieldExternalIDs] = renderExternalIDs(driver, nil)
	}
	out[keyTags] = v.snap.tags.tinyTags(tagMembersDrivers, id)
	for field, viewField := range map[string]string{
		keyPeerGroupTagID:    "peerGroupTag",
		keyVehicleGroupTagID: "vehicleGroupTag",
		keyTrailerGroupTagID: "trailerGroupTag",
	} {
		if tiny := v.snap.tags.tiny(stringValue(driver, field)); tiny != nil {
			out[viewField] = tiny
		}
	}
	if vehicleID := stringValue(driver, fieldStaticAssignedVehicle); vehicleID != "" {
		if vehicle, exists := v.snap.assetByID[vehicleID]; exists {
			out["staticAssignedVehicle"] = map[string]any{
				keyID:   vehicleID,
				keyName: stringValue(vehicle, keyName),
			}
		}
	}
	return out
}

func staticAssignmentsFromDrivers(drivers []Record) []apiAssignment {
	out := make([]apiAssignment, 0, 4)
	for _, driver := range drivers {
		vehicleID := stringValue(driver, fieldStaticAssignedVehicle)
		if vehicleID == "" {
			continue
		}
		start := recordTime(driver, fieldSimStaticAssignedAt)
		if start.IsZero() {
			start = recordTime(driver, fieldCreatedAtTime)
		}
		var end *time.Time
		if deactivatedAt, deactivated := driverDeactivatedAt(driver); deactivated &&
			!deactivatedAt.IsZero() {
			end = &deactivatedAt
		}
		out = append(out, apiAssignment{
			ID:         "static-" + recordID(driver),
			DriverID:   recordID(driver),
			VehicleID:  vehicleID,
			Start:      start,
			End:        end,
			AssignedAt: start,
			Type:       "static",
		})
	}
	return out
}
