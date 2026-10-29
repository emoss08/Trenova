package workerquerytools

import (
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	toolWorkerGet           = "worker_get"
	ToolWorkerGetCredential = "worker_get_credential"
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

// GetSpec reads one worker through the Desk's worker card.
func GetSpec() readspec.Get {
	return readspec.Get{
		Name:      toolWorkerGet,
		Entity:    fieldWorker,
		Resource:  permission.ResourceWorker,
		Summary:   workerGetSummary,
		ParamName: querytool.ParamWorkerID,
		IDSource:  "from worker_list, the page you are on, or this run's subject",
		Shape:     shapeWorkerRow,
	}
}

// CredentialGetSpec reads one credential through the operation the
// credential's Desk read sends. It never selects the number, which is
// Confidential.
func CredentialGetSpec() readspec.Get {
	return readspec.Get{
		Name:      ToolWorkerGetCredential,
		Entity:    "worker credential",
		Resource:  permission.ResourceWorkerCredential,
		Summary:   workerCredentialGetSummary,
		ParamName: paramCredentialID,
		IDSource:  "from worker_list_credentials or the page you are on",
	}
}

// DOTRandomDrawGetSpec reads one random testing round with its selections,
// as the random testing page reads it.
func DOTRandomDrawGetSpec() readspec.Get {
	return readspec.Get{
		Name:        toolWorkerGetDOTDraw,
		Entity:      "random testing round",
		SearchTerms: []string{"who was selected", "selected drivers", "draw results"},
		Resource:    permission.ResourceDOTRandomPool,
		Kinds:       []permission.RecordKind{permission.KindDOTRandomDraw},
		Summary:     dotDrawGetSummary,
		ParamName:   paramDrawID,
		IDSource:    "from worker_list_dot_random_draws. Never guess one",
	}
}

// HOSGetSpec reads a driver's clocks as the telematics page reads them,
// in minutes, with a stale reading or no reading at all named as unknown.
func HOSGetSpec() readspec.Get {
	return readspec.Get{
		Name:   toolWorkerGetHOS,
		Entity: querytool.EntityDriver,
		SearchTerms: []string{
			"hours of service", "hos", "hours left", "drive time", "duty status",
		},
		Resource:  permission.ResourceWorker,
		Summary:   workerHOSGetSummary,
		ParamName: querytool.ParamWorkerID,
		IDSource:  "from worker_list",
		Shape:     shapeHoursOfService,
		Absent:    unreportedHours,
	}
}

// shapeHoursOfService turns each clock into whole minutes, the unit the
// summary promises, writes when the reading was taken in the organization's
// timezone, and marks a reading over twelve hours old as stale.
func shapeHoursOfService(
	record map[string]any,
	_ map[string]struct{},
	clk filtercatalog.Clock,
) {
	for _, key := range slices.Collect(maps.Keys(record)) {
		clock, isClock := strings.CutSuffix(key, hosMillis)
		millis := querytool.OptionalInt(record, key, -1)
		if !isClock || millis < 0 {
			continue
		}
		delete(record, key)
		record[clock+hosMinutes] = millis / millisPerMinute
	}
	recordedAt := int64(querytool.OptionalInt(record, hosRecordedAt, 0))
	stale := clk.Instant()-recordedAt > hosStaleSeconds
	record[hosStale] = stale
	if recordedAt > 0 {
		record[hosRecordedAt] = timeutils.FormatUnixDateTimeIn(recordedAt, clk.Timezone)
	}
	if stale {
		record[graphqlread.OutcomeNote] = staleHoursNote
	}
}

// unreportedHours is a driver no ELD has reported for: their clocks are
// unknown, which is the finding, not a missing record.
func unreportedHours(workerID pulid.ID) map[string]any {
	return map[string]any{
		querytool.ParamWorkerID: workerID.String(),
		hosDutyStatus:           dutyStatusUnknown,
		hosRecordedAt:           "never",
		hosStale:                true,
		graphqlread.OutcomeNote: noELDNote,
	}
}
