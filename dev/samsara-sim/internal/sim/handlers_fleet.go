package sim

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

func (s *Server) registerRouteRoutes() {
	s.mux.HandleFunc("GET /fleet/routes", s.handleRouteList)
	s.mux.HandleFunc("POST /fleet/routes", s.handleRouteCreate)
	s.mux.HandleFunc("GET /fleet/routes/{id}", s.handleRouteGet)
	s.mux.HandleFunc("PATCH /fleet/routes/{id}", s.handleRoutePatch)
	s.mux.HandleFunc("DELETE /fleet/routes/{id}", s.handleRouteDelete)
}

func (s *Server) handleRouteList(writer http.ResponseWriter, request *http.Request) {
	routeIDs := idsFromQuery(request.URL.Query(), "ids", "routeIds")
	statusFilters := idsFromQuery(request.URL.Query(), "status", "statuses")
	if len(statusFilters) > 0 && len(normalizeRouteStatusFilters(statusFilters)) == 0 {
		s.writeAPIError(writer, http.StatusBadRequest, ErrInvalidBody)
		return
	}
	now := s.simNow()

	var records []Record
	var err error
	if s.live != nil {
		records = s.live.Routes(now, routeIDs, statusFilters)
	} else {
		records, err = s.store.List(ResourceRoutes)
		if err != nil {
			s.writeAPIError(writer, http.StatusInternalServerError, err)
			return
		}
		records = filterByIDs(records, routeIDs)
		if len(statusFilters) > 0 {
			records = filterRoutesByStatus(records, statusFilters)
		}
	}

	page, pagination, err := paginate(records, request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	payload := map[string]any{
		"data":       recordsAsAny(page),
		"pagination": pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|route-list", payload)
}

func (s *Server) handleRouteGet(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	if s.live != nil {
		record, ok := s.live.RouteByID(s.simNow(), id)
		if !ok {
			s.writeAPIError(writer, http.StatusNotFound, ErrRecordNotFound)
			return
		}
		payload := map[string]any{"data": record}
		s.respondJSON(writer, request, requestSignature(request)+"|route-get", payload)
		return
	}

	record, err := s.store.Get(ResourceRoutes, id)
	if err != nil {
		s.respondStoreError(writer, err)
		return
	}
	payload := map[string]any{"data": record}
	s.respondJSON(writer, request, requestSignature(request)+"|route-get", payload)
}

func (s *Server) handleRouteCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	if strings.TrimSpace(stringValue(body, "name")) == "" {
		now := s.simNow().Format(time.RFC3339)
		body["name"] = "Sim Route " + now
	}

	created, err := s.store.CreateFromAPI(ResourceRoutes, body, CreateOptions{At: s.simNow()})
	if err != nil {
		s.respondStoreError(writer, err)
		return
	}
	s.dispatchEvent(request, "RouteStopArrival", created)

	payload := map[string]any{"data": created}
	s.respondJSON(writer, request, requestSignature(request)+"|route-create", payload)
}

func (s *Server) handleRoutePatch(writer http.ResponseWriter, request *http.Request) {
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

	updated, err := s.store.PatchFromAPI(ResourceRoutes, id, body, s.simNow())
	if err != nil {
		s.respondStoreError(writer, err)
		return
	}
	s.dispatchEvent(request, "RouteStopEtaUpdated", updated)

	payload := map[string]any{"data": updated}
	s.respondJSON(writer, request, requestSignature(request)+"|route-patch", payload)
}

func (s *Server) handleRouteDelete(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	deleteErr := s.store.Delete(ResourceRoutes, id)
	if deleteErr != nil {
		s.respondStoreError(writer, deleteErr)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) respondStoreError(writer http.ResponseWriter, err error) {
	if errors.Is(err, ErrRecordNotFound) {
		s.writeAPIError(writer, http.StatusNotFound, err)
		return
	}
	if errors.Is(err, ErrRecordConflict) {
		s.writeAPIError(writer, http.StatusConflict, err)
		return
	}
	if errors.Is(err, ErrRecordIDRequired) {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}
	s.writeAPIError(writer, http.StatusInternalServerError, err)
}

func parseTimeRange(
	request *http.Request,
) (startTime, endTime *time.Time, err error) {
	values := request.URL.Query()
	startTime, err = parseOptionalTimePtr(values, fieldStartTime)
	if err != nil {
		return nil, nil, err
	}
	endTime, err = parseOptionalTimePtr(values, fieldEndTime)
	if err != nil {
		return nil, nil, err
	}
	if startTime != nil && endTime != nil && endTime.Before(*startTime) {
		return nil, nil, invalidParameter(fieldEndTime, "must not be before startTime")
	}
	return startTime, endTime, nil
}

func filterAssetLocationRecords(
	records []Record,
	assetIDs []string,
	startTime *time.Time,
	endTime *time.Time,
) []Record {
	lookup := map[string]struct{}{}
	for _, id := range assetIDs {
		clean := strings.TrimSpace(id)
		if clean != "" {
			lookup[clean] = struct{}{}
		}
	}

	filtered := make([]Record, 0, len(records))
	for _, record := range records {
		assetID := nestedString(record, "asset", "id")
		if len(lookup) > 0 {
			if _, ok := lookup[assetID]; !ok {
				continue
			}
		}

		if startTime != nil || endTime != nil {
			happened := stringValue(record, "happenedAtTime")
			if happened == "" {
				continue
			}
			parsed, err := time.Parse(time.RFC3339, happened)
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
		}

		filtered = append(filtered, cloneRecord(record))
	}
	return filtered
}

func filterRoutesByStatus(records []Record, statuses []string) []Record {
	allowed := normalizeRouteStatusFilters(statuses)
	if len(allowed) == 0 {
		return cloneRecords(records)
	}

	filtered := make([]Record, 0, len(records))
	for _, record := range records {
		status := strings.ToLower(strings.TrimSpace(stringValue(record, "status")))
		if status == "" {
			if lifecycle, ok := anyAsMap(record["lifecycle"]); ok {
				if value, okStatus := lifecycle["status"].(string); okStatus {
					status = strings.ToLower(strings.TrimSpace(value))
				}
			}
		}
		if _, ok := allowed[status]; !ok {
			continue
		}
		filtered = append(filtered, cloneRecord(record))
	}
	return filtered
}
