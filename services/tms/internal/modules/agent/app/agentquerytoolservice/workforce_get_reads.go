package agentquerytoolservice

import (
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	toolWorkerGet           = "worker_get"
	toolWorkerGetCredential = "worker_get_credential"
	toolWorkerGetHOS        = "worker_get_hos"
	toolWorkerGetDOTDraw    = "worker_get_dot_random_draw"

	paramCredentialID = "credentialId" //nolint:gosec // G101: a parameter name, not a credential
	paramDrawID       = "drawId"
	dutyStatusUnknown = "Unknown"

	hosRecordedAt   = "recordedAt"
	hosStale        = "stale"
	hosDutyStatus   = "dutyStatus"
	hosMillis       = "Ms"
	hosMinutes      = "Minutes"
	hosStaleSeconds = 12 * 3600
	millisPerMinute = 60_000

	workerGetSummary = "Retrieve one worker (driver) by id, including their profile and " +
		"current state. Use worker_list with a query first when you only have a name."
	workerCredentialGetSummary = "Retrieve one worker credential by id: its type, the " +
		"worker who holds it, its status, and when it was issued and expires. It also says " +
		"whether it was verified and whether a document is on file. Use " +
		"worker_list_credentials first when you do not have an id."
	workerHOSGetSummary = "A driver's hours of service as the ELD last reported them: duty " +
		"status and the drive, shift, cycle and break time remaining, in minutes. Use it " +
		"before promising a delivery time or proposing an assignment. A reading older than " +
		"twelve hours is marked stale and should not be planned on."
	dotDrawGetSummary = "Retrieve one random testing round with every driver it selected. " +
		"Each selection has its substance, rank and where it stands: selected, notified, " +
		"completed with its test, excused or missed. Each selection's id is the drawEntryId " +
		"worker_schedule_dot_test and worker_update_dot_random_selection take."

	// unknownHoursNote keeps an unreported clock from reading as a refusal:
	// the prompt asks for what the person named to be proposed with its
	// concerns beside it, and "plan on nothing" stopped an assignment the
	// person asked for.
	unknownHoursNote = "Unknown hours do not stop work the person asked for: propose it " +
		"and say beside it that the hours are unreported, for them to weigh."
	noELDNote = "No ELD has reported hours of service for this driver, so their drive, " +
		"shift and cycle time are unknown: the driver may not be mapped to the telematics " +
		"provider. Do not estimate hours or call this tool again. " + unknownHoursNote
	staleHoursNote = "This reading is more than twelve hours old; treat the clocks as " +
		"unknown. " + unknownHoursNote
)

// workerGetSpec reads one worker through the Desk's worker card.
func workerGetSpec() getSpec {
	return getSpec{
		name:      toolWorkerGet,
		entity:    fieldWorker,
		resource:  permission.ResourceWorker,
		summary:   workerGetSummary,
		paramName: paramWorkerID,
		idSource:  "from worker_list, the page you are on, or this run's subject",
		shape:     shapeWorkerRow,
	}
}

// workerCredentialGetSpec reads one credential through the operation the
// credential's Desk read sends. It never selects the number, which is
// Confidential.
func workerCredentialGetSpec() getSpec {
	return getSpec{
		name:      toolWorkerGetCredential,
		entity:    "worker credential",
		resource:  permission.ResourceWorkerCredential,
		summary:   workerCredentialGetSummary,
		paramName: paramCredentialID,
		idSource:  "from worker_list_credentials or the page you are on",
	}
}

// dotRandomDrawGetSpec reads one random testing round with its selections,
// as the random testing page reads it.
func dotRandomDrawGetSpec() getSpec {
	return getSpec{
		name:        toolWorkerGetDOTDraw,
		entity:      "random testing round",
		searchTerms: []string{"who was selected", "selected drivers", "draw results"},
		resource:    permission.ResourceDOTRandomPool,
		kinds:       []permission.RecordKind{permission.KindDOTRandomDraw},
		summary:     dotDrawGetSummary,
		paramName:   paramDrawID,
		idSource:    "from worker_list_dot_random_draws. Never guess one",
	}
}

// workerHOSGetSpec reads a driver's clocks as the telematics page reads them,
// in minutes, with a stale reading or no reading at all named as unknown.
func workerHOSGetSpec() getSpec {
	return getSpec{
		name:   toolWorkerGetHOS,
		entity: entityDriver,
		searchTerms: []string{
			"hours of service", "hos", "hours left", "drive time", "duty status",
		},
		resource:  permission.ResourceWorker,
		summary:   workerHOSGetSummary,
		paramName: paramWorkerID,
		idSource:  "from worker_list",
		shape:     shapeHoursOfService,
		absent:    unreportedHours,
	}
}

// shapeHoursOfService turns each clock into whole minutes, the unit the
// summary promises, writes when the reading was taken in the organization's
// timezone, and marks a reading over twelve hours old as stale.
func shapeHoursOfService(
	record map[string]any,
	_ map[string]struct{},
	clk clock,
) {
	for _, key := range slices.Collect(maps.Keys(record)) {
		clock, isClock := strings.CutSuffix(key, hosMillis)
		millis := optionalInt(record, key, -1)
		if !isClock || millis < 0 {
			continue
		}
		delete(record, key)
		record[clock+hosMinutes] = millis / millisPerMinute
	}
	recordedAt := int64(optionalInt(record, hosRecordedAt, 0))
	stale := clk.Instant()-recordedAt > hosStaleSeconds
	record[hosStale] = stale
	if recordedAt > 0 {
		record[hosRecordedAt] = timeutils.FormatUnixDateTimeIn(recordedAt, clk.Timezone)
	}
	if stale {
		record[outcomeNote] = staleHoursNote
	}
}

// unreportedHours is a driver no ELD has reported for: their clocks are
// unknown, which is the finding, not a missing record.
func unreportedHours(workerID pulid.ID) map[string]any {
	return map[string]any{
		paramWorkerID: workerID.String(),
		hosDutyStatus: dutyStatusUnknown,
		hosRecordedAt: "never",
		hosStale:      true,
		outcomeNote:   noELDNote,
	}
}
