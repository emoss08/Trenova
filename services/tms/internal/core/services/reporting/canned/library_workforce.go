package canned

import (
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
)

const (
	entityWorkerCredential      = "worker_credential"
	entityWorkerEmploymentEvent = "worker_employment_event"
	entityWorkerPTOBalance      = "worker_pto_balance"

	credentialTypeEdge = "credentialType"
	workerEdge         = "worker"
	positionEdge       = "position"

	tagWorkforce  = "workforce"
	tagHeadcount  = "headcount"
	tagTurnover   = "turnover"
	tagTimeOff    = "time-off"
	tagCredential = "credentials"

	workerCountColumnID = "worker_count"
	eventCountColumnID  = "event_count"
	positionColumnID    = "position"
	departmentColumnID  = "department"
	balanceDaysColumnID = "balance_days"
)

// days renders a time-off figure. PTO is carried in days to two decimals
// because a half day is a real thing a driver takes.
func days() *reportfmt.Spec {
	decimals := 2
	grouping := true
	return &reportfmt.Spec{
		Style:    reportfmt.StyleNumber,
		Decimals: &decimals,
		Grouping: &grouping,
		Suffix:   " d",
	}
}

func sumDays(id, label, field string, path ...string) report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:      id,
		label:   label,
		agg:     reportcatalog.AggSum,
		field:   field,
		path:    path,
		display: days(),
	})
}

// earliestHire is the oldest hire date in a group. The date lives on the
// profile rather than the worker, so it needs a path the shared timestamp
// helper does not take.
func earliestHire() report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:      "earliest_hire",
		label:   "Earliest Hire",
		agg:     reportcatalog.AggMin,
		field:   "hireDate",
		path:    []string{profileEdge},
		display: dateOnly(),
	})
}

func kindIn(values []any) report.FieldFilter {
	return report.FieldFilter{
		Ref:      report.FieldRef{Field: "kind"},
		Operator: dbtype.OpIn,
		Value:    values,
	}
}

// headcountSnapshot is the roster as it stands today, counted the three ways
// the office asks for it. It is a snapshot rather than a history: the counts
// come off the current roster, and the record of how it got here is the
// employment timeline, which the turnover report reads instead.
func headcountSnapshot() *Entry {
	return &Entry{
		Key:     "headcount-snapshot",
		Version: initialVersion,
		Name:    "Headcount Snapshot",
		Description: "The active roster counted by terminal, position and department, " +
			"with the driving half broken out",
		Category:      categoryCompliance,
		Tags:          []string{tagWorkforce, tagHeadcount, tagRoster},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityWorker,
			Columns: []report.ColumnSpec{
				dimCol(fleetCodeColumnID, fleetLabel, codeField, fleetCodeEdge),
				dimCol(departmentColumnID, "Department", "department", positionEdge),
				dimCol(positionColumnID, "Position", "title", positionEdge),
				styledDim("driving", "Driving Position", "isDrivingPosition", yesNo(), positionEdge),
				styledDim("exempt", "Exempt from Overtime", "flsaExempt", yesNo(), positionEdge),
				countMeasure(workerCountColumnID, "Workers"),
				earliestHire(),
			},
			Filters: andFilters(inParam(statusFieldKey, statusParam)),
			Sort:    []report.SortSpec{desc(workerCountColumnID)},
			Parameters: []report.ParameterDef{
				enumParam(statusParam, statusesLabel,
					[]any{statusActive},
					[]string{statusActive, "Inactive"},
				),
			},
		},
	}
}

