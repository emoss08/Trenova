package sim

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

func (s *Server) registerAdminRoutes() {
	s.mux.HandleFunc("GET /_sim/health", s.handleHealth)
	s.mux.HandleFunc("GET /_sim/map", s.handleMapView)
	s.mux.HandleFunc("GET /_sim/time", s.handleTimeGet)
	s.mux.HandleFunc("PUT /_sim/time", s.handleTimePut)
	s.mux.HandleFunc("POST /_sim/time/step", s.handleTimeStep)
	s.mux.HandleFunc("GET /_sim/scripts/status", s.handleScriptStatus)
	s.mux.HandleFunc("GET /_sim/faults", s.handleFaultList)
	s.mux.HandleFunc("PUT /_sim/faults", s.handleFaultReplace)
	s.mux.HandleFunc("POST /_sim/faults/rules", s.handleFaultCreate)
	s.mux.HandleFunc("DELETE /_sim/faults/rules/{id}", s.handleFaultDelete)
	s.mux.HandleFunc("POST /_sim/faults/reset", s.handleFaultReset)
	s.mux.HandleFunc("GET /_sim/assets/routes", s.handleAssetRouteGeometry)
	s.mux.HandleFunc("GET /_sim/state/summary", s.handleStateSummary)
	s.mux.HandleFunc("POST /_sim/state/reset", s.handleStateReset)
	s.mux.HandleFunc("GET /_sim/scenarios", s.handleScenarioList)
	s.mux.HandleFunc("GET /_sim/scenarios/active", s.handleScenarioActiveGet)
	s.mux.HandleFunc("PUT /_sim/scenarios/active", s.handleScenarioActivePut)
	s.mux.HandleFunc("GET /_sim/events/active", s.handleActiveEvents)
	s.mux.HandleFunc("GET /_sim/events/window", s.handleEventsWindow)
	s.mux.HandleFunc("POST /_sim/events/trigger", s.handleEventTrigger)
	s.mux.HandleFunc("POST /_sim/webhooks/inbox", s.handleWebhookInboxCapture)
	s.mux.HandleFunc("GET /_sim/webhooks/inbox", s.handleWebhookInboxList)
	s.mux.HandleFunc("DELETE /_sim/webhooks/inbox", s.handleWebhookInboxReset)
}

func (s *Server) handleHealth(writer http.ResponseWriter, request *http.Request) {
	_ = request
	payload := map[string]any{
		"status": "ok",
	}
	if err := writeJSON(writer, http.StatusOK, payload); err != nil {
		s.logger.Error("failed to write health response", "error", err.Error())
	}
}

func (s *Server) handleStateSummary(writer http.ResponseWriter, request *http.Request) {
	summary := s.store.Summary()
	if s.live != nil {
		byType, violations, speeding := s.live.ActiveEventSummary(s.simNow())
		summary.ActiveEventsByType = byType
		summary.ViolationsActive = violations
		summary.SpeedingActive = speeding
	}
	payload := map[string]any{"data": summary}
	s.respondJSON(writer, request, requestSignature(request)+"|state-summary", payload)
}

func (s *Server) handleAssetRouteGeometry(writer http.ResponseWriter, request *http.Request) {
	records, err := s.store.List(ResourceAssetLocation)
	if err != nil {
		s.writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}

	filtered := filterAssetLocationRecords(
		records,
		idsFromQuery(request.URL.Query(), "ids"),
		nil,
		nil,
	)
	grouped := map[string][]map[string]any{}
	for _, record := range filtered {
		assetID := nestedString(record, "asset", "id")
		if assetID == "" {
			continue
		}

		location, ok := anyAsMap(record["location"])
		if !ok {
			continue
		}
		lat := floatFromAny(location["latitude"])
		lon := floatFromAny(location["longitude"])
		if !isReasonableCoordinate(lat, lon) {
			continue
		}

		grouped[assetID] = append(grouped[assetID], map[string]any{
			"latitude":  lat,
			"longitude": lon,
			"time":      stringValue(record, "happenedAtTime"),
		})
	}

	assetIDs := make([]string, 0, len(grouped))
	for assetID := range grouped {
		assetIDs = append(assetIDs, assetID)
	}
	sort.Strings(assetIDs)

	data := make([]any, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		points := grouped[assetID]
		sort.Slice(points, func(i, j int) bool {
			return stringValue(Record(points[i]), "time") <
				stringValue(Record(points[j]), "time")
		})
		data = append(data, map[string]any{
			"assetId": assetID,
			"points":  points,
		})
	}

	payload := map[string]any{
		"data": data,
	}
	if encodeErr := writeJSON(writer, http.StatusOK, payload); encodeErr != nil {
		s.logger.Error("failed to write asset route geometry", "error", encodeErr.Error())
	}
}

