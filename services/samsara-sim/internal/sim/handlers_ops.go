package sim

import (
	"net/http"
	"strings"
	"time"
)

func (s *Server) registerFormRoutes() {
	s.mux.HandleFunc("GET /form-templates", s.handleFormTemplateList)
	s.mux.HandleFunc("GET /form-submissions", s.handleFormSubmissionList)
	s.mux.HandleFunc("POST /form-submissions", s.handleFormSubmissionCreate)
	s.mux.HandleFunc("PATCH /form-submissions", s.handleFormSubmissionPatch)
}

func (s *Server) registerMessageRoutes() {
	s.mux.HandleFunc("GET /v1/fleet/messages", s.handleMessageList)
	s.mux.HandleFunc("POST /v1/fleet/messages", s.handleMessageCreate)
}

func (s *Server) registerComplianceRoutes() {
	s.mux.HandleFunc("GET /fleet/hos/clocks", s.handleHOSClockList)
	s.mux.HandleFunc("GET /fleet/hos/logs", s.handleHOSLogList)
	s.mux.HandleFunc("GET /fleet/drivers/tachograph-files/history", s.handleDriverTachograph)
	s.mux.HandleFunc("GET /fleet/vehicles/tachograph-files/history", s.handleVehicleTachograph)
}

func (s *Server) registerVehicleRoutes() {
	s.mux.HandleFunc("GET /fleet/vehicles/stats", s.handleVehicleStats)
}

func (s *Server) registerWebhookRoutes() {
	s.mux.HandleFunc("GET /webhooks", s.handleWebhookList)
	s.mux.HandleFunc("POST /webhooks", s.handleWebhookCreate)
	s.mux.HandleFunc("GET /webhooks/{id}", s.handleWebhookGet)
	s.mux.HandleFunc("PATCH /webhooks/{id}", s.handleWebhookPatch)
	s.mux.HandleFunc("DELETE /webhooks/{id}", s.handleWebhookDelete)
}

func (s *Server) registerLiveShareRoutes() {
	s.mux.HandleFunc("GET /live-shares", s.handleLiveShareList)
	s.mux.HandleFunc("POST /live-shares", s.handleLiveShareCreate)
	s.mux.HandleFunc("PATCH /live-shares", s.handleLiveSharePatch)
	s.mux.HandleFunc("DELETE /live-shares", s.handleLiveShareDelete)
}

func (s *Server) handleFormTemplateList(writer http.ResponseWriter, request *http.Request) {
	records, err := s.store.List(ResourceFormTemplates)
	if err != nil {
		s.writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}

	filtered := filterByIDs(records, idsFromQuery(request.URL.Query(), "ids"))
	page, pagination, err := paginate(filtered, request.URL.Query(), 512)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|form-template-list", payload)
}

func (s *Server) handleFormSubmissionList(writer http.ResponseWriter, request *http.Request) {
	records, err := s.store.List(ResourceFormSubmissions)
	if err != nil {
		s.writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}

	filtered := filterByIDs(records, idsFromQuery(request.URL.Query(), "ids"))
	payload := map[string]any{
		"data": recordsAsAny(filtered),
	}
	s.respondJSON(writer, request, requestSignature(request)+"|form-submission-list", payload)
}

func (s *Server) handleFormSubmissionCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	created, err := s.store.Create(ResourceFormSubmissions, body)
	if err != nil {
		s.respondStoreError(writer, err)
		return
	}
	s.dispatchEvent(request, "FormSubmitted", created)

	payload := map[string]any{"data": created}
	s.respondJSON(writer, request, requestSignature(request)+"|form-submission-create", payload)
}

func (s *Server) handleFormSubmissionPatch(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	id := strings.TrimSpace(stringValue(body, "id"))
	if id == "" {
		s.writeAPIError(writer, http.StatusBadRequest, ErrRecordIDRequired)
		return
	}

	updated, err := s.store.Patch(ResourceFormSubmissions, id, body)
	if err != nil {
		s.respondStoreError(writer, err)
		return
	}
	s.dispatchEvent(request, "FormUpdated", updated)

	payload := map[string]any{"data": updated}
	s.respondJSON(writer, request, requestSignature(request)+"|form-submission-patch", payload)
}

func (s *Server) handleMessageList(writer http.ResponseWriter, request *http.Request) {
	records, err := s.store.List(ResourceMessages)
	if err != nil {
		s.writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}

	payload := map[string]any{
		"data": recordsAsAny(records),
	}
	s.respondJSON(writer, request, requestSignature(request)+"|message-list", payload)
}

