package workerperms

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

const (
	fieldCancelledByID   = "cancelledById"
	fieldEndDate         = "endDate"
	fieldRejectionReason = "rejectionReason"
	fieldStartDate       = "startDate"
)

func Resources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		workerResource(),
		workerPTOResource(),
		workerCredentialResource(),
		workerEmploymentEventResource(),
		workerChecklistResource(),
		workerChecklistTemplateResource(),
		orgHolidayResource(),
		trainingCourseResource(),
		workerTrainingResource(),
		workerDOTTestResource(),
		workerLeaveResource(),
		benefitPlanResource(),
		shiftTemplateResource(),
		workerScheduleResource(),
		shiftSwapResource(),
		timesheetResource(),
		workerPolicyResource(),
		profileChangeRequestResource(),
		jobPositionResource(),
		approvalDelegationResource(),
		workerInjuryResource(),
		dotRandomPoolResource(),
		workerSafetyEventResource(),
		workerDisciplinaryActionResource(),
		workerRecognitionResource(),
		performanceReviewTemplateResource(),
		performanceReviewResource(),
		workerCredentialTypeResource(),
		ptoPolicyResource(),
	}
}

func workerResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceWorker.String(),
		DisplayName: "Worker",
		Description: "Driver and worker management",
		Category:    "Human resources",
		ReferenceView: []string{
			"firstName",
			"lastName",
			"wholeName",
			"profilePicUrl",
			"status",
			"type",
			"fleetCodeId",
			"fleetCode",
			"canBeAssigned",
			"assignmentBlocked",
		},
		Operations: []permissionspec.OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View workers"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Add new workers",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify worker information",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export worker data",
			},
			{
				Operation:   permission.OpImport,
				DisplayName: "Import",
				Description: "Import worker data",
			},
			{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Archive workers",
			},
			{
				Operation:   permission.OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived workers",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: workerFieldSensitivities(),
		Scopes: []permissionspec.ScopeSupport{
			{
				Scope:      permission.DataScopeTeam,
				EnforcedBy: permissionspec.ScopeEnforcedByDatabase,
			},
		},
	}
}

func workerFieldSensitivities() map[string]permission.FieldSensitivity {
	return map[string]permission.FieldSensitivity{
		"id":                     permission.SensitivityInternal,
		"businessUnitId":         permission.SensitivityInternal,
		"organizationId":         permission.SensitivityInternal,
		"workerId":               permission.SensitivityInternal,
		"status":                 permission.SensitivityInternal,
		"type":                   permission.SensitivityInternal,
		"driverType":             permission.SensitivityInternal,
		"firstName":              permission.SensitivityInternal,
		"lastName":               permission.SensitivityInternal,
		"wholeName":              permission.SensitivityInternal,
		"positionId":             permission.SensitivityInternal,
		"userId":                 permission.SensitivityInternal,
		"externalId":             permission.SensitivityInternal,
		"profilePicUrl":          permission.SensitivityInternal,
		"fleetCodeId":            permission.SensitivityInternal,
		"profile":                permission.SensitivityInternal,
		"canBeAssigned":          permission.SensitivityInternal,
		"assignmentBlocked":      permission.SensitivityInternal,
		"availableForDispatch":   permission.SensitivityInternal,
		"eldExempt":              permission.SensitivityInternal,
		"shortHaulExempt":        permission.SensitivityInternal,
		"cdlClass":               permission.SensitivityInternal,
		"cdlRestrictions":        permission.SensitivityInternal,
		"endorsement":            permission.SensitivityInternal,
		"hazmatExpiry":           permission.SensitivityInternal,
		"licenseExpiry":          permission.SensitivityInternal,
		"medicalCardExpiry":      permission.SensitivityInternal,
		"twicExpiry":             permission.SensitivityInternal,
		"physicalDueDate":        permission.SensitivityInternal,
		"mvrDueDate":             permission.SensitivityInternal,
		"complianceStatus":       permission.SensitivityInternal,
		"isQualified":            permission.SensitivityInternal,
		"lastComplianceCheck":    permission.SensitivityInternal,
		"lastMvrCheck":           permission.SensitivityInternal,
		"version":                permission.SensitivityInternal,
		"createdAt":              permission.SensitivityInternal,
		"updatedAt":              permission.SensitivityInternal,
		"gender":                 permission.SensitivityRestricted,
		"addressLine1":           permission.SensitivityRestricted,
		"addressLine2":           permission.SensitivityRestricted,
		"email":                  permission.SensitivityRestricted,
		"phoneNumber":            permission.SensitivityRestricted,
		"city":                   permission.SensitivityRestricted,
		"postalCode":             permission.SensitivityRestricted,
		"stateId":                permission.SensitivityRestricted,
		"managerId":              permission.SensitivityRestricted,
		"licenseStateId":         permission.SensitivityRestricted,
		"emergencyContactName":   permission.SensitivityRestricted,
		"emergencyContactPhone":  permission.SensitivityRestricted,
		"hireDate":               permission.SensitivityRestricted,
		"terminationDate":        permission.SensitivityRestricted,
		"disqualificationReason": permission.SensitivityRestricted,
		"medicalExaminerName":    permission.SensitivityRestricted,
		"dob":                    permission.SensitivityConfidential,
		"licenseNumber":          permission.SensitivityConfidential,
		"twicCardNumber":         permission.SensitivityConfidential,
		"medicalExaminerNpi":     permission.SensitivityConfidential,
		"lastDrugTest":           permission.SensitivityConfidential,
		"drugAlcoholStatus":      permission.SensitivityConfidential,
		"returnToDutyStatus":     permission.SensitivityConfidential,
	}
}

func workerPTOResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceWorkerPTO.String(),
		DisplayName: "Worker PTO",
		Description: "Worker paid time off requests and approvals",
		Category:    "Human resources",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View worker PTO requests",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create worker PTO requests",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify worker PTO requests",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export worker PTO data",
			},
			{
				Operation:   permission.OpApprove,
				DisplayName: "Approve",
				Description: "Approve worker PTO requests",
			},
			{
				Operation:   permission.OpReject,
				DisplayName: "Reject",
				Description: "Reject worker PTO requests",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: "Cancel",
				Description: "Cancel worker PTO requests",
			},
			{
				Operation:   permission.OpManage,
				DisplayName: "Manage Balances",
				Description: "Adjust PTO balances and run accruals",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":                 permission.SensitivityInternal,
			"businessUnitId":     permission.SensitivityInternal,
			"organizationId":     permission.SensitivityInternal,
			"workerId":           permission.SensitivityInternal,
			"approverId":         permission.SensitivityInternal,
			"rejectorId":         permission.SensitivityInternal,
			fieldCancelledByID:   permission.SensitivityInternal,
			"status":             permission.SensitivityInternal,
			"type":               permission.SensitivityInternal,
			fieldStartDate:       permission.SensitivityInternal,
			fieldEndDate:         permission.SensitivityInternal,
			"days":               permission.SensitivityInternal,
			"autoApproved":       permission.SensitivityInternal,
			"version":            permission.SensitivityInternal,
			"createdAt":          permission.SensitivityInternal,
			"updatedAt":          permission.SensitivityInternal,
			"reason":             permission.SensitivityRestricted,
			fieldRejectionReason: permission.SensitivityRestricted,
			"cancellationReason": permission.SensitivityRestricted,
			"balanceAfterDays":   permission.SensitivityRestricted,
		},
	}
}

func workerCredentialResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceWorkerCredential.String(),
		DisplayName:    "Worker Credential",
		Description:    "Licences, medical cards, endorsements and certificates held by workers",
		Category:       "Human resources",
		ReferenceView:  []string{"workerId", "credentialTypeId", "status", "expiresAt"},
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View worker credentials",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Add or renew worker credentials",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify worker credentials",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export worker credential data",
			},
			{
				Operation:   permission.OpApprove,
				DisplayName: "Verify",
				Description: "Mark a credential as verified against its document",
			},
			{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Archive worker credentials",
			},
		},
		// What a credential is, whose, when it was issued and expires, by whom,
		// and whether it is verified and has a scan on file is what dispatch
		// and compliance work from, so it is Internal. The notes and why it
		// was archived stay Restricted, and the number Confidential.
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":               permission.SensitivityInternal,
			"businessUnitId":   permission.SensitivityInternal,
			"organizationId":   permission.SensitivityInternal,
			"workerId":         permission.SensitivityInternal,
			"worker":           permission.SensitivityInternal,
			"credentialTypeId": permission.SensitivityInternal,
			"credentialType":   permission.SensitivityInternal,
			"status":           permission.SensitivityInternal,
			"issuingAuthority": permission.SensitivityInternal,
			"issuedAt":         permission.SensitivityInternal,
			"expiresAt":        permission.SensitivityInternal,
			"health":           permission.SensitivityInternal,
			"daysUntilExpiry":  permission.SensitivityInternal,
			"documentId":       permission.SensitivityInternal,
			"verifiedById":     permission.SensitivityInternal,
			"verifiedAt":       permission.SensitivityInternal,
			"archivedById":     permission.SensitivityInternal,
			"archivedAt":       permission.SensitivityInternal,
			"version":          permission.SensitivityInternal,
			"createdAt":        permission.SensitivityInternal,
			"updatedAt":        permission.SensitivityInternal,
			"number":           permission.SensitivityConfidential,
		},
	}
}

func workerEmploymentEventResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceWorkerEmploymentEvent.String(),
		DisplayName: "Employment Event",
		Description: "Hires, terminations, leave, suspensions, transfers and other employment history",
		Category:    "Human resources",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View a worker's employment timeline",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Record",
				Description: "Record employment events, including terminations and rehires",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Amend",
				Description: "Correct recorded employment events",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export employment history",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func workerChecklistResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceWorkerChecklist.String(),
		DisplayName:    "Worker Checklist",
		Description:    "Onboarding and offboarding checklists run for a worker",
		Category:       "Human resources",
		ReferenceView:  []string{"name", "kind", "status", "dueAt"},
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View a worker's checklists",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Start",
				Description: "Start a checklist for a worker",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Complete",
				Description: "Complete, skip or reopen checklist items",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: "Cancel",
				Description: "Cancel an open checklist",
			},
		},
	}
}

func workerChecklistTemplateResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceWorkerChecklistTemplate.String(),
		DisplayName: "Checklist Template",
		Description: "Reusable onboarding and offboarding checklists",
		Category:    "Human resources",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View checklist templates",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create checklist templates",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify checklist templates",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export checklist templates",
			},
			{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Deactivate checklist templates",
			},
			{
				Operation:   permission.OpRestore,
				DisplayName: "Restore",
				Description: "Reactivate checklist templates",
			},
		},
	}
}

func orgHolidayResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceOrgHoliday.String(),
		DisplayName: "Holiday Calendar",
		Description: "Organisation holidays and PTO blackout dates",
		Category:    "Human resources",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View the holiday calendar",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Add holidays and blackout dates",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify holidays and blackout dates",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Remove holidays and blackout dates",
			},
		},
	}
}

func trainingCourseResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceTrainingCourse.String(),
		DisplayName: "Training Course",
		Description: "Catalog of training courses and which are required per driver type",
		Category:    "Human resources",
		ReferenceView: []string{
			"code",
			"name",
			"category",
			"delivery",
			"isRequired",
			"status",
			"description",
			"contentUrl",
			"durationMinutes",
			"passingScore",
			"validityMonths",
			"renewalWindowDays",
			"dueDaysAfterAssignment",
			"requiresAcknowledgement",
			"requiredForDriverTypes",
			"sortOrder",
		},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View training courses",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create training courses",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify training courses",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export training course data",
			},
			{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Deactivate training courses",
			},
			{
				Operation:   permission.OpRestore,
				DisplayName: "Restore",
				Description: "Reactivate training courses",
			},
		},
	}
}

func workerTrainingResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceWorkerTraining.String(),
		DisplayName:    "Worker Training",
		Description:    "Courses assigned to workers, their results and certifications",
		Category:       "Human resources",
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View worker training",
			},
			{
				Operation:   permission.OpAssign,
				DisplayName: "Assign",
				Description: "Assign courses to workers",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Record result",
				Description: "Record completions, scores and certificates",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: "Waive or cancel",
				Description: "Waive a required course or withdraw an assignment",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export worker training data",
			},
		},
	}
}

func workerDOTTestResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceWorkerDOTTest.String(),
		DisplayName: "Drug & Alcohol Testing",
		Description: "Controlled-substances and alcohol tests, violations and Clearinghouse queries on a worker's record",
		Category:    "Human resources",
		// 49 CFR 40.321 releases test results only to those whose job is DOT
		// compliance, so the kind has no parent: a worker grant never reaches
		// it, and only the safety bundle grants it. That a test was ordered,
		// why and when, and where it stands is what scheduling a collection
		// needs, and is Internal. What it found (the result, the substances,
		// the alcohol reading) and who collected, tested and verified the
		// specimen are Restricted.
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":             permission.SensitivityInternal,
			"businessUnitId": permission.SensitivityInternal,
			"organizationId": permission.SensitivityInternal,
			"workerId":       permission.SensitivityInternal,
			"testType":       permission.SensitivityInternal,
			"isDot":          permission.SensitivityInternal,
			"reason":         permission.SensitivityInternal,
			"status":         permission.SensitivityInternal,
			"scheduledAt":    permission.SensitivityInternal,
			"collectedAt":    permission.SensitivityInternal,
			"resultAt":       permission.SensitivityInternal,
			"safetyEventId":  permission.SensitivityInternal,
			"drawEntryId":    permission.SensitivityInternal,
			"orderedById":    permission.SensitivityInternal,
			"recordedById":   permission.SensitivityInternal,
			"version":        permission.SensitivityInternal,
			"createdAt":      permission.SensitivityInternal,
			"updatedAt":      permission.SensitivityInternal,
		},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View test records, violations and Clearinghouse queries",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Record",
				Description: "Schedule collections and record results and queries",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Correct a test, query or violation record",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: "Cancel",
				Description: "Void a collection that did not happen",
			},
			{
				Operation:   permission.OpManage,
				DisplayName: "Manage violations",
				Description: "Record violations and drive the return-to-duty process",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export testing data for an audit",
			},
		},
	}
}

func workerLeaveResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceWorkerLeave.String(),
		DisplayName: "Leave & FMLA",
		Description: "Leave cases, the FMLA entitlement drawn against them, and medical certification",
		Category:    "Human resources",
		// A leave case carries the reason somebody is off work and the medical
		// certification behind it (29 CFR 825.500(g)), so it has no parent:
		// reading a worker never reaches their leave, and only a worker_leave
		// grant does. A leave case is its own record wherever it is carried,
		// and nothing links to one, so it has no reference view. That a worker
		// is on leave, of what kind, when, and where the certification stands
		// is Internal, as are the days taken; the reason, the certificate and
		// the notes stay Restricted.
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":                       permission.SensitivityInternal,
			"businessUnitId":           permission.SensitivityInternal,
			"organizationId":           permission.SensitivityInternal,
			"workerId":                 permission.SensitivityInternal,
			"leaveType":                permission.SensitivityInternal,
			"status":                   permission.SensitivityInternal,
			"frequency":                permission.SensitivityInternal,
			"fmlaDesignated":           permission.SensitivityInternal,
			"requestedAt":              permission.SensitivityInternal,
			"startsAt":                 permission.SensitivityInternal,
			"endsAt":                   permission.SensitivityInternal,
			"decidedAt":                permission.SensitivityInternal,
			"closedAt":                 permission.SensitivityInternal,
			"certificationStatus":      permission.SensitivityInternal,
			"certificationRequestedAt": permission.SensitivityInternal,
			"certificationDueAt":       permission.SensitivityInternal,
			"certificationReceivedAt":  permission.SensitivityInternal,
			"recertificationDueAt":     permission.SensitivityInternal,
			"certificationLate":        permission.SensitivityInternal,
			"leaveCaseId":              permission.SensitivityInternal,
			"usedOn":                   permission.SensitivityInternal,
			"hours":                    permission.SensitivityInternal,
			"countsAgainstEntitlement": permission.SensitivityInternal,
			"ptoId":                    permission.SensitivityInternal,
			"version":                  permission.SensitivityInternal,
			"createdAt":                permission.SensitivityInternal,
			"updatedAt":                permission.SensitivityInternal,
		},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View leave cases and balances",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Record",
				Description: "Open a leave case and record days",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Correct a case, its certification or its days",
			},
			{
				Operation:   permission.OpApprove,
				DisplayName: "Decide",
				Description: "Approve, deny, designate and close a leave case",
			},
			{
				Operation:   permission.OpManage,
				DisplayName: "Configure",
				Description: "Set the measurement method and entitlement",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export leave data",
			},
		},
		Scopes: []permissionspec.ScopeSupport{
			{
				Scope:      permission.DataScopeTeam,
				EnforcedBy: permissionspec.ScopeEnforcedByApplication,
			},
		},
	}
}

func benefitPlanResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourceBenefitPlan.String(),
		DisplayName:   "Benefits",
		Description:   "Benefit plans and who is enrolled on them",
		Category:      "Human resources",
		ReferenceView: []string{"code", "name", "planType"},
		// An enrollment says what cover somebody has and what they pay for it.
		// It is granted separately for the same reason the injury log is.
		DefaultSensitivity: permission.SensitivityRestricted,
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View plans and enrollments",
			},
			{Operation: permission.OpCreate, DisplayName: "Create", Description: "Add a plan"},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Edit or archive a plan",
			},
			{
				Operation:   permission.OpAssign,
				DisplayName: "Enroll",
				Description: "Put a worker on a plan, record a waiver, or end cover",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export benefits data",
			},
		},
	}
}

func shiftTemplateResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceShiftTemplate.String(),
		DisplayName:    "Shift",
		Description:    "The repeating working patterns a roster is built from",
		Category:       "Human resources",
		ReferenceView:  []string{"code", "name", "color", "startMinute", "durationMinutes"},
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View shifts"},
			{Operation: permission.OpCreate, DisplayName: "Create", Description: "Add a shift"},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Edit a shift or retire it",
			},
		},
	}
}

func workerScheduleResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceWorkerSchedule.String(),
		DisplayName:    "Schedule",
		Description:    "The rota, who is on which shift, and what they would rather work",
		Category:       "Human resources",
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View the rota"},
			{
				Operation:   permission.OpAssign,
				DisplayName: "Assign",
				Description: "Put a worker on a shift or take them off one",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Set a worker's stated availability",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export the rota",
			},
		},
		Scopes: []permissionspec.ScopeSupport{
			{
				Scope:      permission.DataScopeTeam,
				EnforcedBy: permissionspec.ScopeEnforcedByApplication,
			},
		},
	}
}

func shiftSwapResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceShiftSwap.String(),
		DisplayName:    "Shift Swap",
		Description:    "Days drivers arrange to trade between themselves",
		Category:       "Human resources",
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View swap requests",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Propose a swap",
			},
			{
				Operation:   permission.OpApprove,
				DisplayName: "Approve",
				Description: "Approve a swap the two drivers have agreed",
			},
			{
				Operation:   permission.OpReject,
				DisplayName: "Reject",
				Description: "Reject a swap",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: "Withdraw",
				Description: "Withdraw a swap on a driver's behalf",
			},
		},
	}
}

func timesheetResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceTimesheet.String(),
		DisplayName: "Timesheet",
		Description: "Hours worked by staff paid by the clock, and the sign-off on them",
		Category:    "Human resources",
		// A timesheet is a wage record. It is granted separately for the same
		// reason pay is.
		DefaultSensitivity: permission.SensitivityRestricted,
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View timesheets and punches",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Record",
				Description: "Clock somebody in or out, or record hours by hand",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Correct a punch on an open week",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Remove a punch",
			},
			{
				Operation:   permission.OpSubmit,
				DisplayName: "Submit",
				Description: "Hand a week over",
			},
			{
				Operation:   permission.OpApprove,
				DisplayName: "Approve",
				Description: "Approve a week",
			},
			{
				Operation:   permission.OpReject,
				DisplayName: "Reject",
				Description: "Send a week back",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Run payroll over approved weeks, and void a payroll export",
			},
		},
		Scopes: []permissionspec.ScopeSupport{
			{
				Scope:      permission.DataScopeTeam,
				EnforcedBy: permissionspec.ScopeEnforcedByApplication,
			},
		},
	}
}

func workerPolicyResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceWorkerPolicy.String(),
		DisplayName:    "Policy",
		Description:    "Handbooks and policies people sign from the portal, and who has signed them",
		Category:       "Human resources",
		ReferenceView:  []string{"title", "versionLabel"},
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View policies and signatures",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Publish",
				Description: "Publish a policy",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Revise a policy, give it a new version, or retire it",
			},
		},
	}
}

func profileChangeRequestResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceProfileChangeRequest.String(),
		DisplayName: "Profile Change",
		Description: "Changes drivers ask to make to their own record from the portal",
		Category:    "Human resources",
		Operations: []permissionspec.OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View the queue"},
			{
				Operation:   permission.OpApprove,
				DisplayName: "Approve",
				Description: "Apply a requested change to the record",
			},
			{
				Operation:   permission.OpReject,
				DisplayName: "Reject",
				Description: "Turn a request down",
			},
		},
		Scopes: []permissionspec.ScopeSupport{
			{
				Scope:      permission.DataScopeTeam,
				EnforcedBy: permissionspec.ScopeEnforcedByApplication,
			},
		},
	}
}

func jobPositionResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceJobPosition.String(),
		DisplayName:    "Job Position",
		Description:    "Job titles the roster is counted by, and the reporting lines between them",
		Category:       "Human resources",
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View positions and headcount",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Add a position",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Edit a position or archive it",
			},
			{
				Operation:   permission.OpAssign,
				DisplayName: "Assign",
				Description: "Put a worker into a position or under a manager",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export job position data",
			},
		},
	}
}

func approvalDelegationResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceApprovalDelegation.String(),
		DisplayName:    "Approval Delegation",
		Description:    "Handing approval to somebody else while a manager is away",
		Category:       "Human resources",
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "See who is approving in whose place",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Delegate",
				Description: "Hand your own approvals to somebody else",
			},
			{
				Operation:   permission.OpManage,
				DisplayName: "Manage",
				Description: "Delegate on behalf of another manager, and revoke anybody's delegation",
			},
		},
	}
}

func workerInjuryResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourceWorkerInjury.String(),
		DisplayName:   "Injury & Illness",
		Description:   "Injury and illness cases, the OSHA 300 log and the 300A annual summary",
		Category:      "Human resources",
		ReferenceView: []string{"caseNumber", "caseYear", "status"},
		// The log carries body parts, treatment and workers' compensation
		// claims. It is granted separately from the rest of a worker's record
		// because most people who need the roster have no business reading it.
		// What the 300 log itself posts (the case, its classification, the days
		// lost and whether it is closed) is Internal; what happened to the
		// person's body, how it was treated and the claim stay Restricted.
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":               permission.SensitivityInternal,
			"businessUnitId":   permission.SensitivityInternal,
			"organizationId":   permission.SensitivityInternal,
			"workerId":         permission.SensitivityInternal,
			"caseNumber":       permission.SensitivityInternal,
			"caseYear":         permission.SensitivityInternal,
			"classification":   permission.SensitivityInternal,
			"illnessType":      permission.SensitivityInternal,
			"status":           permission.SensitivityInternal,
			"recordable":       permission.SensitivityInternal,
			"occurredAt":       permission.SensitivityInternal,
			"reportedAt":       permission.SensitivityInternal,
			"returnedToWorkAt": permission.SensitivityInternal,
			"daysAway":         permission.SensitivityInternal,
			"daysRestricted":   permission.SensitivityInternal,
			"privacyCase":      permission.SensitivityInternal,
			"logName":          permission.SensitivityInternal,
			"claimStatus":      permission.SensitivityInternal,
			"safetyEventId":    permission.SensitivityInternal,
			"version":          permission.SensitivityInternal,
			"createdAt":        permission.SensitivityInternal,
			"updatedAt":        permission.SensitivityInternal,
		},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View cases and the OSHA log",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Record",
				Description: "Record a case",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Correct or close a case",
			},
			{
				Operation:   permission.OpManage,
				DisplayName: "Certify",
				Description: "Certify and record the posting of the 300A summary",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Delete a case recorded in error",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export the OSHA log",
			},
		},
	}
}

func dotRandomPoolResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourceDOTRandomPool.String(),
		DisplayName:   "Random Testing Pool",
		Description:   "Random selection pools and the rounds drawn from them",
		Category:      "Human resources",
		ReferenceView: []string{"code", "name", "period"},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View pools and past draws",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create a pool",
			},
			{Operation: permission.OpUpdate, DisplayName: "Update", Description: "Edit a pool"},
			{
				Operation:   permission.OpManage,
				DisplayName: "Draw",
				Description: "Run, finalize or cancel a selection round",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export selection evidence",
			},
		},
	}
}

func workerSafetyEventResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceWorkerSafetyEvent.String(),
		DisplayName:    "Worker Safety Event",
		Description:    "Accidents, incidents, near misses, citations and inspections on a worker's record",
		Category:       "Human resources",
		ReferenceView:  []string{"kind", "occurredAt"},
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View safety events and scorecards",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Record safety events",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Edit safety events",
			},
			{
				Operation:   permission.OpClose,
				DisplayName: "Close",
				Description: "Resolve, review and reopen safety events",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Delete open safety events recorded in error",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export safety data",
			},
		},
	}
}

func workerDisciplinaryActionResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourceWorkerDisciplinaryAction.String(),
		DisplayName:   "Disciplinary Action",
		Description:   "Progressive-discipline actions issued to workers",
		Category:      "Human resources",
		ReferenceView: []string{"status", "issuedAt"},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View disciplinary actions",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Issue",
				Description: "Issue disciplinary actions",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: "Rescind",
				Description: "Rescind disciplinary actions",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export disciplinary data",
			},
		},
	}
}

func workerRecognitionResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceWorkerRecognition.String(),
		DisplayName:    "Worker Recognition",
		Description:    "Praise and milestones recorded for workers",
		Category:       "Human resources",
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View recognition",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Give",
				Description: "Record recognition",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Remove recognition",
			},
		},
	}
}

func performanceReviewTemplateResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourcePerformanceReviewTemplate.String(),
		DisplayName:   "Review Template",
		Description:   "Rating items performance reviews are scored against",
		Category:      "Human resources",
		ReferenceView: []string{"code", "name", "cadenceMonths", "items"},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View review templates",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create review templates",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify review templates",
			},
			{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Deactivate review templates",
			},
			{
				Operation:   permission.OpRestore,
				DisplayName: "Restore",
				Description: "Reactivate review templates",
			},
		},
	}
}

func performanceReviewResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourcePerformanceReview.String(),
		DisplayName:   "Performance Review",
		Description:   "Reviews drafted for workers, their ratings, goals and sign-off",
		Category:      "Human resources",
		ReferenceView: []string{"title", "status", "periodEnd"},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View performance reviews",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Start reviews",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Edit review drafts",
			},
			{
				Operation:   permission.OpSubmit,
				DisplayName: "Submit",
				Description: "Submit reviews to the worker",
			},
			{
				Operation:   permission.OpClose,
				DisplayName: "Close",
				Description: "Close and reopen reviews",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Delete review drafts",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export review data",
			},
		},
	}
}

func workerCredentialTypeResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceWorkerCredentialType.String(),
		DisplayName: "Credential Type",
		Description: "Catalog of credential types workers can hold and which are required",
		Category:    "Human resources",
		ReferenceView: []string{
			"code",
			"name",
			"category",
			"isRequired",
			"profileField",
			"validityMonths",
			"sortOrder",
			"requiresNumber",
			"requiresDocument",
			"renewalWindowDays",
			"isSystem",
		},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View credential types",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create credential types",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify credential types",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export credential type data",
			},
			{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Deactivate credential types",
			},
			{
				Operation:   permission.OpRestore,
				DisplayName: "Restore",
				Description: "Reactivate credential types",
			},
		},
	}
}

func ptoPolicyResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourcePTOPolicy.String(),
		DisplayName:   "PTO Policy",
		Description:   "Paid time off accrual policies and the workers assigned to them",
		Category:      "Human resources",
		ReferenceView: []string{"code", "name", "status"},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View PTO policies",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create PTO policies",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify PTO policies",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export PTO policy data",
			},
			{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Deactivate PTO policies",
			},
			{
				Operation:   permission.OpRestore,
				DisplayName: "Restore",
				Description: "Reactivate PTO policies",
			},
			{
				Operation:   permission.OpAssign,
				DisplayName: "Assign",
				Description: "Assign workers to PTO policies",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}
