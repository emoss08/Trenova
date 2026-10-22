package agentquerytoolservice

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
)

const (
	workersSummary = "Find workers (drivers) by name, closest first, or the roster by employment " +
		"or qualification. Credentials due: worker_list_credentials. " +
		"\"none on file\" was never recorded: worse than expiring soon."
	workersQueryNote = "A worker's name, first, last or both; each word matches the start of a " +
		"name and the closest names come first. Omit it to list the roster or to filter " +
		"on a field."
	noteWorkerStatus     = "Not availability; see canBeAssigned and worker_list_time_off."
	noteCanBeAssigned    = "false means dispatch is blocked."
	noteWorkerQualified  = "false means a credential lapsed or is missing."
	pluralWorkers        = "workers"
	workerProfileField   = "profile"
	workerFirstNameField = "firstName"
	workerLastNameField  = "lastName"
	workerAssignable     = "canBeAssigned"
	workerBlockedField   = "assignmentBlocked"
	workerComplianceKey  = "complianceStatus"
)

var workerSearchTerms = []string{
	"look up a driver by name", "find a driver", "driver roster", "worker directory",
	"employee by name",
}

// workerCredentialDates are the profile dates a row always carries, null when
// nothing is on file: a missing credential date is the finding, and a key
// that disappears reads as nothing to worry about.
var workerCredentialDates = []string{"hazmatExpiry", "licenseExpiry", "medicalCardExpiry"}

func workerReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{{
		name:         toolWorkerList,
		entityPlural: pluralWorkers,
		summary:      workersSummary,
		searchTerms:  workerSearchTerms,
		queryNote:    workersQueryNote,
		resource:     permission.ResourceWorker,
		notes: map[string]string{
			paramStatus:          noteWorkerStatus,
			workerAssignable:     noteCanBeAssigned,
			"profileEndorsement": endorsementNote + ".",
			"profileIsQualified": noteWorkerQualified,
		},
		references: map[string]idReference{fieldFleetCodeID: {
			resource: permission.ResourceFleetCode,
			what:     "The fleet code",
			source:   toolListFleetCodes,
		}},
		rank:  rankWorkerRows,
		shape: shapeWorkerRow,
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

// shapeWorkerRow applies to a GraphQL row what the repository row applied:
// a NonCompliant worker cannot be assigned, and the credential dates are
// present, null when nothing is on file, unless the reader may not see them.
func shapeWorkerRow(row map[string]any, withheld map[string]struct{}, _ clock) {
	profile, ok := row[workerProfileField].(map[string]any)
	if !ok {
		return
	}
	if compliance, known := profile[workerComplianceKey].(string); known {
		assignable, _ := row[workerAssignable].(bool)
		blocked, _ := row[workerBlockedField].(string)
		assignable, blocked = dispatchable(assignable, blocked, worker.ComplianceStatus(compliance))
		row[workerAssignable] = assignable
		if blocked != "" {
			row[workerBlockedField] = blocked
		}
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
