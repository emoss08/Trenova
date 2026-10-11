package sim

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const (
	formMaxListIDs          = 100
	formMaxStreamFilterIDs  = 50
	formMaxTitleLength      = 255
	formIncludeExternalIDs  = "externalIds"
	formLookupLookback      = 120 * 24 * time.Hour
	formPDFReadyDelay       = 5 * time.Second
	formPDFJobTTL           = 24 * time.Hour
	formPDFURLTTL           = time.Hour
	formPDFURLPrefix        = "https://samsara-pdf-exports.s3.us-west-2.amazonaws.com/"
	formPDFStatusPending    = "pending"
	formPDFStatusDone       = "done"
	formStreamCursorVersion = 1

	formStatusNotStarted       = "notStarted"
	formStatusInProgress       = "inProgress"
	formStatusArchived         = "archived"
	formStatusChangesRequested = "changesRequested"
	formStatusApproved         = "approved"
	formStatusDenied           = "denied"

	fieldFormTemplate = "formTemplate"
	fieldAssignedTo   = "assignedTo"
	fieldDueAtTime    = "dueAtTime"
	fieldRouteStopID  = "routeStopId"
	fieldRouteID      = "routeId"
	fieldIsRequired   = "isRequired"
	fieldApproval     = "approvalDetails"
	fieldSubmittedAt  = "submittedAtTime"
	fieldSubmittedBy  = "submittedBy"
	fieldFormFields   = "fields"
	fieldTitle        = "title"
	fieldStatus       = "status"
)

var (
	formPatchStatuses = []string{
		formStatusNotStarted, formStatusArchived, formStatusInProgress,
		formStatusChangesRequested, formStatusApproved, formStatusDenied,
	}
	formApprovalStatuses = []string{
		formStatusChangesRequested, formStatusApproved, formStatusDenied,
	}
	formAssigneeTypes = []string{formSubmitterTypeDriver, formUserTypeUser}
)

func (s *Server) registerFormRoutes() {
	s.mux.HandleFunc("GET /form-templates", s.handleFormTemplateList)
	s.mux.HandleFunc("GET /form-submissions", s.handleFormSubmissionList)
	s.mux.HandleFunc("GET /form-submissions/stream", s.handleFormSubmissionStream)
	s.mux.HandleFunc("POST /form-submissions", s.handleFormSubmissionCreate)
	s.mux.HandleFunc("PATCH /form-submissions", s.handleFormSubmissionPatch)
	s.mux.HandleFunc("GET /form-submissions/pdf-exports", s.handleFormPDFExportGet)
	s.mux.HandleFunc("POST /form-submissions/pdf-exports", s.handleFormPDFExportCreate)
}

func boundedIDList(values url.Values, name string, maxIDs int) ([]string, error) {
	ids := splitCSV(values.Get(name))
	if len(ids) > maxIDs {
		return nil, invalidParameter(name, fmt.Sprintf("accepts at most %d values", maxIDs))
	}
	return ids, nil
}

func parseFormInclude(values url.Values) (bool, error) {
	include := false
	for _, value := range splitCSV(values.Get("include")) {
		if value != formIncludeExternalIDs {
			return false, invalidParameter("include", "valid values: `externalIds`")
		}
		include = true
	}
	return include, nil
}

func (s *Server) handleFormTemplateList(writer http.ResponseWriter, request *http.Request) {
	ids, err := boundedIDList(request.URL.Query(), "ids", formMaxListIDs)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	templates, err := s.store.List(ResourceFormTemplates)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sort.SliceStable(templates, func(i, j int) bool {
		return stringValue(templates[i], fieldCreatedAtTime) < stringValue(templates[j], fieldCreatedAtTime)
	})
	s.respondPage(writer, request, filterByIDs(templates, ids), "|form-template-list")
}

func renderFormSubmission(record Record, now time.Time, includeExternalIDs bool) Record {
	out := cloneRecord(record)
	refreshFormMedia(map[string]any(out), now.Add(formMediaURLTTL).UTC().Format(time.RFC3339))
	externalIDs := renderExternalIDs(record, nil)
	switch {
	case includeExternalIDs:
		out[fieldExternalIDs] = externalIDs
	case len(externalIDs) > 0:
		out[fieldExternalIDs] = externalIDs
	default:
		delete(out, fieldExternalIDs)
	}
	return out
}