// workforceTenure is the roster by how long people have been here. Hire date
// rather than a computed tenure: the report engine groups on stored columns,
// and the month somebody started is the thing a turnover conversation is
// actually about.
func workforceTenure() *Entry {
	return &Entry{
		Key:     "workforce-tenure",
		Version: initialVersion,
		Name:    "Workforce Tenure",
		Description: "Every active worker with their hire date, terminal, position and " +
			"manager, for reading tenure across the roster",
		Category:      categoryCompliance,
		Tags:          []string{tagWorkforce, tagRoster, "tenure"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityWorker,
			Columns: []report.ColumnSpec{
				dimCol("first_name", "First Name", firstNameField),
				dimCol("last_name", "Last Name", lastNameField),
				dimCol(statusFieldKey, statusLabel, statusFieldKey),
				dimCol("worker_type", "Worker Type", "type"),
				dimCol("driver_type", "Driver Type", "driverType"),
				dimCol(fleetCodeColumnID, fleetLabel, codeField, fleetCodeEdge),
				dimCol(positionColumnID, "Position", "title", positionEdge),
				dimCol(departmentColumnID, "Department", "department", positionEdge),
				dateDim("hire_date", "Hire Date", "hireDate", profileEdge),
				dateDim("termination_date", "Termination Date", "terminationDate", profileEdge),
			},
			Filters: andFilters(inParam(statusFieldKey, statusParam)),
			Sort:    []report.SortSpec{asc("hire_date")},
			Parameters: []report.ParameterDef{
				enumParam(statusParam, statusesLabel,
					[]any{statusActive},
					[]string{statusActive, "Inactive"},
				),
			},
		},
	}
}

// workforceTurnover counts the comings and goings by month. It reads the
// employment timeline rather than the roster, because the roster only knows
// where somebody is now — the timeline is the only thing that remembers they
// left in March and came back in June.
func workforceTurnover() *Entry {
	return &Entry{
		Key:     "workforce-turnover",
		Version: initialVersion,
		Name:    "Workforce Turnover",
		Description: "Hires, terminations and rehires by month, read from the employment " +
			"timeline rather than the current roster",
		Category:      categoryCompliance,
		Tags:          []string{tagWorkforce, tagTurnover, tagHeadcount},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityWorkerEmploymentEvent,
			Columns: []report.ColumnSpec{
				bucketDim(monthColumnID, "Month", "effectiveAt", report.DateBucketMonth),
				dimCol("event_kind", "Event", "kind"),
				countMeasure(eventCountColumnID, "Events"),
				distinctMeasure("worker_count", "Workers", "workerId"),
			},
			Filters: andFilters(
				windowFilter("effectiveAt"),
				kindIn([]any{"Hired", "Rehired", "Terminated"}),
			),
			Sort:       []report.SortSpec{desc(monthColumnID), asc("event_kind")},
			Parameters: []report.ParameterDef{windowParam(365)},
		},
	}
}

// ptoLiability is what the organisation owes in time off. Accrued balances
// rather than requests: a request that has not been taken is not a liability
// twice, and a balance nobody has requested against still is one.
func ptoLiability() *Entry {
	return &Entry{
		Key:     "pto-liability",
		Version: initialVersion,
		Name:    "Time Off Liability",
		Description: "Accrued time off the organisation is carrying, by type and terminal, " +
			"with what has been accrued and used this year",
		Category:      categoryCompliance,
		Tags:          []string{tagWorkforce, tagTimeOff, "liability"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityWorkerPTOBalance,
			Columns: []report.ColumnSpec{
				dimCol("pto_type", "Time Off Type", "ptoType"),
				dimCol(fleetCodeColumnID, fleetLabel, codeField, workerEdge, fleetCodeEdge),
				distinctMeasure(workerCountColumnID, "Workers", "workerId"),
				sumDays(balanceDaysColumnID, "Accrued Balance", "balanceDays"),
				sumDays("accrued_ytd", "Accrued This Year", "accruedYtdDays"),
				sumDays("used_ytd", "Used This Year", "usedYtdDays"),
				sumDays("carried", "Carried In", "carriedDays"),
				perUnit(
					"avg_balance", "Average Balance per Worker",
					balanceDaysColumnID, workerCountColumnID, days(), 2,
				),
			},
			Sort: []report.SortSpec{desc(balanceDaysColumnID)},
		},
	}
}