func (s *Server) handleStateReset(writer http.ResponseWriter, request *http.Request) {
	s.store.Reset()
	s.eventMu.Lock()
	s.eventSentAt = map[string]time.Time{}
	s.eventMu.Unlock()
	s.clearWebhookInbox()
	if s.scripts != nil {
		if err := s.scripts.Reload(); err != nil {
			s.logger.Warn("failed to reload scenario scripts", "error", err.Error())
		}
	}
	payload := map[string]any{
		"data": s.store.Summary(),
	}
	s.respondJSON(writer, request, requestSignature(request)+"|state-reset", payload)
}

func (s *Server) handleScenarioList(writer http.ResponseWriter, request *http.Request) {
	payload := map[string]any{
		"data":   s.scenarios.Profiles(),
		"active": s.scenarios.ActiveProfile(),
	}
	s.respondJSON(writer, request, requestSignature(request)+"|scenario-list", payload)
}

func (s *Server) handleScenarioActiveGet(writer http.ResponseWriter, request *http.Request) {
	payload := map[string]any{
		"profile": s.scenarios.ActiveProfile(),
	}
	s.respondJSON(writer, request, requestSignature(request)+"|scenario-active-get", payload)
}

func (s *Server) handleScenarioActivePut(writer http.ResponseWriter, request *http.Request) {
	body, err := readBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	profileValue, _ := body["profile"].(string)
	profile := strings.TrimSpace(profileValue)
	if profile == "" {
		s.writeAPIError(writer, http.StatusBadRequest, ErrProfileNotFound)
		return
	}

	if err = s.scenarios.SetActiveProfile(profile); err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	payload := map[string]any{
		"profile": s.scenarios.ActiveProfile(),
	}
	s.respondJSON(writer, request, requestSignature(request)+"|scenario-active-put", payload)
}

func (s *Server) handleActiveEvents(writer http.ResponseWriter, request *http.Request) {
	if s.live == nil {
		payload := map[string]any{"data": []any{}}
		s.respondJSON(writer, request, requestSignature(request)+"|events-active", payload)
		return
	}

	driverIDs := idsFromQuery(request.URL.Query(), "driverIds")
	vehicleIDs := idsFromQuery(request.URL.Query(), "vehicleIds")
	at := s.simNow()
	if atRaw := strings.TrimSpace(request.URL.Query().Get("atTime")); atRaw != "" {
		parsed, err := time.Parse(time.RFC3339, atRaw)
		if err != nil {
			s.writeAPIError(writer, http.StatusBadRequest, ErrInvalidBody)
			return
		}
		at = parsed.UTC()
	}

	events := s.live.ActiveEvents(at, driverIDs, vehicleIDs)
	data := make([]any, 0, len(events))
	for idx := range events {
		event := &events[idx]
		data = append(data, event.ToRecord())
	}
	payload := map[string]any{"data": data}
	s.respondJSON(writer, request, requestSignature(request)+"|events-active", payload)
}

func (s *Server) handleEventsWindow(writer http.ResponseWriter, request *http.Request) {
	if s.live == nil {
		payload := map[string]any{"data": []any{}}
		s.respondJSON(writer, request, requestSignature(request)+"|events-window", payload)
		return
	}

	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	now := s.simNow()
	if startTime == nil || endTime == nil {
		defaultStart := now.Add(-12 * time.Hour)
		defaultEnd := now.Add(12 * time.Hour)
		if startTime == nil {
			startTime = &defaultStart
		}
		if endTime == nil {
			endTime = &defaultEnd
		}
	}

	limit := parseLimit(request.URL.Query(), 512)
	driverIDs := idsFromQuery(request.URL.Query(), "driverIds")
	vehicleIDs := idsFromQuery(request.URL.Query(), "vehicleIds")
	events := s.live.EventsWindow(*startTime, *endTime, driverIDs, vehicleIDs, limit)

	data := make([]any, 0, len(events))
	for idx := range events {
		event := &events[idx]
		data = append(data, event.ToRecord())
	}
	payload := map[string]any{"data": data}
	s.respondJSON(writer, request, requestSignature(request)+"|events-window", payload)
}

