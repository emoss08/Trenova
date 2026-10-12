package sim

import (
	"net/http"
	"time"
)

func (s *Server) registerComplianceRoutes() {
	s.mux.HandleFunc("GET /fleet/hos/clocks", s.handleHOSClockList)
	s.mux.HandleFunc("GET /fleet/hos/logs", s.handleHOSLogList)
	s.mux.HandleFunc("GET /fleet/hos/daily-logs", s.handleHOSDailyLogList)
	s.mux.HandleFunc("GET /fleet/hos/violations", s.handleHOSViolationList)
	s.mux.HandleFunc("GET /fleet/drivers/tachograph-files/history", s.handleDriverTachograph)
	s.mux.HandleFunc("GET /fleet/vehicles/tachograph-files/history", s.handleVehicleTachograph)
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
	page, pagination, err := paginate(records, request)
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
	page, pagination, err := paginate(records, request)
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

func (s *Server) dispatchLiveEvents(request *http.Request, at time.Time, vehicleIDs []string) {
	if s.live == nil {
		return
	}

	emissions := s.live.WebhookEmissionsForWindow(at, vehicleIDs)
	for idx := range emissions {
		emission := &emissions[idx]
		s.dispatchEventOnce(request, emission.UniqueKey, emission.EventType, emission.Data)
	}
	s.dispatchGeofenceEvents(request, at)
	s.dispatchRouteStopEvents(request, at)
	s.dispatchDvirEvents(request, at)
	s.dispatchFormEvents(request, at)
	s.dispatchDocumentEvents(request, at)
}

func (s *Server) dispatchRouteStopEvents(request *http.Request, at time.Time) {
	if s.live == nil {
		return
	}

	windowStart := s.routeStopWindow.nextStart(at, defaultAssetLookback, geofenceMaxWindowLookback)
	emissions := s.live.RouteStopWebhookEmissions(at, windowStart, at, nil)
	for idx := range emissions {
		emission := &emissions[idx]
		s.dispatchEventOnce(request, emission.UniqueKey, emission.EventType, emission.Data)
	}
}

func (s *Server) dispatchGeofenceEvents(request *http.Request, at time.Time) {
	if s.live == nil {
		return
	}

	windowStart := s.geofenceWindow.nextStart(at, defaultAssetLookback, geofenceMaxWindowLookback)
	emissions := s.live.GeofenceWebhookEmissions(at, windowStart, at, nil)
	for idx := range emissions {
		emission := &emissions[idx]
		if !s.dispatchEventOnce(request, emission.UniqueKey, emission.EventType, emission.Data) {
			continue
		}
		if emission.EventType == geofenceEventEntry {
			s.geofenceEntries.Add(1)
		} else {
			s.geofenceExits.Add(1)
		}
	}
}

func (s *Server) dispatchDvirEvents(request *http.Request, at time.Time) {
	if s.live == nil {
		return
	}

	windowStart := s.dvirWindow.nextStart(at, defaultAssetLookback, geofenceMaxWindowLookback)
	emissions := s.live.DvirWebhookEmissions(at, windowStart, at)
	for idx := range emissions {
		emission := &emissions[idx]
		s.dispatchEventOnce(request, emission.UniqueKey, emission.EventType, emission.Data)
	}
}

func (s *Server) dispatchFormEvents(request *http.Request, at time.Time) {
	if s.live == nil {
		return
	}

	windowStart := s.formWindow.nextStart(at, defaultAssetLookback, geofenceMaxWindowLookback)
	emissions := s.live.FormWebhookEmissions(at, windowStart, at)
	for idx := range emissions {
		emission := &emissions[idx]
		s.dispatchEventOnce(request, emission.UniqueKey, emission.EventType, emission.Data)
	}
}
