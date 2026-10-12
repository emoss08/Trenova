package permission

func (r *Registry) registerWorkerResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceWorker.String(),
		DisplayName: "Worker",
		Description: "Driver and worker management",
		Category:    "Workers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View workers"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add new workers"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify worker information"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export worker data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import worker data"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Archive workers"},
			{Operation: OpRestore, DisplayName: "Restore", Description: "Restore archived workers"},
		},
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"id":                     SensitivityInternal,
			"businessUnitId":         SensitivityInternal,
			"organizationId":         SensitivityInternal,
			"workerId":               SensitivityInternal,
			"status":                 SensitivityInternal,
			"type":                   SensitivityInternal,
			"driverType":             SensitivityInternal,
			"firstName":              SensitivityInternal,
			"lastName":               SensitivityInternal,
			"wholeName":              SensitivityInternal,
			"positionId":             SensitivityInternal,
			"userId":                 SensitivityInternal,
			"externalId":             SensitivityInternal,
			"profilePicUrl":          SensitivityInternal,
			"fleetCodeId":            SensitivityInternal,
			"canBeAssigned":          SensitivityInternal,
			"assignmentBlocked":      SensitivityInternal,
			"availableForDispatch":   SensitivityInternal,
			"eldExempt":              SensitivityInternal,
			"shortHaulExempt":        SensitivityInternal,
			"cdlClass":               SensitivityInternal,
			"cdlRestrictions":        SensitivityInternal,
			"endorsement":            SensitivityInternal,
			"hazmatExpiry":           SensitivityInternal,
			"licenseExpiry":          SensitivityInternal,
			"medicalCardExpiry":      SensitivityInternal,
			"twicExpiry":             SensitivityInternal,
			"physicalDueDate":        SensitivityInternal,
			"mvrDueDate":             SensitivityInternal,
			"complianceStatus":       SensitivityInternal,
			"isQualified":            SensitivityInternal,
			"lastComplianceCheck":    SensitivityInternal,
			"lastMvrCheck":           SensitivityInternal,
			"version":                SensitivityInternal,
			"createdAt":              SensitivityInternal,
			"updatedAt":              SensitivityInternal,
			"gender":                 SensitivityRestricted,
			"addressLine1":           SensitivityRestricted,
			"addressLine2":           SensitivityRestricted,
			"email":                  SensitivityRestricted,
			"phoneNumber":            SensitivityRestricted,
			"city":                   SensitivityRestricted,
			"postalCode":             SensitivityRestricted,
			"stateId":                SensitivityRestricted,
			"managerId":              SensitivityRestricted,
			"licenseStateId":         SensitivityRestricted,
			"emergencyContactName":   SensitivityRestricted,
			"emergencyContactPhone":  SensitivityRestricted,
			"hireDate":               SensitivityRestricted,
			"terminationDate":        SensitivityRestricted,
			"disqualificationReason": SensitivityRestricted,
			"medicalExaminerName":    SensitivityRestricted,
			"dob":                    SensitivityConfidential,
			"licenseNumber":          SensitivityConfidential,
			"twicCardNumber":         SensitivityConfidential,
			"medicalExaminerNpi":     SensitivityConfidential,
			"lastDrugTest":           SensitivityConfidential,
			"drugAlcoholStatus":      SensitivityConfidential,
			"returnToDutyStatus":     SensitivityConfidential,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceWorkerPTO.String(),
		DisplayName: "Worker PTO",
		Description: "Worker paid time off requests and approvals",
		Category:    "Workers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View worker PTO requests"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create worker PTO requests"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify worker PTO requests"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export worker PTO data"},
			{
				Operation:   OpApprove,
				DisplayName: "Approve",
				Description: "Approve worker PTO requests",
			},
			{Operation: OpReject, DisplayName: "Reject", Description: "Reject worker PTO requests"},
			{Operation: OpCancel, DisplayName: "Cancel", Description: "Cancel worker PTO requests"},
			{
				Operation:   OpManage,
				DisplayName: "Manage Balances",
				Description: "Adjust PTO balances and run accruals",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerCredential.String(),
		DisplayName:    "Worker Credential",
		Description:    "Licences, medical cards, endorsements and certificates held by workers",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View worker credentials"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Add or renew worker credentials",
			},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify worker credentials"},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export worker credential data",
			},
			{
				Operation:   OpApprove,
				DisplayName: "Verify",
				Description: "Mark a credential as verified against its document",
			},
			{
				Operation:   OpArchive,
				DisplayName: "Archive",
				Description: "Archive worker credentials",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerEmploymentEvent.String(),
		DisplayName:    "Employment Event",
		Description:    "Hires, terminations, leave, suspensions, transfers and other employment history",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View a worker's employment timeline",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Record",
				Description: "Record employment events, including terminations and rehires",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Amend",
				Description: "Correct recorded employment events",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export employment history"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerChecklist.String(),
		DisplayName:    "Worker Checklist",
		Description:    "Onboarding and offboarding checklists run for a worker",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View a worker's checklists"},
			{
				Operation:   OpCreate,
				DisplayName: "Start",
				Description: "Start a checklist for a worker",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Complete",
				Description: "Complete, skip or reopen checklist items",
			},
			{Operation: OpCancel, DisplayName: "Cancel", Description: "Cancel an open checklist"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceWorkerChecklistTemplate.String(),
		DisplayName: "Checklist Template",
		Description: "Reusable onboarding and offboarding checklists",
		Category:    "Workers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View checklist templates"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create checklist templates"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify checklist templates"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export checklist templates"},
			{
				Operation:   OpArchive,
				DisplayName: "Archive",
				Description: "Deactivate checklist templates",
			},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Reactivate checklist templates",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceOrgHoliday.String(),
		DisplayName: "Holiday Calendar",
		Description: "Organisation holidays and PTO blackout dates",
		Category:    "Workers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View the holiday calendar"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Add holidays and blackout dates",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Modify holidays and blackout dates",
			},
			{
				Operation:   OpDelete,
				DisplayName: "Delete",
				Description: "Remove holidays and blackout dates",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceTrainingCourse.String(),
		DisplayName: "Training Course",
		Description: "Catalog of training courses and which are required per driver type",
		Category:    "Workers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View training courses"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create training courses"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify training courses"},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export training course data",
			},
			{
				Operation:   OpArchive,
				DisplayName: "Archive",
				Description: "Deactivate training courses",
			},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Reactivate training courses",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerTraining.String(),
		DisplayName:    "Worker Training",
		Description:    "Courses assigned to workers, their results and certifications",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View worker training"},
			{Operation: OpAssign, DisplayName: "Assign", Description: "Assign courses to workers"},
			{
				Operation:   OpUpdate,
				DisplayName: "Record result",
				Description: "Record completions, scores and certificates",
			},
			{
				Operation:   OpCancel,
				DisplayName: "Waive or cancel",
				Description: "Waive a required course or withdraw an assignment",
			},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export worker training data",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerDOTTest.String(),
		DisplayName:    "Drug & Alcohol Testing",
		Description:    "Controlled-substances and alcohol tests, violations and Clearinghouse queries on a worker's record",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View test records, violations and Clearinghouse queries",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Record",
				Description: "Schedule collections and record results and queries",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Correct a test, query or violation record",
			},
			{
				Operation:   OpCancel,
				DisplayName: "Cancel",
				Description: "Void a collection that did not happen",
			},
			{
				Operation:   OpManage,
				DisplayName: "Manage violations",
				Description: "Record violations and drive the return-to-duty process",
			},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export testing data for an audit",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerLeave.String(),
		DisplayName:    "Leave & FMLA",
		Description:    "Leave cases, the FMLA entitlement drawn against them, and medical certification",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		// A leave case carries the reason somebody is off work and the medical
		// certification behind it. It is granted separately for the same reason
		// the injury log is.
		DefaultSensitivity: SensitivityRestricted,
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View leave cases and balances"},
			{
				Operation:   OpCreate,
				DisplayName: "Record",
				Description: "Open a leave case and record days",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Correct a case, its certification or its days",
			},
			{
				Operation:   OpApprove,
				DisplayName: "Decide",
				Description: "Approve, deny, designate and close a leave case",
			},
			{
				Operation:   OpManage,
				DisplayName: "Configure",
				Description: "Set the measurement method and entitlement",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export leave data"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceBenefitPlan.String(),
		DisplayName:    "Benefits",
		Description:    "Benefit plans and who is enrolled on them",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		// An enrollment says what cover somebody has and what they pay for it.
		// It is granted separately for the same reason the injury log is.
		DefaultSensitivity: SensitivityRestricted,
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View plans and enrollments"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add a plan"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Edit or archive a plan"},
			{
				Operation:   OpAssign,
				DisplayName: "Enroll",
				Description: "Put a worker on a plan, record a waiver, or end cover",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export benefits data"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceShiftTemplate.String(),
		DisplayName:    "Shift",
		Description:    "The repeating working patterns a roster is built from",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View shifts"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add a shift"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Edit a shift or retire it",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerSchedule.String(),
		DisplayName:    "Schedule",
		Description:    "The rota, who is on which shift, and what they would rather work",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View the rota"},
			{
				Operation:   OpAssign,
				DisplayName: "Assign",
				Description: "Put a worker on a shift or take them off one",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Set a worker's stated availability",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export the rota"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceShiftSwap.String(),
		DisplayName:    "Shift Swap",
		Description:    "Days drivers arrange to trade between themselves",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View swap requests"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Propose a swap"},
			{
				Operation:   OpApprove,
				DisplayName: "Approve",
				Description: "Approve a swap the two drivers have agreed",
			},
			{Operation: OpReject, DisplayName: "Reject", Description: "Reject a swap"},
			{
				Operation:   OpCancel,
				DisplayName: "Withdraw",
				Description: "Withdraw a swap on a driver's behalf",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceTimesheet.String(),
		DisplayName:    "Timesheet",
		Description:    "Hours worked by staff paid by the clock, and the sign-off on them",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		// A timesheet is a wage record. It is granted separately for the same
		// reason pay is.
		DefaultSensitivity: SensitivityRestricted,
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View timesheets and punches"},
			{
				Operation:   OpCreate,
				DisplayName: "Record",
				Description: "Clock somebody in or out, or record hours by hand",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Correct a punch on an open week",
			},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Remove a punch"},
			{Operation: OpSubmit, DisplayName: "Submit", Description: "Hand a week over"},
			{Operation: OpApprove, DisplayName: "Approve", Description: "Approve a week"},
			{Operation: OpReject, DisplayName: "Reject", Description: "Send a week back"},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Run payroll over approved weeks",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerPolicy.String(),
		DisplayName:    "Policy",
		Description:    "Handbooks and policies people sign from the portal, and who has signed them",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View policies and signatures"},
			{Operation: OpCreate, DisplayName: "Publish", Description: "Publish a policy"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Revise a policy, give it a new version, or retire it",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceProfileChangeRequest.String(),
		DisplayName:    "Profile Change",
		Description:    "Changes drivers ask to make to their own record from the portal",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View the queue"},
			{
				Operation:   OpApprove,
				DisplayName: "Approve",
				Description: "Apply a requested change to the record",
			},
			{Operation: OpReject, DisplayName: "Reject", Description: "Turn a request down"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceJobPosition.String(),
		DisplayName:    "Job Position",
		Description:    "Job titles the roster is counted by, and the reporting lines between them",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View positions and headcount"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add a position"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Edit a position or archive it",
			},
			{
				Operation:   OpAssign,
				DisplayName: "Assign",
				Description: "Put a worker into a position or under a manager",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export job position data"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceApprovalDelegation.String(),
		DisplayName:    "Approval Delegation",
		Description:    "Handing approval to somebody else while a manager is away",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "See who is approving in whose place",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Delegate",
				Description: "Hand your own approvals to somebody else",
			},
			{
				Operation:   OpManage,
				DisplayName: "Manage",
				Description: "Delegate on behalf of another manager, and revoke anybody's delegation",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerInjury.String(),
		DisplayName:    "Injury & Illness",
		Description:    "Injury and illness cases, the OSHA 300 log and the 300A annual summary",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		// The log carries body parts, treatment and workers' compensation
		// claims. It is granted separately from the rest of a worker's record
		// because most people who need the roster have no business reading it.
		DefaultSensitivity: SensitivityRestricted,
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View cases and the OSHA log"},
			{Operation: OpCreate, DisplayName: "Record", Description: "Record a case"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Correct or close a case"},
			{
				Operation:   OpManage,
				DisplayName: "Certify",
				Description: "Certify and record the posting of the 300A summary",
			},
			{
				Operation:   OpDelete,
				DisplayName: "Delete",
				Description: "Delete a case recorded in error",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export the OSHA log"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDOTRandomPool.String(),
		DisplayName: "Random Testing Pool",
		Description: "Random selection pools and the rounds drawn from them",
		Category:    "Workers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View pools and past draws"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create a pool"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Edit a pool"},
			{
				Operation:   OpManage,
				DisplayName: "Draw",
				Description: "Run and finalise a selection round",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export selection evidence"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerSafetyEvent.String(),
		DisplayName:    "Worker Safety Event",
		Description:    "Accidents, incidents, near misses, citations and inspections on a worker's record",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View safety events and scorecards",
			},
			{Operation: OpCreate, DisplayName: "Create", Description: "Record safety events"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Edit safety events"},
			{
				Operation:   OpClose,
				DisplayName: "Close",
				Description: "Resolve, review and reopen safety events",
			},
			{
				Operation:   OpDelete,
				DisplayName: "Delete",
				Description: "Delete open safety events recorded in error",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export safety data"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerDisciplinaryAction.String(),
		DisplayName:    "Disciplinary Action",
		Description:    "Progressive-discipline actions issued to workers",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View disciplinary actions"},
			{Operation: OpCreate, DisplayName: "Issue", Description: "Issue disciplinary actions"},
			{
				Operation:   OpCancel,
				DisplayName: "Rescind",
				Description: "Rescind disciplinary actions",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export disciplinary data"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceWorkerRecognition.String(),
		DisplayName:    "Worker Recognition",
		Description:    "Praise and milestones recorded for workers",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View recognition"},
			{Operation: OpCreate, DisplayName: "Give", Description: "Record recognition"},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Remove recognition"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourcePerformanceReviewTemplate.String(),
		DisplayName: "Review Template",
		Description: "Rating items performance reviews are scored against",
		Category:    "Workers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View review templates"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create review templates"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify review templates"},
			{
				Operation:   OpArchive,
				DisplayName: "Archive",
				Description: "Deactivate review templates",
			},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Reactivate review templates",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourcePerformanceReview.String(),
		DisplayName:    "Performance Review",
		Description:    "Reviews drafted for workers, their ratings, goals and sign-off",
		Category:       "Workers",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View performance reviews"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Start reviews"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Edit review drafts"},
			{
				Operation:   OpSubmit,
				DisplayName: "Submit",
				Description: "Submit reviews to the worker",
			},
			{Operation: OpClose, DisplayName: "Close", Description: "Close and reopen reviews"},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Delete review drafts"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export review data"},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceWorkerCredentialType.String(),
		DisplayName: "Credential Type",
		Description: "Catalog of credential types workers can hold and which are required",
		Category:    "Workers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View credential types"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create credential types"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify credential types"},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export credential type data",
			},
			{
				Operation:   OpArchive,
				DisplayName: "Archive",
				Description: "Deactivate credential types",
			},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Reactivate credential types",
			},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourcePTOPolicy.String(),
		DisplayName: "PTO Policy",
		Description: "Paid time off accrual policies and the workers assigned to them",
		Category:    "Workers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View PTO policies"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create PTO policies"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify PTO policies"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export PTO policy data"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Deactivate PTO policies"},
			{Operation: OpRestore, DisplayName: "Restore", Description: "Reactivate PTO policies"},
			{
				Operation:   OpAssign,
				DisplayName: "Assign",
				Description: "Assign workers to PTO policies",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})
}
