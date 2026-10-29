package workerquerytools

import (
	"cmp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

const (
	workersSummary = "Find workers (drivers) by name, closest first, or the roster by employment " +
		"or qualification. Lacking a credential: withoutCredential; due: " +
		"worker_list_credentials. \"none on file\" was never recorded, worse than expiring " +
		"soon. dispatchable: can take a load now."
	workersQueryNote = "A worker's name, first, last or both; each word matches the start of a " +
		"name and the closest names come first. Omit it to list the roster or to filter " +
		"on a field."
	noteWorkerStatus  = "Not availability; see dispatchable and worker_list_time_off."
	noteCanBeAssigned = "false means the driver is already off dispatch (a dispatch hold, " +
		"leave, suspension or separation), which worker_place_dispatch_hold refuses; true " +
		"with dispatchable false is a driver kept off by something else, such as " +
		"credentials, who can still be held."
	noteDispatchable = "true for drivers a load may be given now: on dispatch, not " +
		"blocked or marked unavailable, active and with compliant credentials; false for " +
		"the rest, whose notDispatchableBecause says why."
	noteWorkerQualified   = "false means a credential lapsed or is missing."
	pluralWorkers         = "workers"
	workerProfileField    = "profile"
	workerFirstNameField  = "firstName"
	workerLastNameField   = "lastName"
	workerAssignable      = "canBeAssigned"
	workerBlockedField    = "assignmentBlocked"
	workerComplianceKey   = "complianceStatus"
	workerStatusField     = "status"
	workerAvailableField  = "availableForDispatch"
	workerPictureField    = "profilePicUrl"
	DispatchableField     = "dispatchable"
	workerNotDispatchBy   = "notDispatchableBecause"
	workerComplianceKeyAt = workerProfileField + "." + workerComplianceKey
)

var workerSearchTerms = []string{
	"look up a driver by name", "find a driver", "driver roster", "worker directory",
	"employee by name",
}

// workerCredentialDates are the profile dates a row always carries, null when
// nothing is on file: a missing credential date is the finding, and a key
// that disappears reads as nothing to worry about.
var workerCredentialDates = []string{"hazmatExpiry", "licenseExpiry", "medicalCardExpiry"}

func ReadSpecs() []readspec.List {
	return []readspec.List{{
		Name:         querytool.ToolWorkerList,
		EntityPlural: pluralWorkers,
		Summary:      workersSummary,
		SearchTerms:  workerSearchTerms,
		QueryNote:    workersQueryNote,
		Resource:     permission.ResourceWorker,
		Notes: map[string]string{
			querytool.ParamStatus: noteWorkerStatus,
			workerAssignable:      noteCanBeAssigned,
			"profileEndorsement":  endorsementNote + ".",
			"profileIsQualified":  noteWorkerQualified,
		},
		References: map[string]readspec.Reference{querytool.FieldFleetCodeID: {
			Resource: permission.ResourceFleetCode,
			What:     "The fleet code",
			Source:   querytool.ToolListFleetCodes,
		}},
		Rank:       rankWorkerRows,
		Shape:      shapeWorkerRow,
		Derived:    []string{DispatchableField, workerNotDispatchBy},
		Unoffered:  []string{workerBlockedField, workerAvailableField, workerPictureField},
		Conditions: []readspec.Condition{dispatchableCondition},
	}}
}

// rankWorkerRows puts the closest names first, as the repository list did.
func rankWorkerRows(query string, rows []map[string]any) []map[string]any {
	type scored struct {
		row   map[string]any
		score float64
	}
	ranked := make([]scored, 0, len(rows))
	for _, row := range rows {
		first, _ := row[workerFirstNameField].(string)
		last, _ := row[workerLastNameField].(string)
		ranked = append(ranked, scored{row: row, score: namesScore(query, first, last)})
	}
	slices.SortStableFunc(ranked, func(a, b scored) int { return cmp.Compare(b.score, a.score) })
	out := make([]map[string]any, 0, len(ranked))
	for _, entry := range ranked {
		out = append(out, entry.row)
	}

	return out
}

const activeWorker = string(domaintypes.StatusActive)

var dispatchableCondition = readspec.Condition{
	Name:        DispatchableField,
	Description: noteDispatchable,
	When: []domaintypes.FieldFilter{
		{Field: workerAssignable, Operator: dbtype.OpEqual, Value: true},
		{Field: workerAvailableField, Operator: dbtype.OpEqual, Value: true},
		{Field: workerBlockedField, Operator: dbtype.OpIsNull},
		{Field: workerStatusField, Operator: dbtype.OpEqual, Value: activeWorker},
		{
			Field:    workerComplianceKeyAt,
			Operator: dbtype.OpNotEqual,
			Value:    string(worker.ComplianceStatusNonCompliant),
		},
	},
	Unless: []domaintypes.FieldFilter{
		{Field: workerAssignable, Operator: dbtype.OpEqual, Value: false},
		{Field: workerAvailableField, Operator: dbtype.OpEqual, Value: false},
		{Field: workerBlockedField, Operator: dbtype.OpIsNotNull},
		{Field: workerStatusField, Operator: dbtype.OpNotEqual, Value: activeWorker},
		{
			Field:    workerComplianceKeyAt,
			Operator: dbtype.OpEqual,
			Value:    string(worker.ComplianceStatusNonCompliant),
		},
	},
}

func shapeWorkerRow(row map[string]any, withheld map[string]struct{}, _ filtercatalog.Clock) {
	dispatchStanding(row)
	profile, ok := row[workerProfileField].(map[string]any)
	if !ok {
		return
	}
	for _, date := range workerCredentialDates {
		if _, hidden := withheld[workerProfileField+"."+date]; hidden {
			continue
		}
		if _, present := profile[date]; !present {
			profile[date] = nil
		}
	}
}

func dispatchStanding(row map[string]any) {
	assignable, known := row[workerAssignable].(bool)
	if !known {
		return
	}
	blocked, _ := row[workerBlockedField].(string)
	blocked = strings.TrimSpace(blocked)
	reasons := make([]string, 0, 4)
	switch {
	case !assignable && blocked != "":
		reasons = append(reasons, "already off dispatch ("+blocked+")")
	case !assignable:
		reasons = append(reasons, "already off dispatch")
	case blocked != "":
		reasons = append(reasons, "blocked from assignment ("+blocked+")")
	}
	if available, set := row[workerAvailableField].(bool); set && !available {
		reasons = append(reasons, "marked unavailable for dispatch")
	}
	if status, _ := row[workerStatusField].(string); status != "" && status != activeWorker {
		reasons = append(reasons, "not active ("+status+")")
	}
	if profile, ok := row[workerProfileField].(map[string]any); ok {
		if compliance, _ := profile[workerComplianceKey].(string); compliance ==
			string(worker.ComplianceStatusNonCompliant) {
			reasons = append(reasons, "credentials are not compliant")
		}
	}
	row[DispatchableField] = len(reasons) == 0
	if len(reasons) > 0 {
		row[workerNotDispatchBy] = strings.Join(reasons, "; ")
	}
}