func (s *Server) generatedFormSubmissions(now, start, end time.Time) []Record {
	if s.live == nil {
		return []Record{}
	}
	return s.live.GeneratedFormSubmissions(now, start.Add(-time.Second), end, nil, nil)
}

func (s *Server) formSubmissionPool(now, start, end time.Time) ([]Record, error) {
	stored, err := s.store.List(ResourceFormSubmissions)
	if err != nil {
		return nil, err
	}
	storedIDs := make(map[string]struct{}, len(stored))
	for _, record := range stored {
		storedIDs[recordID(record)] = struct{}{}
	}
	generated := s.generatedFormSubmissions(now, start, end)
	out := make([]Record, 0, len(stored)+len(generated))
	out = append(out, stored...)
	for _, record := range generated {
		if _, overridden := storedIDs[recordID(record)]; !overridden {
			out = append(out, record)
		}
	}
	return out, nil
}

func (s *Server) lookupFormSubmissions(now time.Time, refs []string) ([]Record, error) {
	pool, err := s.formSubmissionPool(now, now.Add(-formLookupLookback), now.Add(time.Second))
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, raw := range refs {
		record, idx := findRecordByRef(pool, raw, nil)
		if idx < 0 {
			continue
		}
		if _, dup := seen[recordID(record)]; dup {
			continue
		}
		seen[recordID(record)] = struct{}{}
		out = append(out, record)
	}
	return out, nil
}

func (s *Server) handleFormSubmissionList(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	ids, err := boundedIDList(values, "ids", formMaxListIDs)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if len(ids) == 0 {
		s.writeError(writer, invalidParameter("ids", "is required"))
		return
	}
	include, err := parseFormInclude(values)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	records, err := s.lookupFormSubmissions(now, ids)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	data := make([]any, 0, len(records))
	for _, record := range records {
		data = append(data, renderFormSubmission(record, now, include))
	}
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+"|form-submission-list",
		map[string]any{keyData: data},
	)
}

type formStreamFilter struct {
	templates  map[string]struct{}
	users      map[string]struct{}
	drivers    map[string]struct{}
	routeStops map[string]struct{}
}

func parseFormStreamFilter(values url.Values) (formStreamFilter, error) {
	lists := make(map[string][]string, 4)
	for _, name := range []string{"formTemplateIds", "userIds", "driverIds", "assignedToRouteStopIds"} {
		ids, err := boundedIDList(values, name, formMaxStreamFilterIDs)
		if err != nil {
			return formStreamFilter{}, err
		}
		lists[name] = ids
	}
	return formStreamFilter{
		templates:  toStringSet(lists["formTemplateIds"]),
		users:      toStringSet(lists["userIds"]),
		drivers:    toStringSet(lists["driverIds"]),
		routeStops: toStringSet(lists["assignedToRouteStopIds"]),
	}, nil
}

func (f formStreamFilter) matches(record Record) bool {
	if !matchesStringFilter(f.templates, nestedString(record, fieldFormTemplate, keyID)) {
		return false
	}
	if !matchesStringFilter(f.routeStops, stringValue(record, fieldRouteStopID)) {
		return false
	}
	if len(f.users) == 0 && len(f.drivers) == 0 {
		return true
	}
	submitter := nestedString(record, fieldSubmittedBy, keyID)
	switch nestedString(record, fieldSubmittedBy, keyType) {
	case formSubmitterTypeDriver:
		_, ok := f.drivers[submitter]
		return ok
	case formUserTypeUser:
		_, ok := f.users[submitter]
		return ok
	default:
		return false
	}
}

type formStreamCursor struct {
	Version int    `json:"v"`
	Time    string `json:"t"`
	ID      string `json:"i,omitempty"`
}

