package canned

import (
	"github.com/emoss08/trenova/internal/core/domain/report"
)

func driverRoster() *Entry {
	return &Entry{
		Key:     "driver-roster",
		Version: initialVersion,
		Name:    "Driver Roster",
		Description: "Every driver on the board with fleet, class, dispatch availability, " +
			"and hire date",
		Category:      categoryCompliance,
		Tags:          []string{tagDrivers, tagRoster, tagOperations},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityWorker,
			Columns: []report.ColumnSpec{
				dimCol("first_name", "First Name", firstNameField),
				dimCol("last_name", "Last Name", lastNameField),
				dimCol(statusFieldKey, "Status", statusFieldKey),
				dimCol("worker_type", "Worker Type", "type"),
				dimCol("driver_type", "Driver Type", "driverType"),
				dimCol("fleet_code", "Fleet", codeField, fleetCodeEdge),
				dimCol("cdl_class", "CDL Class", "cdlClass", profileEdge),
				dimCol("endorsements", "Endorsements", "endorsement", profileEdge),
				dateDim("hire_date", "Hire Date", "hireDate", profileEdge),
				styledDim("qualified", "Qualified", "isQualified", yesNo(), profileEdge),
				styledDim("can_be_assigned", "Assignable", "canBeAssigned", yesNo()),
				styledDim(
					"available_for_dispatch",
					"Available for Dispatch",
					"availableForDispatch",
					yesNo(),
				),
				styledDim(
					"assignment_blocked",
					"Assignment Blocked Reason",
					"assignmentBlocked",
					emptyDash(),
				),
			},
			Filters: andFilters(inParam(statusFieldKey, statusParam)),
			Sort:    []report.SortSpec{asc("last_name")},
			Parameters: []report.ParameterDef{
				enumParam(statusParam, statusesLabel,
					[]any{statusActive},
					[]string{statusActive, "Inactive"},
				),
			},
		},
	}
}

func driverQualificationStatus() *Entry {
	return &Entry{
		Key:     "driver-qualification-status",
		Version: initialVersion,
		Name:    "Driver Qualification File Status",
		Description: "DQ file at a glance — compliance standing, credential dates, MVR " +
			"and physical due dates, and disqualification reasons",
		Category:      categoryCompliance,
		Tags:          []string{tagDrivers, tagCompliance, "dq"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityWorker,
			Columns: []report.ColumnSpec{
				dimCol("first_name", "First Name", firstNameField),
				dimCol("last_name", "Last Name", lastNameField),
				dimCol("driver_type", "Driver Type", "driverType"),
				dimCol("fleet_code", "Fleet", codeField, fleetCodeEdge),
				dimCol(
					"compliance_status",
					"Compliance Status",
					"complianceStatus",
					profileEdge,
				),
				styledDim("qualified", "Qualified", "isQualified", yesNo(), profileEdge),
				styledDim(
					"disqualification_reason",
					"Disqualification Reason",
					"disqualificationReason",
					emptyDash(),
					profileEdge,
				),
				dimCol("cdl_class", "CDL Class", "cdlClass", profileEdge),
				dimCol("endorsements", "Endorsements", "endorsement", profileEdge),
				dimCol("cdl_restrictions", "CDL Restrictions", "cdlRestrictions", profileEdge),
				dateDim("license_expiry", "License Expiry", "licenseExpiry", profileEdge),
				dateDim("medical_card_expiry", "Medical Card Expiry",
					"medicalCardExpiry", profileEdge),
				dateDim("mvr_due", "MVR Due", "mvrDueDate", profileEdge),
				dateDim("physical_due", "Physical Due", "physicalDueDate", profileEdge),
				dateDim("last_compliance_check", "Last Compliance Check",
					"lastComplianceCheck", profileEdge),
				dateDim("hire_date", "Hire Date", "hireDate", profileEdge),
			},
			Filters: andFilters(
				statusEquals(statusActive),
				inParam("complianceStatus", "complianceStatuses", profileEdge),
			),
			Sort: []report.SortSpec{asc("last_name")},
			Parameters: []report.ParameterDef{
				enumParam("complianceStatuses", "Compliance Statuses",
					[]any{"NonCompliant", "Pending"},
					[]string{"Compliant", "NonCompliant", "Pending"},
				),
			},
		},
	}
}

func expiringWorkerCredentials() *Entry {
	return &Entry{
		Key:     "expiring-worker-credentials",
		Version: initialVersion,
		Name:    "Expiring Driver Credentials",
		Description: "Any driver credential — CDL, medical card, hazmat, or TWIC — " +
			"expiring inside the horizon, with fleet and contact routing",
		Category:      categoryCompliance,
		Tags:          []string{tagDrivers, tagCompliance, "credentials"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityWorker,
			Columns: []report.ColumnSpec{
				dimCol("first_name", "First Name", firstNameField),
				dimCol("last_name", "Last Name", lastNameField),
				dimCol(statusFieldKey, "Status", statusFieldKey),
				dimCol("driver_type", "Driver Type", "driverType"),
				dimCol("fleet_code", "Fleet", codeField, fleetCodeEdge),
				dimCol("cdl_class", "CDL Class", "cdlClass", profileEdge),
				dimCol(
					"compliance_status",
					"Compliance Status",
					"complianceStatus",
					profileEdge,
				),
				dateDim("license_expiry", "License Expiry", "licenseExpiry", profileEdge),
				dateDim("medical_card_expiry", "Medical Card Expiry",
					"medicalCardExpiry", profileEdge),
				dateDim("hazmat_expiry", "Hazmat Expiry", "hazmatExpiry", profileEdge),
				dateDim("twic_expiry", "TWIC Expiry", "twicExpiry", profileEdge),
				dateDim("mvr_due", "MVR Due", "mvrDueDate", profileEdge),
			},
			Filters: withGroups(
				andFilters(statusEquals(statusActive)),
				orGroup(
					horizonFilter("licenseExpiry", profileEdge),
					horizonFilter("medicalCardExpiry", profileEdge),
					horizonFilter("hazmatExpiry", profileEdge),
					horizonFilter("twicExpiry", profileEdge),
				),
			),
			Sort:       []report.SortSpec{asc("license_expiry")},
			Parameters: []report.ParameterDef{horizonParam(30)},
		},
	}
}
