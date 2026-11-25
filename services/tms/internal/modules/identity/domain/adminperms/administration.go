package adminperms

import (
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

func administrationResources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		organizationResource(),
		businessUnitResource(),
		userResource(),
		roleResource(),
		auditLogResource(),
		securityAuditLogResource(),
		auditLegalHoldResource(),
		tableConfigurationResource(),
		customFieldDefinitionResource(),
		sequenceConfigResource(),
		integrationResource(),
		emailProfileResource(),
		emailLogResource(),
		emailSuppressionResource(),
		ediResource(),
		apiKeyResource(),
		platformCatalogResource(),
		identityProviderResource(),
		scimDirectoryResource(),
		provisioningAuditResource(),
		accessPolicyResource(),
		authEventResource(),
		riskDecisionResource(),
		externalIdentityResource(),
		mfaAuthenticatorResource(),
		supportAccessResource(),
		tableChangeAlertResource(),
		homeLayoutPresetResource(),
		documentTemplateResource(),
	}
}

func organizationResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceOrganization.String(),
		Feature:            platformcatalog.FeatureCoreTMS,
		DisplayName:        "Organization",
		Description:        "Organization settings and configuration",
		Category:           "Organization settings",
		ReferenceView:      []string{"name", "logoUrl", "iconUrl"},
		Operations:         permissionspec.StandardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldFeatures: map[string]platformcatalog.FeatureKey{
			"brokerageEnabled": platformcatalog.FeatureCarrierBrokerage,
		},
	}
}

func businessUnitResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceBusinessUnit.String(),
		Feature:            platformcatalog.FeatureCoreTMS,
		DisplayName:        "Business Unit",
		Description:        "Business unit management",
		Category:           "Organization settings",
		ReferenceView:      []string{"name", "code"},
		Operations:         permissionspec.StandardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func userResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourceUser.String(),
		Feature:       platformcatalog.FeatureAdministration,
		DisplayName:   "User",
		Description:   "User account management",
		Category:      "Users & security",
		ReferenceView: []string{"name", "username", "photo"},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View user accounts",
			},
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
	}
}

func roleResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourceRole.String(),
		Feature:       platformcatalog.FeatureAdministration,
		DisplayName:   "Role",
		Description:   "Role and permission management",
		Category:      "Users & security",
		ReferenceView: []string{"name", "isSystem"},
		Operations: []permissionspec.OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View roles"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create new roles",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify roles",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Delete roles",
			},
		},
		DefaultSensitivity: permission.SensitivityConfidential,
	}
}

func auditLogResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceAuditLog.String(),
		Feature:     platformcatalog.FeatureAdministration,
		DisplayName: "Audit Log",
		Description: "The organization's audit log, narrowed to the kinds of record the person can read",
		Category:    "Users & security",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "Read the audit log for the kinds of record you can read",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export the audit log to a CSV or JSON Lines file",
			},
		},
		DefaultSensitivity: permission.SensitivityConfidential,
	}
}

func securityAuditLogResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceSecurityAuditLog.String(),
		Feature:     platformcatalog.FeatureAdministration,
		DisplayName: "Security Audit Log",
		Description: "Security entries in the audit log (access and administrative changes, refused requests) and the alerts raised from them",
		Category:    "Users & security",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "Read security entries, check the audit log's seals and receive security alerts",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Manage alerts",
				Description: "Turn security alerts on or off and set their thresholds",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func auditLegalHoldResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceAuditLegalHold.String(),
		Feature:     platformcatalog.FeatureIdentity,
		DisplayName: "Audit Legal Hold",
		Description: "Holds that keep audit entries past their retention for litigation or an investigation",
		Category:    "Users & security",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "See the organization's legal holds",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Place",
				Description: "Place a legal hold",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Release",
				Description: "Release a legal hold",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func tableConfigurationResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceTableConfiguration.String(),
		Feature:     platformcatalog.FeatureAdministration,
		DisplayName: "Table Configuration",
		Description: "Saved table views. Anyone may keep private views; sharing one with the organization needs these",
		Category:    "Organization settings",
		Operations: []permissionspec.OperationDefinition{
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
	}
}

func customFieldDefinitionResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceCustomFieldDefinition.String(),
		Feature:     platformcatalog.FeatureAdministration,
		DisplayName: "Custom Field Definition",
		Description: "Custom field definitions and resource assignments",
		Category:    "Organization settings",
		Operations: []permissionspec.OperationDefinition{
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
	}
}

func sequenceConfigResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceSequenceConfig.String(),
		Feature:            platformcatalog.FeatureAdministration,
		DisplayName:        "Sequence Configuration",
		Description:        "Sequence generator configuration",
		Category:           "Organization settings",
		Operations:         permissionspec.StandardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func integrationResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceIntegration.String(),
		Feature:            platformcatalog.FeatureAdministration,
		DisplayName:        "Integration",
		Description:        "External integration configuration and synchronization",
		Category:           "Organization settings",
		Operations:         permissionspec.StandardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func emailProfileResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceEmailProfile.String(),
		Feature:            platformcatalog.FeatureAdministration,
		DisplayName:        "Email Profile",
		Description:        "Transactional email sender profiles and assignments",
		Category:           "Organization settings",
		ReferenceView:      []string{"name"},
		Operations:         permissionspec.StandardOpsWithDelete,
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
	}
}

func emailLogResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceEmailLog.String(),
		Feature:            platformcatalog.FeatureAdministration,
		DisplayName:        "Email Log",
		Description:        "Transactional email send and delivery logs",
		Category:           "Organization settings",
		ReferenceView:      []string{"status", "sentAt", "failedAt", "providerMessageId"},
		Operations:         permissionspec.ReadOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	}
}

func emailSuppressionResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceEmailSuppression.String(),
		Feature:            platformcatalog.FeatureAdministration,
		DisplayName:        "Email Suppression",
		Description:        "Email bounce, complaint, and manual suppression records",
		Category:           "Organization settings",
		Operations:         permissionspec.StandardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityConfidential,
	}
}

func ediResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceEDI.String(),
		Feature:     platformcatalog.FeatureEDIIntegration,
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
		Operations: permissionspec.Relabel(
			permissionspec.StandardOpsWithDelete,
			permission.OpUpdate,
			"Update",
			"Modify EDI settings, and apply, approve, reject, cancel, activate and archive EDI changes, transfers, connections and template versions",
		),
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: ediFieldSensitivities(),
	}
}

var ediFieldSensitivitiesTable = map[string]permission.FieldSensitivity{
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
}

func ediFieldSensitivities() map[string]permission.FieldSensitivity {
	return maps.Clone(ediFieldSensitivitiesTable)
}

func apiKeyResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceAPIKey.String(),
		Feature:            platformcatalog.FeatureAPIKeys,
		DisplayName:        "API Key",
		Description:        "API key creation, rotation, revocation, and permission management",
		Category:           "Users & security",
		Operations:         permissionspec.StandardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func platformCatalogResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourcePlatformCatalog.String(),
		Feature:            platformcatalog.FeatureAdministration,
		DisplayName:        "Platform Catalog",
		Description:        "Platform catalog and entitlement metadata administration",
		Category:           "Organization settings",
		Operations:         permissionspec.ReadOnlyOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func identityProviderResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceIdentityProvider.String(),
		Feature:            platformcatalog.FeatureIdentity,
		DisplayName:        "Identity Provider",
		Description:        "OIDC identity provider configuration",
		Category:           "Users & security",
		Operations:         permissionspec.StandardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func scimDirectoryResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceSCIMDirectory.String(),
		Feature:            platformcatalog.FeatureIdentity,
		DisplayName:        "SCIM Directory",
		Description:        "SCIM directory, token, and group mapping administration",
		Category:           "Users & security",
		Operations:         permissionspec.StandardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func provisioningAuditResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceProvisioningAudit.String(),
		Feature:            platformcatalog.FeatureIdentity,
		DisplayName:        "Provisioning Audit",
		Description:        "SCIM provisioning activity",
		Category:           "Users & security",
		Operations:         permissionspec.ReadOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	}
}

func accessPolicyResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceAccessPolicy.String(),
		Feature:            platformcatalog.FeatureIdentity,
		DisplayName:        "Access Policy",
		Description:        "Organization access policy decisions",
		Category:           "Users & security",
		Operations:         permissionspec.StandardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func authEventResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceAuthEvent.String(),
		Feature:            platformcatalog.FeatureAdministration,
		DisplayName:        "Authentication Event",
		Description:        "Authentication and sign-in activity",
		Category:           "Users & security",
		Operations:         permissionspec.ReadOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	}
}

func riskDecisionResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceRiskDecision.String(),
		Feature:            platformcatalog.FeatureIdentity,
		DisplayName:        "Risk Decision",
		Description:        "Authentication risk decisions",
		Category:           "Users & security",
		Operations:         permissionspec.ReadOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	}
}

func externalIdentityResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceExternalIdentity.String(),
		Feature:            platformcatalog.FeatureIdentity,
		DisplayName:        "External Identity",
		Description:        "Federated identity links",
		Category:           "Users & security",
		Operations:         permissionspec.ReadOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	}
}

func mfaAuthenticatorResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceMFAAuthenticator.String(),
		Feature:            platformcatalog.FeatureAdministration,
		DisplayName:        "MFA Authenticator",
		Description:        "MFA authenticator inventory",
		Category:           "Users & security",
		Operations:         permissionspec.ReadOnlyOps,
		DefaultSensitivity: permission.SensitivityConfidential,
	}
}

// Support access lets a named Trenova staff member into a managed instance for a set time. It
// is a person's decision: no API key or agent may hold it (neither runtimePolicy nor the agent
// ceiling lists it), and a support session may not use it (supportaccess.denied).
func supportAccessResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceSupportAccess.String(),
		Feature:     platformcatalog.FeatureAdministration,
		DisplayName: "Trenova Support Access",
		Description: "Let a named Trenova staff member into this organization for a set time",
		Category:    "Users & security",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View support access grants and what was done under each",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Grant",
				Description: "Give a Trenova staff member access for a set time",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Revoke",
				Description: "End a support access grant and its sessions at once",
			},
		},
		DefaultSensitivity: permission.SensitivityConfidential,
	}
}

func tableChangeAlertResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceTableChangeAlert.String(),
		Feature:     platformcatalog.FeatureTableChangeAlerts,
		DisplayName: "Table Change Alert",
		Description: "Table change alert subscriptions and notifications",
		Category:    "Organization settings",
		Operations: []permissionspec.OperationDefinition{
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
	}
}

func homeLayoutPresetResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceHomeLayoutPreset.String(),
		Feature:     platformcatalog.FeatureAdministration,
		DisplayName: "Home Screen Preset",
		Description: "Home screens administrators author and assign to roles",
		Category:    "Organization settings",
		Operations: []permissionspec.OperationDefinition{
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
	}
}

// Publishing is separated from editing because they carry different risk: a
// draft changes nothing a customer sees, and activating one changes every
// invoice and notice the organization sends from that moment on.
func documentTemplateResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceDocumentTemplate.String(),
		Feature:     platformcatalog.FeatureDocumentManagement,
		DisplayName: "Document & Message Template",
		Description: "Templates for customer documents, emails, and notifications",
		Category:    "Documents",
		Operations: permissionspec.Relabel(
			append(slices.Clone(permissionspec.StandardOpsWithDelete),
				permissionspec.OperationDefinition{
					Operation:   permission.OpActivate,
					DisplayName: "Publish",
					Description: "Publish a version, making it what customers receive",
				},
				permissionspec.OperationDefinition{
					Operation:   permission.OpArchive,
					DisplayName: "Archive",
					Description: "Retire a template version",
				},
				permissionspec.OperationDefinition{
					Operation:   permission.OpAssign,
					DisplayName: "Assign",
					Description: "Assign a template to a specific customer",
				},
				permissionspec.OperationDefinition{
					Operation:   permission.OpUnassign,
					DisplayName: "Unassign",
					Description: "Remove a customer's template override",
				},
			),
			permission.OpUpdate,
			"Update",
			"Modify templates and send test messages",
		),
		DefaultSensitivity: permission.SensitivityInternal,
	}
}