func (s *Server) handleMessageCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	text, _ := body["text"].(string)
	text = strings.TrimSpace(text)
	if text == "" {
		s.writeAPIError(writer, http.StatusBadRequest, ErrInvalidBody)
		return
	}

	driverIDs := numbersAsInt64(body["driverIds"])
	if len(driverIDs) == 0 {
		s.writeAPIError(writer, http.StatusBadRequest, ErrInvalidBody)
		return
	}

	createdData := make([]any, 0, len(driverIDs))
	persisted := make([]Record, 0, len(driverIDs))
	nowMillis := s.simNow().UnixMilli()
	for _, driverID := range driverIDs {
		createdData = append(createdData, map[string]any{
			"driverId": driverID,
			"text":     text,
		})
		persisted = append(persisted, Record{
			"driverId": driverID,
			"text":     text,
			"isRead":   false,
			"sender": map[string]any{
				"name": "Dispatch",
				"type": "dispatch",
			},
			"sentAtMs": nowMillis,
		})
	}
	s.store.AppendMessages(persisted)

	payload := map[string]any{
		"data": createdData,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|message-create", payload)
}

func (s *Server) handleHOSClockList(writer http.ResponseWriter, request *http.Request) {
	driverIDs := idsFromQuery(request.URL.Query(), "driverIds")
	now := s.simNow()

	var records []Record
	if s.live != nil {
		records = s.live.HOSClocks(now, driverIDs)
	} else {
		seedRecords, err := s.store.List(ResourceHOSClocks)
		if err != nil {
			s.writeAPIError(writer, http.StatusInternalServerError, err)
			return
		}
		records = filterByNestedIDs(seedRecords, driverIDs, "driver", "id")
	}

	page, pagination, err := paginate(records, request.URL.Query(), 512)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|hos-clocks", payload)
}

func (s *Server) handleHOSLogList(writer http.ResponseWriter, request *http.Request) {
	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	driverIDs := idsFromQuery(request.URL.Query(), "driverIds")
	now := s.simNow()

	var records []Record
	if s.live != nil {
		records = s.live.HOSLogs(now, driverIDs, startTime, endTime)
	} else {
		seedRecords, listErr := s.store.List(ResourceHOSLogs)
		if listErr != nil {
			s.writeAPIError(writer, http.StatusInternalServerError, listErr)
			return
		}
		seedRecords = filterByNestedIDs(seedRecords, driverIDs, "driver", "id")
		records = filterHOSLogsByTimeWindow(seedRecords, startTime, endTime)
	}

	page, pagination, err := paginate(records, request.URL.Query(), 512)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|hos-logs", payload)
}

func (s *Server) handleDriverTachograph(writer http.ResponseWriter, request *http.Request) {
	records, err := s.store.List(ResourceDriverTachograph)
	if err != nil {
		s.writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}
	records = filterByNestedIDs(
		records,
		idsFromQuery(request.URL.Query(), "driverIds"),
		"driver",
		"id",
	)
	page, pagination, err := paginate(records, request.URL.Query(), 512)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|driver-tacho", payload)
}

func (s *Server) handleVehicleTachograph(writer http.ResponseWriter, request *http.Request) {
	records, err := s.store.List(ResourceVehicleTachograph)
	if err != nil {
		s.writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}
	records = filterByNestedIDs(
		records,
		idsFromQuery(request.URL.Query(), "vehicleIds"),
		"vehicle",
		"id",
	)
	page, pagination, err := paginate(records, request.URL.Query(), 512)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|vehicle-tacho", payload)
}

func (s *Server) handleVehicleStats(writer http.ResponseWriter, request *http.Request) {
	vehicleIDs := idsFromQuery(request.URL.Query(), "vehicleIds")
	now := s.simNow()

	var records []Record
	if s.live != nil {
		records = s.live.VehicleStats(now, vehicleIDs)
	} else {
		seedRecords, err := s.store.List(ResourceVehicleStats)
		if err != nil {
			s.writeAPIError(writer, http.StatusInternalServerError, err)
			return
		}
		records = filterByIDs(seedRecords, vehicleIDs)
	}
	s.dispatchLiveEvents(request, now, vehicleIDs)

	page, pagination, err := paginate(records, request.URL.Query(), 512)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|vehicle-stats", payload)
}

