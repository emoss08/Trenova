package sim

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	hosViolationTypeRestBreak    = "restbreakMissed"
	hosViolationTypeShift        = "shiftHours"
	hosViolationTypeShiftDriving = "shiftDrivingHours"
	hosViolationTypeCycle        = "cycleHoursOn"
)

func hosViolationTypeForSimEvent(simEventType string) string {
	switch simEventType {
	case simEventViolationBreak:
		return hosViolationTypeRestBreak
	case simEventViolationShift:
		return hosViolationTypeShift
	case simEventViolationDrive:
		return hosViolationTypeShiftDriving
	case simEventViolationCycle:
		return hosViolationTypeCycle
	default:
		return ""
	}
}

func hosViolationDescription(violationType string) string {
	switch violationType {
	case hosViolationTypeRestBreak:
		return "Rest Break Missed Violation"
	case hosViolationTypeShift:
		return "Shift Hours Violation"
	case hosViolationTypeShiftDriving:
		return "Shift Driving Hours Violation"
	case hosViolationTypeCycle:
		return "Cycle Hours On Violation"
	default:
		return "HOS Violation"
	}
}

func (l *LiveSimulator) HOSViolations(
	windowStart time.Time,
	windowEnd time.Time,
	driverIDs []string,
	violationTypes []string,
) []Record {
	allowedTypes := toStringSet(violationTypes)
	roster := l.loadDriverRoster()
	events := l.EventsWindow(windowStart, windowEnd, driverIDs, nil, 0)

	out := make([]Record, 0, len(events))
	for idx := range events {
		event := &events[idx]
		violationType := hosViolationTypeForSimEvent(event.Type)
		if violationType == "" {
			continue
		}
		if !matchesStringFilter(allowedTypes, violationType) {
			continue
		}
		out = append(out, l.hosViolationRecord(event, violationType, roster))
	}

	sort.Slice(out, func(i, j int) bool {
		left := stringValue(out[i], "violationStartTime")
		right := stringValue(out[j], "violationStartTime")
		if left == right {
			return recordID(out[i]) < recordID(out[j])
		}
		return left < right
	})
	return out
}

func (l *LiveSimulator) hosViolationRecord(
	event *SimEvent,
	violationType string,
	roster map[string]driverRoster,
) Record {
	driverID := strings.TrimSpace(event.DriverID)
	dayStart := event.StartsAt.UTC().Truncate(24 * time.Hour)
	dayCtx := l.buildDailyEventContext(driverID, event.VehicleID, dayStart)
	workdayStart := l.shiftStartForDay(driverID, dayStart)

	return Record{
		"id":                 event.ID,
		"type":               violationType,
		"description":        hosViolationDescription(violationType),
		"durationMs":         event.EndsAt.Sub(event.StartsAt).Milliseconds(),
		"violationStartTime": event.StartsAt.UTC().Format(time.RFC3339),
		"driver": map[string]any{
			"id":   driverID,
			"name": firstNonEmpty(roster[driverID].Name, driverID),
		},
		"day": map[string]any{
			"startTime": workdayStart.UTC().Format(time.RFC3339),
			"endTime":   dayCtx.DayEnd.UTC().Format(time.RFC3339),
		},
	}
}

func (s *Server) handleHOSViolationList(writer http.ResponseWriter, request *http.Request) {
	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeAPIError(writer, http.StatusBadRequest, err)
		return
	}

	now := s.simNow()
	if endTime == nil {
		windowEnd := now
		endTime = &windowEnd
	}
	if startTime == nil {
		windowStart := endTime.Add(-24 * time.Hour)
		startTime = &windowStart
	}

	driverIDs := idsFromQuery(request.URL.Query(), "driverIds")
	violationTypes := idsFromQuery(request.URL.Query(), "types")

	records := []Record{}
	if s.live != nil {
		records = s.live.HOSViolations(*startTime, *endTime, driverIDs, violationTypes)
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
	s.respondJSON(writer, request, requestSignature(request)+"|hos-violations", payload)
}
