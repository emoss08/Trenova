package permission

func (r *Registry) registerComplianceResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceQualification.String(),
		DisplayName: "Qualification",
		Description: "Driver qualification management",
		Category:    "Human resources",
		Operations:  standardOpsWithDelete,
		// Where a previous employer's safety performance history request
		// stands (who, when, how often followed up, what kind of answer) is
		// Internal. The employer's contact details, the driver's drug and
		// alcohol history and what the employer said stay Restricted.
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"id":                            SensitivityInternal,
			"businessUnitId":                SensitivityInternal,
			"organizationId":                SensitivityInternal,
			"workerId":                      SensitivityInternal,
			"employerName":                  SensitivityInternal,
			"employerDotNumber":             SensitivityInternal,
			"employerMcNumber":              SensitivityInternal,
			"employedFrom":                  SensitivityInternal,
			"employedTo":                    SensitivityInternal,
			"wasDotRegulated":               SensitivityInternal,
			"status":                        SensitivityInternal,
			"method":                        SensitivityInternal,
			"requestedAt":                   SensitivityInternal,
			"responseReceivedAt":            SensitivityInternal,
			"lastFollowUpAt":                SensitivityInternal,
			"followUpCount":                 SensitivityInternal,
			"drugAlcoholResponseReceivedAt": SensitivityInternal,
			"hadAccidents":                  SensitivityInternal,
			"accidentCount":                 SensitivityInternal,
			"version":                       SensitivityInternal,
			"createdAt":                     SensitivityInternal,
			"updatedAt":                     SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDocument.String(),
		DisplayName: "Document",
		Description: "Uploaded document records and intelligence output",
		Category:    "Documents",
		ReferenceView: []string{
			"originalName",
			"fileType",
			"fileSize",
		},
		Operations: relabel(relabel(operations(
			OpRead, OpCreate, OpUpdate, OpExport, OpImport, OpDelete, OpApprove, OpReject,
		), OpCreate, "Create", "Upload documents and cancel uploads"), OpUpdate, "Update", "Modify documents and restore deleted ones"),
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDocumentType.String(),
		DisplayName: "Document Type",
		Description: "Document type configuration and classification targets",
		Category:    "Documents",
		ReferenceView: []string{
			"code",
			"name",
			"color",
			"documentCategory",
			"documentClassification",
		},
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDocumentControl.String(),
		DisplayName:        "Document Control",
		Description:        "Document extraction settings",
		Category:           "Documents",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDocumentParsingRule.String(),
		DisplayName: "Document Parsing Rule",
		Description: "Tenant-managed parsing rules, versions, fixtures, and simulations",
		Category:    "Intake",
		Operations: relabel(append(GetAllOperations(), OperationDefinition{
			Operation:   OpDelete,
			DisplayName: "Delete",
			Description: "Delete rule sets and fixtures",
		}), OpActivate, "Activate", "Activate and publish parsing rule versions"),
		DefaultSensitivity: SensitivityInternal,
	})

	// The intake queue is desk work, so its data scope decides whose scans a
	// person sees: their own, or everybody's. Filing an item also needs
	// document create on the record it lands on, which is checked separately.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCaptureBatch.String(),
		DisplayName: "Intake",
		Description: "Scans and print jobs waiting in intake, and filing them onto records",
		Category:    "Intake",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "View scanned and printed batches in intake",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Capture",
				Description: "Scan or print into Trenova from a paired device, approve device pairings and cancel capture requests",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "File",
				Description: "Split, reorder and file captured pages onto records",
			},
			{
				Operation:   OpDelete,
				DisplayName: "Discard",
				Description: "Discard captured pages without filing them",
			},
		},
		DefaultSensitivity: SensitivityInternal,
		Scopes: []ScopeSupport{
			{Scope: DataScopeOwn, EnforcedBy: ScopeEnforcedByApplication},
		},
	})

	// Devices are separate from batches because they are a different
	// privilege: scanning is ordinary work, while seeing and revoking every
	// machine in the organization that can act for somebody is not. A person
	// always manages their own devices without it.
	_ = r.Register(&ResourceDefinition{
		Resource:      ResourceCaptureDevice.String(),
		DisplayName:   "Capture Device",
		Description:   "Paired Trenova Capture installs across the organization",
		Category:      "Intake",
		ReferenceView: []string{"name", "machineName"},
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "View every paired device and who it acts for",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Revoke",
				Description: "Revoke any paired device",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceCaptureProfile.String(),
		DisplayName:        "Capture Profile",
		Description:        "Named scan settings offered to people who scan",
		Category:           "Intake",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityInternal,
	})
}