func (s *Server) dispatchLiveEvents(request *http.Request, at time.Time, vehicleIDs []string) {
	if s.live == nil {
		return
	}

	activeEvents := s.live.ActiveEvents(at, nil, vehicleIDs)
	for idx := range activeEvents {
		event := &activeEvents[idx]
		webhookEventType := s.live.EventWebhookType(event.Type)
		uniqueKey := webhookEventType + "|" + event.ID
		s.dispatchEventOnce(
			request,
			uniqueKey,
			webhookEventType,
			s.live.EventWebhookPayload(event),
		)
	}
}

func (s *Server) handleWebhookList(writer http.ResponseWriter, request *http.Request) {
	records, err := s.store.List(ResourceWebhooks)
	if err != nil {
		s.writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}

	filtered := filterByIDs(records, idsFromQuery(request.URL.Query(), "ids"))
	page, pagination, err := paginate(filtered, request.URL.Query(), 512)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|webhook-list", payload)
}

func (s *Server) handleWebhookGet(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	record, err := s.store.Get(ResourceWebhooks, id)
	if err != nil {
		s.respondStoreError(writer, err)
		return
	}
	s.respondJSON(writer, request, requestSignature(request)+"|webhook-get", record)
}

func (s *Server) handleWebhookCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	created, err := s.store.Create(ResourceWebhooks, body)
	if err != nil {
		s.respondStoreError(writer, err)
		return
	}
	s.respondJSON(writer, request, requestSignature(request)+"|webhook-create", created)
}

func (s *Server) handleWebhookPatch(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	body, err := readRecordBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	updated, err := s.store.Patch(ResourceWebhooks, id, body)
	if err != nil {
		s.respondStoreError(writer, err)
		return
	}
	s.respondJSON(writer, request, requestSignature(request)+"|webhook-patch", updated)
}

func (s *Server) handleWebhookDelete(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	if err = s.store.Delete(ResourceWebhooks, id); err != nil {
		s.respondStoreError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLiveShareList(writer http.ResponseWriter, request *http.Request) {
	records, err := s.store.List(ResourceLiveShares)
	if err != nil {
		s.writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}

	filtered := filterByIDs(records, idsFromQuery(request.URL.Query(), "ids"))
	page, pagination, err := paginate(filtered, request.URL.Query(), 100)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|live-share-list", payload)
}

func (s *Server) handleLiveShareCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	created, err := s.store.Create(ResourceLiveShares, body)
	if err != nil {
		s.respondStoreError(writer, err)
		return
	}
	payload := map[string]any{"data": created}
	s.respondJSON(writer, request, requestSignature(request)+"|live-share-create", payload)
}

func (s *Server) handleLiveSharePatch(writer http.ResponseWriter, request *http.Request) {
	id, err := queryID(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	body, err := readRecordBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	updated, err := s.store.Patch(ResourceLiveShares, id, body)
	if err != nil {
		s.respondStoreError(writer, err)
		return
	}
	payload := map[string]any{"data": updated}
	s.respondJSON(writer, request, requestSignature(request)+"|live-share-patch", payload)
}

func (s *Server) handleLiveShareDelete(writer http.ResponseWriter, request *http.Request) {
	id, err := queryID(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	if err = s.store.Delete(ResourceLiveShares, id); err != nil {
		s.respondStoreError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func filterHOSLogsByTimeWindow(records []Record, startTime, endTime *time.Time) []Record {
	if startTime == nil && endTime == nil {
		return cloneRecords(records)
	}

	filtered := make([]Record, 0, len(records))
	for _, record := range records {
		cloned := cloneRecord(record)
		rawLogs, logsOK := cloned["hosLogs"].([]any)
		if !logsOK {
			filtered = append(filtered, cloned)
			continue
		}

		nextLogs := make([]any, 0, len(rawLogs))
		for _, entry := range rawLogs {
			logEntry, entryOK := entry.(map[string]any)
			if !entryOK {
				continue
			}
			startValue, _ := logEntry["logStartTime"].(string)
			if startValue == "" {
				continue
			}
			parsed, err := time.Parse(time.RFC3339, startValue)
			if err != nil {
				continue
			}
			parsed = parsed.UTC()
			if startTime != nil && parsed.Before(*startTime) {
				continue
			}
			if endTime != nil && parsed.After(*endTime) {
				continue
			}
			nextLogs = append(nextLogs, cloneAny(logEntry))
		}

		cloned["hosLogs"] = nextLogs
		filtered = append(filtered, cloned)
	}
	return filtered
}
