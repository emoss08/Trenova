package permission

import (
	"slices"
)

func (r *Registry) registerAdministrationResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceOrganization.String(),
		DisplayName:        "Organization",
		Description:        "Organization settings and configuration",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceBusinessUnit.String(),
		DisplayName:        "Business Unit",
		Description:        "Business unit management",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceUser.String(),
		DisplayName: "User",
		Description: "User account management",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View user accounts"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create new users"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify user accounts"},
			{
				Operation:   OpAssign,
				DisplayName: "Assign Roles",
				Description: "Assign roles to users",
			},
			{
				Operation:   OpUnassign,
				DisplayName: "Unassign Roles",
				Description: "Remove roles from users",
			},
		},
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceRole.String(),
		DisplayName: "Role",
		Description: "Role and permission management",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View roles"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create new roles"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify roles"},
		},
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAuditLog.String(),
		DisplayName:        "Audit Log",
		Description:        "System audit logs",
		Category:           "Administration",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceTableConfiguration.String(),
		DisplayName:        "Table Configuration",
		Description:        "User table preferences and configurations",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityPublic,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCustomFieldDefinition.String(),
		DisplayName: "Custom Field Definition",
		Description: "Custom field definitions and resource assignments",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View custom field definitions"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Create custom field definitions",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Modify custom field definitions",
			},
			{
				Operation:   OpDelete,
				DisplayName: "Delete",
				Description: "Delete custom field definitions",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceSequenceConfig.String(),
		DisplayName:        "Sequence Configuration",
		Description:        "Sequence generator configuration",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceIntegration.String(),
		DisplayName:        "Integration",
		Description:        "External integration configuration and synchronization",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceEmailProfile.String(),
		DisplayName:        "Email Profile",
		Description:        "Transactional email sender profiles and assignments",
		Category:           "Administration",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceEmailLog.String(),
		DisplayName:        "Email Log",
		Description:        "Transactional email send and delivery logs",
		Category:           "Administration",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceEmailSuppression.String(),
		DisplayName:        "Email Suppression",
		Description:        "Email bounce, complaint, and manual suppression records",
		Category:           "Administration",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceEDI.String(),
		DisplayName:        "EDI",
		Description:        "EDI partner setup, mapping profiles, and load tender transfers",
		Category:           "Administration",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAPIKey.String(),
		DisplayName:        "API Key",
		Description:        "API key creation, rotation, revocation, and permission management",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourcePlatformCatalog.String(),
		DisplayName:        "Platform Catalog",
		Description:        "Platform catalog and entitlement metadata administration",
		Category:           "Administration",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDocumentOperation.String(),
		DisplayName: "Document Operation",
		Description: "Document diagnostics, extraction, preview, and search operations",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View document diagnostics"},
			{Operation: OpUpdate, DisplayName: "Run", Description: "Run document operations"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceIdentityProvider.String(),
		DisplayName:        "Identity Provider",
		Description:        "OIDC identity provider configuration",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceSCIMDirectory.String(),
		DisplayName:        "SCIM Directory",
		Description:        "SCIM directory, token, and group mapping administration",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceProvisioningAudit.String(),
		DisplayName:        "Provisioning Audit",
		Description:        "SCIM provisioning activity",
		Category:           "Administration",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAccessPolicy.String(),
		DisplayName:        "Access Policy",
		Description:        "Organization access policy decisions",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAuthEvent.String(),
		DisplayName:        "Authentication Event",
		Description:        "Authentication and sign-in activity",
		Category:           "Administration",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceRiskDecision.String(),
		DisplayName:        "Risk Decision",
		Description:        "Authentication risk decisions",
		Category:           "Administration",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceExternalIdentity.String(),
		DisplayName:        "External Identity",
		Description:        "Federated identity links",
		Category:           "Administration",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceMFAAuthenticator.String(),
		DisplayName:        "MFA Authenticator",
		Description:        "MFA authenticator inventory",
		Category:           "Administration",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceTableChangeAlert.String(),
		DisplayName: "Table Change Alert",
		Description: "Table change alert subscriptions and notifications",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View table change alert subscriptions",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Create table change alert subscriptions",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Modify, pause, and resume table change alert subscriptions",
			},
			{
				Operation:   OpDelete,
				DisplayName: "Delete",
				Description: "Delete table change alert subscriptions",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceHomeLayoutPreset.String(),
		DisplayName: "Home Screen Preset",
		Description: "Home screens administrators author and assign to roles",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View home screen presets",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Create home screen presets",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Modify, assign, and lock home screen presets",
			},
			{
				Operation:   OpDelete,
				DisplayName: "Delete",
				Description: "Delete home screen presets",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	// Publishing is separated from editing because they carry different risk: a
	// draft changes nothing a customer sees, and activating one changes every
	// invoice and notice the organization sends from that moment on.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDocumentTemplate.String(),
		DisplayName: "Document & Message Template",
		Description: "Templates for customer documents, emails, and notifications",
		Category:    "Administration",
		Operations: append(slices.Clone(standardOpsWithDelete),
			OperationDefinition{
				Operation:   OpActivate,
				DisplayName: "Publish",
				Description: "Publish a version, making it what customers receive",
			},
			OperationDefinition{
				Operation:   OpArchive,
				DisplayName: "Archive",
				Description: "Retire a template version",
			},
			OperationDefinition{
				Operation:   OpAssign,
				DisplayName: "Assign",
				Description: "Assign a template to a specific customer",
			},
			OperationDefinition{
				Operation:   OpUnassign,
				DisplayName: "Unassign",
				Description: "Remove a customer's template override",
			},
		),
		DefaultSensitivity: SensitivityInternal,
	})
}