func (s *Server) handleEventTrigger(writer http.ResponseWriter, request *http.Request) {
	body, err := readBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	eventType, _ := body["eventType"].(string)
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		s.writeAPIError(writer, http.StatusBadRequest, ErrWebhookEventTypeRequired)
		return
	}

	payload := body["payload"]
	if payload == nil {
		payload = map[string]any{}
	}

	profile := s.profileFromContext(request.Context())
	if s.scenarios != nil && s.scenarios.ShouldOmitEvent(profile, eventType, payload) {
		response := map[string]any{
			"status":    "omitted",
			"profile":   profile,
			"eventType": eventType,
		}
		s.respondJSON(writer, request, requestSignature(request)+"|event-trigger", response)
		return
	}
	if s.dispatcher == nil {
		response := map[string]any{
			"status":    "disabled",
			"profile":   profile,
			"eventType": eventType,
		}
		s.respondJSON(writer, request, requestSignature(request)+"|event-trigger", response)
		return
	}

	if err = s.dispatcher.Dispatch(profile, eventType, payload); err != nil &&
		!errors.Is(err, ErrWebhookQueueSaturated) {
		s.writeAPIError(writer, http.StatusInternalServerError, err)
		return
	}
	if errors.Is(err, ErrWebhookQueueSaturated) {
		s.writeAPIError(writer, http.StatusTooManyRequests, err)
		return
	}

	response := map[string]any{
		"status":    "queued",
		"eventType": eventType,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|event-trigger", response)
}

func (s *Server) handleWebhookInboxCapture(writer http.ResponseWriter, request *http.Request) {
	body, err := readBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	record := Record{
		"receivedAtTime": s.simNow().UTC().Format(time.RFC3339),
		"eventType":      strings.TrimSpace(stringValue(body, "eventType")),
		"eventTime":      strings.TrimSpace(stringValue(body, "eventTime")),
		"payload":        cloneAny(body),
		"delivery": map[string]any{
			"id":       strings.TrimSpace(request.Header.Get("X-Samsara-Sim-Delivery-Id")),
			"sequence": strings.TrimSpace(request.Header.Get("X-Samsara-Sim-Delivery-Sequence")),
			"attempt":  strings.TrimSpace(request.Header.Get("X-Samsara-Sim-Delivery-Attempt")),
		},
		"signature": map[string]any{
			"timestamp": strings.TrimSpace(request.Header.Get("X-Samsara-Timestamp")),
			"value":     strings.TrimSpace(request.Header.Get("X-Samsara-Signature")),
		},
	}
	s.appendWebhookInbox(record)
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleWebhookInboxList(writer http.ResponseWriter, request *http.Request) {
	limit := parseLimit(request.URL.Query(), 200)
	records := s.listWebhookInbox(limit)
	payload := map[string]any{
		"data": recordsAsAny(records),
	}
	s.respondJSON(writer, request, requestSignature(request)+"|webhook-inbox-list", payload)
}

func (s *Server) handleWebhookInboxReset(writer http.ResponseWriter, request *http.Request) {
	s.clearWebhookInbox()
	payload := map[string]any{
		"data": []any{},
	}
	s.respondJSON(writer, request, requestSignature(request)+"|webhook-inbox-reset", payload)
}

func (s *Server) handleTimeGet(writer http.ResponseWriter, request *http.Request) {
	_ = s.simNow()
	payload := map[string]any{
		"data": s.clockPayload(),
	}
	s.respondJSON(writer, request, requestSignature(request)+"|time-get", payload)
}

func (s *Server) handleTimePut(writer http.ResponseWriter, request *http.Request) {
	body, err := readBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	if rawSetTime, ok := body["setTime"].(string); ok && strings.TrimSpace(rawSetTime) != "" {
		parsed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(rawSetTime))
		if parseErr != nil {
			s.writeAPIError(writer, http.StatusBadRequest, ErrInvalidBody)
			return
		}
		s.clock.SetTime(parsed.UTC())
	}

	if rawSpeed, exists := body["speed"]; exists {
		speedValue := floatFromAny(rawSpeed)
		if speedValue < 0.1 || speedValue > 20 {
			s.writeAPIError(writer, http.StatusBadRequest, ErrClockSpeedInvalid)
			return
		}
		s.clock.SetSpeed(speedValue)
	}

	if rawPaused, ok := body["paused"].(bool); ok {
		s.clock.SetPaused(rawPaused)
	}

	payload := map[string]any{
		"data": s.clockPayload(),
	}
	s.respondJSON(writer, request, requestSignature(request)+"|time-put", payload)
}