func encodeFormStreamCursor(at, id string) (string, error) {
	raw, err := sonic.Marshal(formStreamCursor{Version: formStreamCursorVersion, Time: at, ID: id})
	if err != nil {
		return "", fmt.Errorf("encode stream cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeFormStreamCursor(value string) (formStreamCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return formStreamCursor{}, ErrCursorInvalid
	}
	cursor := formStreamCursor{}
	if err = sonic.Unmarshal(raw, &cursor); err != nil || cursor.Version != formStreamCursorVersion {
		return formStreamCursor{}, ErrCursorInvalid
	}
	if _, err = time.Parse(time.RFC3339, cursor.Time); err != nil {
		return formStreamCursor{}, ErrCursorInvalid
	}
	return cursor, nil
}

func (c formStreamCursor) before(updatedAt, id string) bool {
	if updatedAt != c.Time {
		return updatedAt > c.Time
	}
	return id > c.ID
}

func (s *Server) handleFormSubmissionStream(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	if startTime == nil {
		s.writeError(writer, invalidParameter(fieldStartTime, "is required"))
		return
	}
	filter, err := parseFormStreamFilter(values)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	include, err := parseFormInclude(values)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	var cursor *formStreamCursor
	if after := strings.TrimSpace(values.Get("after")); after != "" {
		decoded, decodeErr := decodeFormStreamCursor(after)
		if decodeErr != nil {
			s.writeError(writer, decodeErr)
			return
		}
		cursor = &decoded
	}

	now := s.simNow()
	windowEnd := now.Add(time.Second)
	if endTime != nil {
		windowEnd = *endTime
	}
	pool, err := s.formSubmissionPool(now, *startTime, windowEnd)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	selected := make([]Record, 0, len(pool))
	for _, record := range pool {
		updated, parseErr := parseRFC3339(stringValue(record, fieldUpdatedAtTime))
		if parseErr != nil || updated.Before(*startTime) || !updated.Before(windowEnd) {
			continue
		}
		if !filter.matches(record) {
			continue
		}
		if cursor != nil && !cursor.before(stringValue(record, fieldUpdatedAtTime), recordID(record)) {
			continue
		}
		selected = append(selected, record)
	}
	sortFormSubmissionsByUpdate(selected)

	pageSize := pagePolicyFor(request).Size
	page := selected[:min(pageSize, len(selected))]
	hasNext := len(selected) > len(page)
	endCursor := ""
	switch {
	case len(page) > 0 && (hasNext || endTime == nil):
		last := page[len(page)-1]
		endCursor, err = encodeFormStreamCursor(stringValue(last, fieldUpdatedAtTime), recordID(last))
	case len(page) == 0 && endTime == nil && cursor != nil:
		endCursor, err = encodeFormStreamCursor(cursor.Time, cursor.ID)
	case len(page) == 0 && endTime == nil:
		endCursor, err = encodeFormStreamCursor(
			startTime.Add(-time.Second).UTC().Format(time.RFC3339),
			"",
		)
	}
	if err != nil {
		s.writeError(writer, err)
		return
	}
	data := make([]any, 0, len(page))
	for _, record := range page {
		data = append(data, renderFormSubmission(record, now, include))
	}
	s.respondJSON(writer, request, requestSignature(request)+"|form-submission-stream", map[string]any{
		keyData:       data,
		keyPagination: map[string]any{"endCursor": endCursor, "hasNextPage": hasNext},
	})
}

func sortFormSubmissionsByUpdate(records []Record) {
	sort.SliceStable(records, func(i, j int) bool {
		left := stringValue(records[i], fieldUpdatedAtTime)
		right := stringValue(records[j], fieldUpdatedAtTime)
		if left != right {
			return left < right
		}
		return recordID(records[i]) < recordID(records[j])
	})
}

type formWriteContext struct {
	server    *Server
	view      *fleetView
	templates []Record
	users     map[string]struct{}
	now       time.Time
}

func (s *Server) newFormWriteContext() (*formWriteContext, error) {
	templates, err := s.store.List(ResourceFormTemplates)
	if err != nil {
		return nil, err
	}
	return &formWriteContext{
		server:    s,
		view:      s.fleetView(),
		templates: templates,
		users:     formUserIDs(templates),
		now:       s.simNow(),
	}, nil
}

func (c *formWriteContext) template(id string) (Record, bool) {
	for _, template := range c.templates {
		if recordID(template) == id {
			return template, true
		}
	}
	return nil, false
}

func (c *formWriteContext) resolveTemplate(body Record) (Record, error) {
	raw, present := body[fieldFormTemplate]
	if !present || raw == nil {
		return nil, invalidField(fieldFormTemplate, "is required")
	}
	ref, ok := anyAsMap(raw)
	if !ok {
		return nil, invalidField(fieldFormTemplate, "must be an object")
	}
	templateID, err := requiredString(ref, keyID, fieldFormTemplate)
	if err != nil {
		return nil, err
	}
	template, found := c.template(templateID)
	if !found {
		return nil, missingReference("form template", templateID)
	}
	if rawRevision, has := ref["revisionId"]; has && rawRevision != nil {
		revision, isString := rawRevision.(string)
		if !isString {
			return nil, invalidField(fieldFormTemplate+".revisionId", "must be a string")
		}
		if strings.TrimSpace(revision) != stringValue(template, "revisionId") {
			return nil, invalidField(
				fieldFormTemplate+".revisionId",
				fmt.Sprintf("%q is not a revision of form template %s", revision, templateID),
			)
		}
	}
	return template, nil
}

func (c *formWriteContext) assignee(raw any) (map[string]any, error) {
	assignee, ok := anyAsMap(raw)
	if !ok {
		return nil, invalidField(fieldAssignedTo, "must be an object")
	}
	id, err := requiredString(assignee, keyID, fieldAssignedTo)
	if err != nil {
		return nil, err
	}
	kind, err := requiredEnum(assignee, keyType, fieldAssignedTo, formAssigneeTypes)
	if err != nil {
		return nil, err
	}
	if kind == formSubmitterTypeDriver {
		driver, exists := c.view.snap.driverByID[id]
		if !exists {
			return nil, missingReference(keyDriver, id)
		}
		if driverDeactivated(driver) {
			return nil, invalidField(fieldAssignedTo+".id", "driver "+id+" is deactivated")
		}
	} else if _, exists := c.users[id]; !exists {
		return nil, missingReference(formUserTypeUser, id)
	}
	return map[string]any{keyID: id, keyType: kind}, nil
}

func (c *formWriteContext) applyCommonFields(record, body Record, template Record, mediaSeed string) error {
	if raw, ok := body[fieldTitle]; ok && raw != nil {
		title, err := sanitizeString(raw, &fieldRule{MaxLen: formMaxTitleLength}, fieldTitle)
		if err != nil {
			return err
		}
		record[fieldTitle] = title
	}
	if raw, ok := body[fieldAssignedTo]; ok && raw != nil {
		assignee, err := c.assignee(raw)
		if err != nil {
			return err
		}
		record[fieldAssignedTo] = assignee
		record[fieldAssignedAtTime] = c.now.UTC().Format(time.RFC3339)
	}
	if raw, ok := body[fieldDueAtTime]; ok && raw != nil {
		due, err := sanitizeTime(raw, fieldDueAtTime)
		if err != nil {
			return err
		}
		record[fieldDueAtTime] = due
	}
	if raw, ok := body[fieldIsRequired]; ok && raw != nil {
		required, err := sanitizeBool(raw, fieldIsRequired)
		if err != nil {
			return err
		}
		record[fieldIsRequired] = required
	}
	if raw, ok := body[fieldRouteStopID]; ok && raw != nil {
		stopID, isString := raw.(string)
		if !isString || strings.TrimSpace(stopID) == "" {
			return invalidField(fieldRouteStopID, "must be a non-empty string")
		}
		match, found := findRouteStop(c.server.live.routeRefs(c.now), stopID, false)
		if !found {
			return missingReference("route stop", stopID)
		}
		record[fieldRouteStopID] = match.Stop.ID
		record[fieldRouteID] = match.Route.ID
	}
	if raw, ok := body[fieldFormFields]; ok && raw != nil {
		inputs, err := newFormInputContext(c.view, template, c.users, mediaSeed).sanitizeInputs(raw)
		if err != nil {
			return err
		}
		record[fieldFormFields] = mergeFormInputs(listOf(record[fieldFormFields]), inputs)
	}
	return nil
}

func formMediaSeed(body Record, now time.Time) string {
	return now.UTC().Format(time.RFC3339Nano) + "|" + fmt.Sprintf("%x", fnvHash64(fmt.Sprint(body)))
}

func (s *Server) handleFormSubmissionCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	ctx, err := s.newFormWriteContext()
	if err != nil {
		s.writeError(writer, err)
		return
	}
	template, err := ctx.resolveTemplate(body)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	status, ok := body[fieldStatus].(string)
	if !ok {
		s.writeError(writer, invalidField(fieldStatus, "is required"))
		return
	}
	if status != formStatusNotStarted {
		s.writeError(writer, invalidField(fieldStatus, "must be one of `notStarted`"))
		return
	}
	stamp := ctx.now.UTC().Format(time.RFC3339)
	record := Record{
		fieldStatus: formStatusNotStarted,
		fieldFormTemplate: map[string]any{
			keyID:        recordID(template),
			"revisionId": stringValue(template, "revisionId"),
		},
		fieldSubmittedAt: stamp,
		fieldSubmittedBy: map[string]any{keyID: formAPIUserID(ctx.templates), keyType: formUserTypeUser},
		fieldFormFields:  []any{},
	}
	if err = ctx.applyCommonFields(record, body, template, formMediaSeed(body, ctx.now)); err != nil {
		s.writeError(writer, err)
		return
	}
	if _, set := record[fieldIsRequired]; !set {
		_, assigned := record[fieldAssignedTo]
		record[fieldIsRequired] = assigned
	}
	var created Record
	err = s.store.Transact(func(tx *storeTx) error {
		inserted, insertErr := tx.insert(ResourceFormSubmissions, record, ctx.now)
		created = cloneRecord(inserted)
		return insertErr
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+"|form-submission-create",
		map[string]any{keyData: renderFormSubmission(created, ctx.now, false)},
	)
}

func validateFormStatusChange(template Record, current, next string, body Record) error {
	_, requiresApproval := anyAsMap(template["approvalConfig"])
	approval := slices.Contains(formApprovalStatuses, next)
	switch {
	case current == formStatusArchived && next != formStatusArchived:
		return invalidField(fieldStatus, "an archived form submission cannot change status")
	case approval && !requiresApproval:
		return invalidField(fieldStatus, fmt.Sprintf("%q is only valid for forms that require approvals", next))
	case approval && current != formSubmissionStatusNeedsReview:
		return invalidField(fieldStatus, fmt.Sprintf("%q requires a submission in needsReview, not %s", next, current))
	case (next == formStatusNotStarted || next == formStatusInProgress) &&
		current != formStatusNotStarted && current != formStatusInProgress:
		return invalidField(fieldStatus, fmt.Sprintf("a %s form submission cannot return to %s", current, next))
	}
	if next == formStatusChangesRequested || next == formStatusDenied {
		comment := strings.TrimSpace(nestedString(body, fieldApproval, "comment"))
		if comment == "" {
			return invalidField(fieldApproval+".comment", "is required when status is "+next)
		}
	}
	return nil
}

func (s *Server) handleFormSubmissionPatch(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	id, ok := body[keyID].(string)
	if !ok || strings.TrimSpace(id) == "" {
		s.writeError(writer, invalidField(keyID, "is required"))
		return
	}
	id = strings.TrimSpace(id)
	ctx, err := s.newFormWriteContext()
	if err != nil {
		s.writeError(writer, err)
		return
	}
	current, err := s.formSubmissionForWrite(ctx.now, id)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	template, found := ctx.template(nestedString(current, fieldFormTemplate, keyID))
	if !found {
		s.writeError(writer, missingReference("form template", nestedString(current, fieldFormTemplate, keyID)))
		return
	}
	next := cloneRecord(current)
	if err = applyFormStatusPatch(next, body, template); err != nil {
		s.writeError(writer, err)
		return
	}
	if err = ctx.applyCommonFields(next, body, template, formMediaSeed(body, ctx.now)); err != nil {
		s.writeError(writer, err)
		return
	}
	var updated Record
	err = s.store.Transact(func(tx *storeTx) error {
		if _, idx := tx.find(ResourceFormSubmissions, id); idx < 0 {
			if replaceErr := tx.replace(
				ResourceFormSubmissions,
				append(tx.records(ResourceFormSubmissions), cloneRecord(current)),
			); replaceErr != nil {
				return replaceErr
			}
		}
		result, updateErr := tx.update(ResourceFormSubmissions, id, func(record Record) error {
			for key := range record {
				delete(record, key)
			}
			for key, value := range next {
				record[key] = value
			}
			return nil
		}, ctx.now)
		updated = cloneRecord(result)
		return updateErr
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	rendered := renderFormSubmission(updated, ctx.now, false)
	s.dispatchEvent(request, "FormUpdated", map[string]any{"form": cloneRecord(rendered)})
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+"|form-submission-patch",
		map[string]any{keyData: rendered},
	)
}

func applyFormStatusPatch(record, body Record, template Record) error {
	rawApproval, hasApproval := body[fieldApproval]
	if hasApproval && rawApproval != nil {
		approval, ok := anyAsMap(rawApproval)
		if !ok {
			return invalidField(fieldApproval, "must be an object")
		}
		if comment, present := approval["comment"]; present && comment != nil {
			if _, isString := comment.(string); !isString {
				return invalidField(fieldApproval+".comment", "must be a string")
			}
		}
	}
	rawStatus, hasStatus := body[fieldStatus]
	if !hasStatus || rawStatus == nil {
		if hasApproval && rawApproval != nil {
			return invalidField(fieldApproval, "is only valid when requesting changes, approving or denying")
		}
		return nil
	}
	status, ok := rawStatus.(string)
	if !ok || !slices.Contains(formPatchStatuses, status) {
		return invalidField(fieldStatus, "must be one of "+strings.Join(quoteAll(formPatchStatuses), ", "))
	}
	if hasApproval && rawApproval != nil && !slices.Contains(formApprovalStatuses, status) {
		return invalidField(fieldApproval, "is only valid when requesting changes, approving or denying")
	}
	if err := validateFormStatusChange(template, stringValue(record, fieldStatus), status, body); err != nil {
		return err
	}
	record[fieldStatus] = status
	if comment := strings.TrimSpace(nestedString(body, fieldApproval, "comment")); comment != "" {
		record[fieldApproval] = map[string]any{"comment": comment}
	} else if slices.Contains(formApprovalStatuses, status) {
		delete(record, fieldApproval)
	}
	return nil
}

func (s *Server) formSubmissionForWrite(now time.Time, id string) (Record, error) {
	stored, err := s.store.Get(ResourceFormSubmissions, id)
	if err == nil {
		return stored, nil
	}
	records, err := s.lookupFormSubmissions(now, []string{id})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 || recordID(records[0]) != id {
		return nil, notFound("form submission", id)
	}
	return records[0], nil
}

func renderFormPDFExport(job Record, now time.Time) map[string]any {
	requestedAt, _ := parseRFC3339(stringValue(job, "requestedAtTime"))
	pdfID := recordID(job)
	out := map[string]any{
		keyID:             stringValue(job, "formSubmissionId"),
		"pdfId":           pdfID,
		"requestedAtTime": requestedAt.Format(time.RFC3339),
		"expiresAtTime":   requestedAt.Add(formPDFJobTTL).Format(time.RFC3339),
		"jobStatus":       formPDFStatusPending,
	}
	completedAt := requestedAt.Add(formPDFReadyDelay)
	if now.Before(completedAt) {
		return out
	}
	out["jobStatus"] = formPDFStatusDone
	out["completedAtTime"] = completedAt.Format(time.RFC3339)
	out["pdfUrl"] = formPDFURLPrefix + pdfID + ".pdf"
	out["pdfUrlExpiresAtTime"] = now.Add(formPDFURLTTL).UTC().Format(time.RFC3339)
	return out
}

func (s *Server) handleFormPDFExportCreate(writer http.ResponseWriter, request *http.Request) {
	submissionID, err := queryID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	submission, err := s.formSubmissionForWrite(now, submissionID)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	var job Record
	err = s.store.Transact(func(tx *storeTx) error {
		inserted, insertErr := tx.insert(ResourceFormPDFExports, Record{
			"formSubmissionId": recordID(submission),
			"requestedAtTime":  now.UTC().Format(time.RFC3339),
		}, now)
		job = cloneRecord(inserted)
		return insertErr
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondJSONStatus(
		writer,
		request,
		http.StatusAccepted,
		requestSignature(request)+"|form-pdf-export-create",
		map[string]any{keyData: renderFormPDFExport(job, now)},
	)
}

func (s *Server) handleFormPDFExportGet(writer http.ResponseWriter, request *http.Request) {
	pdfID := queryValue(request, "pdfId")
	if pdfID == "" {
		s.writeError(writer, invalidParameter("pdfId", "is required"))
		return
	}
	job, err := s.store.Get(ResourceFormPDFExports, pdfID)
	if err != nil {
		s.writeError(writer, notFound("form submission PDF export", pdfID))
		return
	}
	now := s.simNow()
	requestedAt, err := parseRFC3339(stringValue(job, "requestedAtTime"))
	if err != nil || !now.Before(requestedAt.Add(formPDFJobTTL)) {
		s.writeError(writer, notFound("form submission PDF export", pdfID))
		return
	}
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+"|form-pdf-export-get",
		map[string]any{keyData: renderFormPDFExport(job, now)},
	)
}
