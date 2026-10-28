package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *Registry) registerComplianceResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceQualification.String(),
		DisplayName: "Qualification",
		Description: "Driver qualification management",
		Category:    "Human resources",
		Operations:  standardOpsWithDelete,
		// Where a previous employer's safety performance history request
		// stands (who, when, how often followed up, what kind of answer) is
		// Internal. The employer's contact details, the driver's drug and
		// alcohol history and what the employer said stay Restricted.
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":                            permission.SensitivityInternal,
			"businessUnitId":                permission.SensitivityInternal,
			"organizationId":                permission.SensitivityInternal,
			"workerId":                      permission.SensitivityInternal,
			"employerName":                  permission.SensitivityInternal,
			"employerDotNumber":             permission.SensitivityInternal,
			"employerMcNumber":              permission.SensitivityInternal,
			"employedFrom":                  permission.SensitivityInternal,
			"employedTo":                    permission.SensitivityInternal,
			"wasDotRegulated":               permission.SensitivityInternal,
			"status":                        permission.SensitivityInternal,
			"method":                        permission.SensitivityInternal,
			"requestedAt":                   permission.SensitivityInternal,
			"responseReceivedAt":            permission.SensitivityInternal,
			"lastFollowUpAt":                permission.SensitivityInternal,
			"followUpCount":                 permission.SensitivityInternal,
			"drugAlcoholResponseReceivedAt": permission.SensitivityInternal,
			"hadAccidents":                  permission.SensitivityInternal,
			"accidentCount":                 permission.SensitivityInternal,
			"version":                       permission.SensitivityInternal,
			"createdAt":                     permission.SensitivityInternal,
			"updatedAt":                     permission.SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceDocument.String(),
		DisplayName: "Document",
		Description: "Uploaded document records and intelligence output",
		Category:    "Documents",
		ReferenceView: []string{
			"originalName",
			"fileType",
			"fileSize",
		},
		Operations: relabel(relabel(operations(
			permission.OpRead,
			permission.OpCreate,
			permission.OpUpdate,
			permission.OpExport,
			permission.OpImport,
			permission.OpDelete,
			permission.OpApprove,
			permission.OpReject,
		), permission.OpCreate, "Create", "Upload documents and cancel uploads"), permission.OpUpdate, "Update", "Modify documents and restore deleted ones"),
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceDocumentType.String(),
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
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceDocumentControl.String(),
		DisplayName:        "Document Control",
		Description:        "Document extraction settings",
		Category:           "Documents",
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityInternal,
	})

	// The intake queue is desk work, so its data scope decides whose scans a
	// person sees: their own, or everybody's. Filing an item also needs
	// document create on the record it lands on, which is checked separately.
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceCaptureBatch.String(),
		DisplayName: "Intake",
		Description: "Scans and print jobs waiting in intake, and filing them onto records",
		Category:    "Intake",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View scanned and printed batches in intake",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Capture",
				Description: "Scan or print into Trenova from a paired device, approve device pairings and cancel capture requests",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "File",
				Description: "Split, reorder and file captured pages onto records",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Discard",
				Description: "Discard captured pages without filing them",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
		Scopes: []ScopeSupport{
			{Scope: permission.DataScopeOwn, EnforcedBy: ScopeEnforcedByApplication},
		},
	})

	// Devices are separate from batches because they are a different
	// privilege: scanning is ordinary work, while seeing and revoking every
	// machine in the organization that can act for somebody is not. A person
	// always manages their own devices without it.
	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceCaptureDevice.String(),
		DisplayName:   "Capture Device",
		Description:   "Paired Trenova Capture installs across the organization",
		Category:      "Intake",
		ReferenceView: []string{"name", "machineName"},
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View every paired device and who it acts for",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Revoke",
				Description: "Revoke any paired device",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceCaptureProfile.String(),
		DisplayName:        "Capture Profile",
		Description:        "Named scan settings offered to people who scan",
		Category:           "Intake",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityInternal,
	})
}
