package permissioncatalog

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *Registry) registerAdministrationResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceOrganization.String(),
		DisplayName:        "Organization",
		Description:        "Organization settings and configuration",
		Category:           "Organization settings",
		ReferenceView:      []string{"name", "logoUrl"},
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceBusinessUnit.String(),
		DisplayName:        "Business Unit",
		Description:        "Business unit management",
		Category:           "Organization settings",
		ReferenceView:      []string{"name", "code"},
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceUser.String(),
		DisplayName:   "User",
		Description:   "User account management",
		Category:      "Users & security",
		ReferenceView: []string{"name", "username", "profilePicUrl", "thumbnailUrl"},
		Operations: []OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View user accounts"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create new users",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify user accounts",
			},
			{
				Operation:   permission.OpAssign,
				DisplayName: "Assign Roles",
				Description: "Assign roles to users",
			},
			{
				Operation:   permission.OpUnassign,
				DisplayName: "Unassign Roles",
				Description: "Remove roles from users",
			},
		},
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceRole.String(),
		DisplayName:   "Role",
		Description:   "Role and permission management",
		Category:      "Users & security",
		ReferenceView: []string{"name", "isSystem"},
		Operations: []OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View roles"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create new roles",
			},
			{Operation: permission.OpUpdate, DisplayName: "Update", Description: "Modify roles"},
			{Operation: permission.OpDelete, DisplayName: "Delete", Description: "Delete roles"},
		},
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceAuditLog.String(),
		DisplayName:        "Audit Log",
		Description:        "System audit logs",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceTableConfiguration.String(),
		DisplayName: "Table Configuration",
		Description: "Saved table views. Anyone may keep private views; sharing one with the organization needs these",
		Category:    "Organization settings",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "See saved views on the home page",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Share",
				Description: "Share a view with the organization",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Change shared",
				Description: "Change a view shared with the organization",
			},
		},
		DefaultSensitivity: permission.SensitivityPublic,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceCustomFieldDefinition.String(),
		DisplayName: "Custom Field Definition",
		Description: "Custom field definitions and resource assignments",
		Category:    "Organization settings",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View custom field definitions",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create custom field definitions",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify custom field definitions",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Delete custom field definitions",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceSequenceConfig.String(),
		DisplayName:        "Sequence Configuration",
		Description:        "Sequence generator configuration",
		Category:           "Organization settings",
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceIntegration.String(),
		DisplayName:        "Integration",
		Description:        "External integration configuration and synchronization",
		Category:           "Organization settings",
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceEmailProfile.String(),
		DisplayName:        "Email Profile",
		Description:        "Transactional email sender profiles and assignments",
		Category:           "Organization settings",
		ReferenceView:      []string{"name"},
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityRestricted,
		// Who mail comes from and where replies go is printed on every message
		// sent, so it is Internal: it is how an agent chooses a profile. How
		// the profile connects to its provider stays Restricted.
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":             permission.SensitivityInternal,
			"businessUnitId": permission.SensitivityInternal,
			"organizationId": permission.SensitivityInternal,
			"name":           permission.SensitivityInternal,
			"description":    permission.SensitivityInternal,
			"senderName":     permission.SensitivityInternal,
			"senderEmail":    permission.SensitivityInternal,
			"replyToEmail":   permission.SensitivityInternal,
			"status":         permission.SensitivityInternal,
			"version":        permission.SensitivityInternal,
			"createdAt":      permission.SensitivityInternal,
			"updatedAt":      permission.SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceEmailLog.String(),
		DisplayName:        "Email Log",
		Description:        "Transactional email send and delivery logs",
		Category:           "Organization settings",
		ReferenceView:      []string{"status", "sentAt", "failedAt", "providerMessageId"},
		Operations:         readOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceEmailSuppression.String(),
		DisplayName:        "Email Suppression",
		Description:        "Email bounce, complaint, and manual suppression records",
		Category:           "Organization settings",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceEDI.String(),
		DisplayName: "EDI",
		Description: "EDI partner setup, mapping profiles, and load tender transfers",
		Category:    "EDI",
		ReferenceView: []string{
			"kind",
			"status",
			"code",
			"name",
			"enabledForInbound",
			"enabledForOutbound",
		},
		Operations: relabel(
			standardOpsWithDelete,
			permission.OpUpdate,
			"Update",
			"Modify EDI settings, and apply, approve, reject, cancel, activate and archive EDI changes, transfers, connections and template versions",
		),
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"acceptedAt":               permission.SensitivityInternal,
			"ackLastError":             permission.SensitivityInternal,
			"ackMessageId":             permission.SensitivityInternal,
			"ackReceivedAt":            permission.SensitivityInternal,
			"ackStatus":                permission.SensitivityInternal,
			"ackStatusCounts":          permission.SensitivityInternal,
			"address":                  permission.SensitivityInternal,
			"answersEverySet":          permission.SensitivityInternal,
			"asOf":                     permission.SensitivityInternal,
			"bol":                      permission.SensitivityInternal,
			"businessUnitId":           permission.SensitivityInternal,
			"canAccept":                permission.SensitivityInternal,
			"canDecide":                permission.SensitivityInternal,
			"capabilities":             permission.SensitivityInternal,
			"charges":                  permission.SensitivityRestricted,
			"checksum":                 permission.SensitivityInternal,
			"city":                     permission.SensitivityInternal,
			"code":                     permission.SensitivityInternal,
			"commodities":              permission.SensitivityInternal,
			"commodity":                permission.SensitivityInternal,
			"communicationProfileId":   permission.SensitivityInternal,
			"communicationProfileName": permission.SensitivityInternal,
			"config":                   permission.SensitivityConfidential,
			"connection":               permission.SensitivityInternal,
			"contactEmail":             permission.SensitivityRestricted,
			"contactName":              permission.SensitivityInternal,
			"contactPhone":             permission.SensitivityRestricted,
			"count":                    permission.SensitivityInternal,
			"counterparty":             permission.SensitivityInternal,
			"country":                  permission.SensitivityInternal,
			"createdAt":                permission.SensitivityInternal,
			"customerId":               permission.SensitivityInternal,
			"customerLabel":            permission.SensitivityInternal,
			"decoded":                  permission.SensitivityInternal,
			"defaultMappingProfileId":  permission.SensitivityInternal,
			"defaultTransport":         permission.SensitivityInternal,
			"defaultTransportId":       permission.SensitivityInternal,
			"deliveryAttempts":         permission.SensitivityInternal,
			"deliveryLastAttemptAt":    permission.SensitivityInternal,
			"deliveryLastError":        permission.SensitivityInternal,
			"deliveryRemotePath":       permission.SensitivityInternal,
			"deliverySentAt":           permission.SensitivityInternal,
			"deliveryStatus":           permission.SensitivityInternal,
			"description":              permission.SensitivityInternal,
			"diagnosticCount":          permission.SensitivityInternal,
			"diagnostics":              permission.SensitivityInternal,
			"direction":                permission.SensitivityInternal,
			"documentTypeId":           permission.SensitivityInternal,
			"ediConnectionId":          permission.SensitivityInternal,
			"ediPartnerId":             permission.SensitivityInternal,
			"elementPosition":          permission.SensitivityInternal,
			"enabledForInbound":        permission.SensitivityInternal,
			"enabledForOutbound":       permission.SensitivityInternal,
			"entityType":               permission.SensitivityInternal,
			"failureReason":            permission.SensitivityInternal,
			"file":                     permission.SensitivityInternal,
			"fileName":                 permission.SensitivityInternal,
			"generatedAt":              permission.SensitivityInternal,
			"groupControlNumber":       permission.SensitivityInternal,
			"inboundFileId":            permission.SensitivityInternal,
			"inboundMessageId":         permission.SensitivityInternal,
			"interchangeControlNumber": permission.SensitivityInternal,
			"internalOrganization":     permission.SensitivityInternal,
			"internalOrganizationId":   permission.SensitivityInternal,
			"invoice":                  permission.SensitivityInternal,
			"invoiceId":                permission.SensitivityInternal,
			"isaReceiverId":            permission.SensitivityInternal,
			"isaReceiverQualifier":     permission.SensitivityInternal,
			"isaSenderId":              permission.SensitivityInternal,
			"isaSenderQualifier":       permission.SensitivityInternal,
			"kind":                     permission.SensitivityInternal,
			"loadTenderInbound":        permission.SensitivityInternal,
			"loadTenderOutbound":       permission.SensitivityInternal,
			"location":                 permission.SensitivityInternal,
			"mappingSnapshot":          permission.SensitivityRestricted,
			"message":                  permission.SensitivityInternal,
			"messageCount":             permission.SensitivityInternal,
			"messages":                 permission.SensitivityInternal,
			"method":                   permission.SensitivityInternal,
			"name":                     permission.SensitivityInternal,
			"oldestGeneratedAt":        permission.SensitivityInternal,
			"organizationId":           permission.SensitivityInternal,
			"overdueAfterSeconds":      permission.SensitivityInternal,
			"overdueByPartner":         permission.SensitivityInternal,
			"overdueCount":             permission.SensitivityInternal,
			"overdueNotListed":         permission.SensitivityInternal,
			"partner":                  permission.SensitivityInternal,
			"partnerCode":              permission.SensitivityInternal,
			"partnerDocumentProfileId": permission.SensitivityInternal,
			"partnerId":                permission.SensitivityInternal,
			"partnerName":              permission.SensitivityInternal,
			"partners":                 permission.SensitivityInternal,
			"path":                     permission.SensitivityInternal,
			"pieces":                   permission.SensitivityInternal,
			"postalCode":               permission.SensitivityInternal,
			"processedAt":              permission.SensitivityInternal,
			"raw":                      permission.SensitivityRestricted,
			"rawContent":               permission.SensitivityRestricted,
			"rawPurgedAt":              permission.SensitivityInternal,
			"rawX12":                   permission.SensitivityRestricted,
			"receivedAt":               permission.SensitivityInternal,
			"recentDeadLettered":       permission.SensitivityInternal,
			"rejectedAt":               permission.SensitivityInternal,
			"rejectionReason":          permission.SensitivityInternal,
			"remotePath":               permission.SensitivityInternal,
			"requestedAt":              permission.SensitivityInternal,
			"revokedAt":                permission.SensitivityInternal,
			"secretState":              permission.SensitivityConfidential,
			"segmentCount":             permission.SensitivityInternal,
			"segmentId":                permission.SensitivityInternal,
			"sequence":                 permission.SensitivityInternal,
			"serviceTypeLabel":         permission.SensitivityInternal,
			"severity":                 permission.SensitivityInternal,
			"shipmentId":               permission.SensitivityInternal,
			"shipmentStatus":           permission.SensitivityInternal,
			"shipmentTypeLabel":        permission.SensitivityInternal,
			"since":                    permission.SensitivityInternal,
			"sizeBytes":                permission.SensitivityInternal,
			"source":                   permission.SensitivityInternal,
			"sourceBusinessUnitId":     permission.SensitivityInternal,
			"sourceOrganizationId":     permission.SensitivityInternal,
			"sourcePartner":            permission.SensitivityInternal,
			"sourcePartnerId":          permission.SensitivityInternal,
			"sourceShipmentId":         permission.SensitivityInternal,
			"state":                    permission.SensitivityInternal,
			"status":                   permission.SensitivityInternal,
			"stopCount":                permission.SensitivityInternal,
			"stops":                    permission.SensitivityInternal,
			"submittedAt":              permission.SensitivityInternal,
			"suggestedFix":             permission.SensitivityInternal,
			"suspendedAt":              permission.SensitivityInternal,
			"targetBusinessUnitId":     permission.SensitivityInternal,
			"targetOrganizationId":     permission.SensitivityInternal,
			"targetPartner":            permission.SensitivityInternal,
			"targetPartnerId":          permission.SensitivityInternal,
			"targetShipmentId":         permission.SensitivityInternal,
			"temperatureMax":           permission.SensitivityInternal,
			"temperatureMin":           permission.SensitivityInternal,
			"templateId":               permission.SensitivityInternal,
			"templateVersionId":        permission.SensitivityInternal,
			"tenderPayload":            permission.SensitivityRestricted,
			"timezone":                 permission.SensitivityInternal,
			"transactionControlNumber": permission.SensitivityInternal,
			"transactionCount":         permission.SensitivityInternal,
			"transactionSet":           permission.SensitivityInternal,
			"transfer":                 permission.SensitivityInternal,
			"transferId":               permission.SensitivityInternal,
			"type":                     permission.SensitivityInternal,
			"unresolvedMappings":       permission.SensitivityInternal,
			"updatedAt":                permission.SensitivityInternal,
			"version":                  permission.SensitivityInternal,
			"weight":                   permission.SensitivityInternal,
			"windowCloses":             permission.SensitivityInternal,
			"windowOpens":              permission.SensitivityInternal,
			"x12Version":               permission.SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceAPIKey.String(),
		DisplayName:        "API Key",
		Description:        "API key creation, rotation, revocation, and permission management",
		Category:           "Users & security",
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourcePlatformCatalog.String(),
		DisplayName:        "Platform Catalog",
		Description:        "Platform catalog and entitlement metadata administration",
		Category:           "Organization settings",
		Operations:         readOnlyOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceIdentityProvider.String(),
		DisplayName:        "Identity Provider",
		Description:        "OIDC identity provider configuration",
		Category:           "Users & security",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceSCIMDirectory.String(),
		DisplayName:        "SCIM Directory",
		Description:        "SCIM directory, token, and group mapping administration",
		Category:           "Users & security",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceProvisioningAudit.String(),
		DisplayName:        "Provisioning Audit",
		Description:        "SCIM provisioning activity",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceAccessPolicy.String(),
		DisplayName:        "Access Policy",
		Description:        "Organization access policy decisions",
		Category:           "Users & security",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceAuthEvent.String(),
		DisplayName:        "Authentication Event",
		Description:        "Authentication and sign-in activity",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceRiskDecision.String(),
		DisplayName:        "Risk Decision",
		Description:        "Authentication risk decisions",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceExternalIdentity.String(),
		DisplayName:        "External Identity",
		Description:        "Federated identity links",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceMFAAuthenticator.String(),
		DisplayName:        "MFA Authenticator",
		Description:        "MFA authenticator inventory",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceTableChangeAlert.String(),
		DisplayName: "Table Change Alert",
		Description: "Table change alert subscriptions and notifications",
		Category:    "Organization settings",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View table change alert subscriptions",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create table change alert subscriptions",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify, pause, and resume table change alert subscriptions",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Delete table change alert subscriptions",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"conditionMatch": permission.SensitivityInternal,
			"eventTypes":     permission.SensitivityInternal,
			"status":         permission.SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceHomeLayoutPreset.String(),
		DisplayName: "Home Screen Preset",
		Description: "Home screens administrators author and assign to roles",
		Category:    "Organization settings",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View home screen presets",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create home screen presets",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify, assign, and lock home screen presets",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Delete home screen presets",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	// Publishing is separated from editing because they carry different risk: a
	// draft changes nothing a customer sees, and activating one changes every
	// invoice and notice the organization sends from that moment on.
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceDocumentTemplate.String(),
		DisplayName: "Document & Message Template",
		Description: "Templates for customer documents, emails, and notifications",
		Category:    "Documents",
		Operations: relabel(append(slices.Clone(standardOpsWithDelete),
			OperationDefinition{
				Operation:   permission.OpActivate,
				DisplayName: "Publish",
				Description: "Publish a version, making it what customers receive",
			},
			OperationDefinition{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Retire a template version",
			},
			OperationDefinition{
				Operation:   permission.OpAssign,
				DisplayName: "Assign",
				Description: "Assign a template to a specific customer",
			},
			OperationDefinition{
				Operation:   permission.OpUnassign,
				DisplayName: "Unassign",
				Description: "Remove a customer's template override",
			},
		), permission.OpUpdate, "Update", "Modify templates and send test messages"),
		DefaultSensitivity: permission.SensitivityInternal,
	})
}
