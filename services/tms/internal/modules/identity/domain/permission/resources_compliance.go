package permission

func (r *Registry) registerComplianceResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceQualification.String(),
		DisplayName:        "Qualification",
		Description:        "Driver qualification management",
		Category:           "Compliance",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDocumentClassification.String(),
		DisplayName:        "Document Classification",
		Description:        "Document classification definitions",
		Category:           "Compliance",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDocument.String(),
		DisplayName: "Document",
		Description: "Uploaded document records and intelligence output",
		Category:    "Compliance",
		Operations: operations(
			OpRead, OpCreate, OpUpdate, OpExport, OpImport, OpDelete, OpApprove, OpReject,
		),
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDocumentType.String(),
		DisplayName:        "Document Type",
		Description:        "Document type configuration and classification targets",
		Category:           "Compliance",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDocumentControl.String(),
		DisplayName:        "Document Control",
		Description:        "Document intelligence and OCR settings",
		Category:           "Compliance",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDocumentParsingRule.String(),
		DisplayName: "Document Parsing Rule",
		Description: "Tenant-managed parsing rules, versions, fixtures, and simulations",
		Category:    "Compliance",
		Operations: append(GetAllOperations(), OperationDefinition{
			Operation:   OpDelete,
			DisplayName: "Delete",
			Description: "Delete rule sets and fixtures",
		}),
		DefaultSensitivity: SensitivityInternal,
	})

	// The intake queue is desk work, so its data scope decides whose scans a
	// person sees: their own, or everybody's. Filing an item also needs
	// document create on the record it lands on, which is checked separately.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCaptureBatch.String(),
		DisplayName: "Document Capture",
		Description: "Scans and print jobs waiting in intake, and filing them onto records",
		Category:    "Compliance",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View scanned and printed batches in intake",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Capture",
				Description: "Scan or print into Trenova from a paired device",
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
	})

	// Devices are separate from batches because they are a different
	// privilege: scanning is ordinary work, while seeing and revoking every
	// machine in the organization that can act for somebody is not. A person
	// always manages their own devices without it.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCaptureDevice.String(),
		DisplayName: "Capture Device",
		Description: "Paired Trenova Capture installs across the organization",
		Category:    "Compliance",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
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
		Category:           "Compliance",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityInternal,
	})
}