// credentialExpiryForecast reads the credential registry rather than the
// profile columns the older report uses. The registry is where a carrier's own
// credential types live, so this is the only one of the two that can see a
// certificate nobody thought to put a column on.
func credentialExpiryForecast() *Entry {
	return &Entry{
		Key:     "credential-expiry-forecast",
		Version: initialVersion,
		Name:    "Credential Expiry Forecast",
		Description: "Every tracked credential expiring inside the horizon, from the " +
			"credential registry rather than the fixed profile columns",
		Category:      categoryCompliance,
		Tags:          []string{tagWorkforce, tagCredential, tagCompliance},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityWorkerCredential,
			Columns: []report.ColumnSpec{
				dimCol("last_name", "Last Name", lastNameField, workerEdge),
				dimCol("first_name", "First Name", firstNameField, workerEdge),
				dimCol(fleetCodeColumnID, fleetLabel, codeField, workerEdge, fleetCodeEdge),
				dimCol("credential", "Credential", nameField, credentialTypeEdge),
				dimCol("credential_code", "Code", codeField, credentialTypeEdge),
				styledDim("required", "Required", "isRequired", yesNo(), credentialTypeEdge),
				dimCol(statusFieldKey, statusLabel, statusFieldKey),
				styledDim("number", "Number", "number", emptyDash()),
				dimCol("issuing_authority", "Issuing Authority", "issuingAuthority"),
				dateDim("expires_at", "Expires", "expiresAt"),
				dateDim("verified_at", "Verified", "verifiedAt"),
			},
			Filters: andFilters(
				statusEquals(statusActive),
				horizonFilter("expiresAt"),
			),
			Sort:       []report.SortSpec{asc("expires_at")},
			Parameters: []report.ParameterDef{horizonParam(60)},
		},
	}
}

// dqfCredentialDetail is the credential half of a driver qualification file,
// one row per credential rather than one per driver.
//
// It is the companion to the roster-level report, not a replacement: which
// required credentials a driver is *missing* is an absence, and an absence is
// not a row anything can select. The roll-up that answers it — compliance
// status and the disqualification reason — is on the worker profile and is in
// the roster report.
func dqfCredentialDetail() *Entry {
	return &Entry{
		Key:     "dqf-credential-detail",
		Version: initialVersion,
		Name:    "DQ File Credential Detail",
		Description: "Every credential on file for every driver, with type, requirement, " +
			"issue and expiry dates and who verified it",
		Category:      categoryCompliance,
		Tags:          []string{tagDrivers, tagCompliance, "dq", tagCredential},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityWorkerCredential,
			Columns: []report.ColumnSpec{
				dimCol("last_name", "Last Name", lastNameField, workerEdge),
				dimCol("first_name", "First Name", firstNameField, workerEdge),
				dimCol(fleetCodeColumnID, fleetLabel, codeField, workerEdge, fleetCodeEdge),
				dimCol("credential", "Credential", nameField, credentialTypeEdge),
				dimCol("category", "Category", "category", credentialTypeEdge),
				styledDim("required", "Required", "isRequired", yesNo(), credentialTypeEdge),
				dimCol(statusFieldKey, statusLabel, statusFieldKey),
				styledDim("number", "Number", "number", emptyDash()),
				dateDim("issued_at", "Issued", "issuedAt"),
				dateDim("expires_at", "Expires", "expiresAt"),
				dateDim("verified_at", "Verified", "verifiedAt"),
			},
			Filters: andFilters(inParam(statusFieldKey, statusParam)),
			Sort:    []report.SortSpec{asc("last_name"), asc("credential")},
			Parameters: []report.ParameterDef{
				enumParam(statusParam, statusesLabel,
					[]any{statusActive},
					[]string{statusActive, "Inactive"},
				),
			},
		},
	}
}