func (s *Server) handleTimeStep(writer http.ResponseWriter, request *http.Request) {
	body, err := readBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	durationMs := int64(floatFromAny(body["durationMs"]))
	if durationMs < 1 || durationMs > 86400000 {
		s.writeAPIError(writer, http.StatusBadRequest, ErrClockStepInvalid)
		return
	}
	s.clock.Step(time.Duration(durationMs) * time.Millisecond)

	payload := map[string]any{
		"data": s.clockPayload(),
	}
	s.respondJSON(writer, request, requestSignature(request)+"|time-step", payload)
}

func (s *Server) handleScriptStatus(writer http.ResponseWriter, request *http.Request) {
	status := ScriptStatus{
		Loaded:   false,
		Warnings: []string{"script engine is not configured"},
	}
	if s.scripts != nil {
		status = s.scripts.Status()
	}
	payload := map[string]any{
		"data": status,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|script-status", payload)
}

func (s *Server) handleFaultList(writer http.ResponseWriter, request *http.Request) {
	rules := []FaultRule{}
	if s.faults != nil {
		rules = s.faults.Snapshot()
	}
	payload := map[string]any{
		"data": rules,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|fault-list", payload)
}

func (s *Server) handleFaultReplace(writer http.ResponseWriter, request *http.Request) {
	if s.faults == nil {
		s.writeAPIError(writer, http.StatusInternalServerError, ErrFaultRuleInvalid)
		return
	}

	body, err := readBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	rawRules, ok := body["rules"].([]any)
	if !ok {
		s.writeAPIError(writer, http.StatusBadRequest, ErrInvalidBody)
		return
	}

	rules, decodeErr := decodeFaultRules(rawRules)
	if decodeErr != nil {
		s.writeAPIError(writer, http.StatusBadRequest, decodeErr)
		return
	}
	if err = s.faults.Replace(rules); err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	payload := map[string]any{"data": s.faults.Snapshot()}
	s.respondJSON(writer, request, requestSignature(request)+"|fault-replace", payload)
}

func (s *Server) handleFaultCreate(writer http.ResponseWriter, request *http.Request) {
	if s.faults == nil {
		s.writeAPIError(writer, http.StatusInternalServerError, ErrFaultRuleInvalid)
		return
	}

	body, err := readBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	rule, decodeErr := decodeFaultRuleFromAny(body)
	if decodeErr != nil {
		s.writeAPIError(writer, http.StatusBadRequest, decodeErr)
		return
	}
	created, createErr := s.faults.Add(&rule)
	if createErr != nil {
		s.writeAPIError(writer, http.StatusBadRequest, createErr)
		return
	}

	payload := map[string]any{"data": created}
	s.respondJSON(writer, request, requestSignature(request)+"|fault-create", payload)
}

func (s *Server) handleFaultDelete(writer http.ResponseWriter, request *http.Request) {
	if s.faults == nil {
		s.writeAPIError(writer, http.StatusInternalServerError, ErrFaultRuleInvalid)
		return
	}

	id, err := pathID(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	if !s.faults.Delete(id) {
		s.writeAPIError(writer, http.StatusNotFound, ErrRecordNotFound)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleFaultReset(writer http.ResponseWriter, request *http.Request) {
	if s.faults == nil {
		s.writeAPIError(writer, http.StatusInternalServerError, ErrFaultRuleInvalid)
		return
	}
	s.faults.Reset()
	payload := map[string]any{"data": []any{}}
	s.respondJSON(writer, request, requestSignature(request)+"|fault-reset", payload)
}

func (s *Server) clockPayload() map[string]any {
	snapshot := s.clock.Snapshot()
	return map[string]any{
		"now":          snapshot.Now.UTC().Format(time.RFC3339),
		"paused":       snapshot.Paused,
		"speed":        round(snapshot.Speed, 2),
		"lastWallTime": snapshot.LastWallTime.UTC().Format(time.RFC3339),
		"lastSimTime":  snapshot.LastSimTime.UTC().Format(time.RFC3339),
	}
}

func decodeFaultRules(raw []any) ([]FaultRule, error) {
	rules := make([]FaultRule, 0, len(raw))
	for idx := range raw {
		rule, err := decodeFaultRuleFromAny(raw[idx])
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func decodeFaultRuleFromAny(raw any) (FaultRule, error) {
	bytes, err := sonic.Marshal(raw)
	if err != nil {
		return FaultRule{}, ErrInvalidBody
	}
	rule := FaultRule{}
	if err = sonic.Unmarshal(bytes, &rule); err != nil {
		return FaultRule{}, ErrInvalidBody
	}
	return rule, nil
}
