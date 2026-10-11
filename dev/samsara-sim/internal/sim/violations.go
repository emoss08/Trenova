package sim

import (
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/stringutils"
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

var hosViolationTypes = []string{
	"NONE",
	"californiaMealbreakMissed",
	hosViolationTypeCycle,
	"cycleOffHoursAfterOnDutyHours",
	"dailyDrivingHours",
	"dailyOffDutyDeferralAddToDay2Consecutive",
	"dailyOffDutyDeferralNotPartMandatory",
	"dailyOffDutyDeferralTwoDayDrivingLimit",
	"dailyOffDutyDeferralTwoDayOffDuty",
	"dailyOffDutyNonResetHours",
	"dailyOffDutyTotalHours",
	"dailyOnDutyHours",
	"mandatory24HoursOffDuty",
	hosViolationTypeRestBreak,
	hosViolationTypeShiftDriving,
	hosViolationTypeShift,
	"shiftOnDutyHours",
	"unsubmittedLogs",
}

type hosRulesetLimits struct {
	Region     string
	ShiftHours int
	DriveHours int
	CycleHours int
}

func hosRulesetLimitsFor(driver Record) hosRulesetLimits {
	limits := hosRulesetLimits{Region: "USA Property", ShiftHours: 14, DriveHours: 11, CycleHours: 70}
	rulesets := listOf(nestedAny(driver, fieldEldSettings, "rulesets"))
	if len(rulesets) == 0 {
		rulesets = listOf(nestedAny(Record(eldSettingsFor(driver)), "rulesets"))
	}
	if len(rulesets) == 0 {
		return limits
	}
	ruleset := Record(mapOf(rulesets[0]))
	switch stringValue(ruleset, "shift") {
	case "US Interstate Passenger":
		limits = hosRulesetLimits{Region: "USA Passenger", ShiftHours: 15, DriveHours: 10}
	case "Texas Intrastate":
		limits = hosRulesetLimits{Region: "Texas Intrastate", ShiftHours: 15, DriveHours: 12}
	}
	limits.CycleHours = 70
	cycle := stringValue(ruleset, keyCycle)
	if fields := strings.Fields(cycle); len(fields) >= 2 {
		if hours, err := strconv.Atoi(fields[1]); err == nil {
			limits.CycleHours = hours
		}
	}
	return limits
}

func hosViolationDescription(violationType string, limits hosRulesetLimits) string {
	switch violationType {
	case hosViolationTypeRestBreak:
		return "Rest Break Missed (8 hours)"
	case hosViolationTypeShift:
		return fmt.Sprintf("Shift Hours (%s-%d hours)", limits.Region, limits.ShiftHours)
	case hosViolationTypeShiftDriving:
		return fmt.Sprintf("Shift Driving Hours (%s-%d hours)", limits.Region, limits.DriveHours)
	case hosViolationTypeCycle:
		return fmt.Sprintf("Cycle Hours On (%s-%d hours)", limits.Region, limits.CycleHours)
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
	snapshot := l.fleet()
	roster := l.loadDriverRoster()
	events := l.EventsWindow(windowStart, windowEnd.Add(time.Second), driverIDs, nil, 0)

	out := make([]Record, 0, len(events))
	for idx := range events {
		event := &events[idx]
		violationType := hosViolationTypeForSimEvent(event.Type)
		if violationType == "" || !matchesStringFilter(allowedTypes, violationType) {
			continue
		}
		reportedStart := event.StartsAt.UTC().Truncate(time.Second)
		if reportedStart.Before(windowStart) || reportedStart.After(windowEnd) {
			continue
		}
		driverID := strings.TrimSpace(event.DriverID)
		driver := snapshot.driverByID[driverID]
		dayStart, dayEnd := driverLogDayAt(driver, event.StartsAt)
		name := stringutils.FirstNonEmptyTrimmed(
			stringValue(driver, keyName),
			roster[driverID].Name,
			driverID,
		)
		driverRef := map[string]any{keyID: driverID, keyName: name}
		if externalIDs := externalIDsOf(driver); len(externalIDs) > 0 {
			driverRef[fieldExternalIDs] = renderExternalIDs(driver, nil)
		}
		out = append(out, Record{
			keyID:                event.ID,
			keyType:              violationType,
			keyDescription:       hosViolationDescription(violationType, hosRulesetLimitsFor(driver)),
			"durationMs":         event.EndsAt.Sub(event.StartsAt).Milliseconds(),
			"violationStartTime": reportedStart.Format(time.RFC3339),
			keyDriver:            driverRef,
			"day": map[string]any{
				fieldStartTime: dayStart.Format(time.RFC3339),
				fieldEndTime:   dayEnd.Format(time.RFC3339),
			},
		})
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

func (s *Server) handleHOSViolationList(writer http.ResponseWriter, request *http.Request) {
	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	values := request.URL.Query()
	view := s.fleetView()
	windowStart, windowEnd := view.now, view.now
	if startTime != nil {
		windowStart = *startTime
	}
	if endTime != nil {
		windowEnd = *endTime
	}
	if windowEnd.Before(windowStart) {
		s.writeError(writer, invalidParameter(fieldEndTime, "must not be before startTime"))
		return
	}
	violationTypes := csvQueryValues(values, paramTypes)
	for _, violationType := range violationTypes {
		if !slices.Contains(hosViolationTypes, violationType) {
			s.writeError(writer, invalidParameter(
				paramTypes,
				fmt.Sprintf("%q is not a supported violation type", violationType),
			))
			return
		}
	}
	drivers, err := view.hosDrivers(values, hosDriverQuery{ExternalRefs: true})
	if err != nil {
		s.writeError(writer, err)
		return
	}

	records := []Record{}
	if len(drivers) > 0 {
		records = s.live.HOSViolations(windowStart, windowEnd, driverIDsOf(drivers), violationTypes)
	}
	page, pagination, err := paginate(records, request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	payload := map[string]any{
		keyData:       groupHOSViolationsByDriver(page),
		keyPagination: pagination,
	}
	s.respondJSON(writer, request, requestSignature(request)+"|hos-violations", payload)
}

func groupHOSViolationsByDriver(records []Record) []any {
	groupIndex := make(map[string]int, len(records))
	grouped := make([][]any, 0, len(records))
	for _, record := range records {
		violation := cloneRecord(record)
		delete(violation, "id")

		driverID := nestedString(record, "driver", "id")
		idx, ok := groupIndex[driverID]
		if !ok {
			idx = len(grouped)
			groupIndex[driverID] = idx
			grouped = append(grouped, make([]any, 0, 1))
		}
		grouped[idx] = append(grouped[idx], violation)
	}

	data := make([]any, 0, len(grouped))
	for _, violations := range grouped {
		data = append(data, map[string]any{"violations": violations})
	}
	return data
}
