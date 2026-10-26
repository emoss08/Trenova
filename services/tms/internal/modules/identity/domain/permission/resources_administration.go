package permission

import (
	"slices"
)

func (r *Registry) registerAdministrationResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceOrganization.String(),
		DisplayName:        "Organization",
		Description:        "Organization settings and configuration",
		Category:           "Organization settings",
		ReferenceView:      []string{"name", "logoUrl"},
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceBusinessUnit.String(),
		DisplayName:        "Business Unit",
		Description:        "Business unit management",
		Category:           "Organization settings",
		ReferenceView:      []string{"name", "code"},
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      ResourceUser.String(),
		DisplayName:   "User",
		Description:   "User account management",
		Category:      "Users & security",
		ReferenceView: []string{"name", "username", "profilePicUrl", "thumbnailUrl"},
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View user accounts"},
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
		Resource:      ResourceRole.String(),
		DisplayName:   "Role",
		Description:   "Role and permission management",
		Category:      "Users & security",
		ReferenceView: []string{"name", "isSystem"},
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View roles"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create new roles"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify roles"},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Delete roles"},
		},
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAuditLog.String(),
		DisplayName:        "Audit Log",
		Description:        "System audit logs",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceTableConfiguration.String(),
		DisplayName: "Table Configuration",
		Description: "Saved table views. Anyone may keep private views; sharing one with the organization needs these",
		Category:    "Organization settings",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "See saved views on the home page",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Share",
				Description: "Share a view with the organization",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Change shared",
				Description: "Change a view shared with the organization",
			},
		},
		DefaultSensitivity: SensitivityPublic,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCustomFieldDefinition.String(),
		DisplayName: "Custom Field Definition",
		Description: "Custom field definitions and resource assignments",
		Category:    "Organization settings",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View custom field definitions"},
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
		Category:           "Organization settings",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceIntegration.String(),
		DisplayName:        "Integration",
		Description:        "External integration configuration and synchronization",
		Category:           "Organization settings",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceEmailProfile.String(),
		DisplayName:        "Email Profile",
		Description:        "Transactional email sender profiles and assignments",
		Category:           "Organization settings",
		ReferenceView:      []string{"name"},
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityRestricted,
		// Who mail comes from and where replies go is printed on every message
		// sent, so it is Internal: it is how an agent chooses a profile. How
		// the profile connects to its provider stays Restricted.
		FieldSensitivities: map[string]FieldSensitivity{
			"id":             SensitivityInternal,
			"businessUnitId": SensitivityInternal,
			"organizationId": SensitivityInternal,
			"name":           SensitivityInternal,
			"description":    SensitivityInternal,
			"senderName":     SensitivityInternal,
			"senderEmail":    SensitivityInternal,
			"replyToEmail":   SensitivityInternal,
			"status":         SensitivityInternal,
			"version":        SensitivityInternal,
			"createdAt":      SensitivityInternal,
			"updatedAt":      SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceEmailLog.String(),
		DisplayName:        "Email Log",
		Description:        "Transactional email send and delivery logs",
		Category:           "Organization settings",
		ReferenceView:      []string{"status", "sentAt", "failedAt", "providerMessageId"},
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceEmailSuppression.String(),
		DisplayName:        "Email Suppression",
		Description:        "Email bounce, complaint, and manual suppression records",
		Category:           "Organization settings",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceEDI.String(),
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
			OpUpdate,
			"Update",
			"Modify EDI settings, and apply, approve, reject, cancel, activate and archive EDI changes, transfers, connections and template versions",
		),
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"acceptedAt":               SensitivityInternal,
			"ackLastError":             SensitivityInternal,
			"ackMessageId":             SensitivityInternal,
			"ackReceivedAt":            SensitivityInternal,
			"ackStatus":                SensitivityInternal,
			"ackStatusCounts":          SensitivityInternal,
			"address":                  SensitivityInternal,
			"answersEverySet":          SensitivityInternal,
			"asOf":                     SensitivityInternal,
			"bol":                      SensitivityInternal,
			"businessUnitId":           SensitivityInternal,
			"canAccept":                SensitivityInternal,
			"canDecide":                SensitivityInternal,
			"capabilities":             SensitivityInternal,
			"charges":                  SensitivityRestricted,
			"checksum":                 SensitivityInternal,
			"city":                     SensitivityInternal,
			"code":                     SensitivityInternal,
			"commodities":              SensitivityInternal,
			"commodity":                SensitivityInternal,
			"communicationProfileId":   SensitivityInternal,
			"communicationProfileName": SensitivityInternal,
			"config":                   SensitivityConfidential,
			"connection":               SensitivityInternal,
			"contactEmail":             SensitivityRestricted,
			"contactName":              SensitivityInternal,
			"contactPhone":             SensitivityRestricted,
			"count":                    SensitivityInternal,
			"counterparty":             SensitivityInternal,
			"country":                  SensitivityInternal,
			"createdAt":                SensitivityInternal,
			"customerId":               SensitivityInternal,
			"customerLabel":            SensitivityInternal,
			"decoded":                  SensitivityInternal,
			"defaultMappingProfileId":  SensitivityInternal,
			"defaultTransport":         SensitivityInternal,
			"defaultTransportId":       SensitivityInternal,
			"deliveryAttempts":         SensitivityInternal,
			"deliveryLastAttemptAt":    SensitivityInternal,
			"deliveryLastError":        SensitivityInternal,
			"deliveryRemotePath":       SensitivityInternal,
			"deliverySentAt":           SensitivityInternal,
			"deliveryStatus":           SensitivityInternal,
			"description":              SensitivityInternal,
			"diagnosticCount":          SensitivityInternal,
			"diagnostics":              SensitivityInternal,
			"direction":                SensitivityInternal,
			"documentTypeId":           SensitivityInternal,
			"ediConnectionId":          SensitivityInternal,
			"ediPartnerId":             SensitivityInternal,
			"elementPosition":          SensitivityInternal,
			"enabledForInbound":        SensitivityInternal,
			"enabledForOutbound":       SensitivityInternal,
			"entityType":               SensitivityInternal,
			"failureReason":            SensitivityInternal,
			"file":                     SensitivityInternal,
			"fileName":                 SensitivityInternal,
			"generatedAt":              SensitivityInternal,
			"groupControlNumber":       SensitivityInternal,
			"inboundFileId":            SensitivityInternal,
			"inboundMessageId":         SensitivityInternal,
			"interchangeControlNumber": SensitivityInternal,
			"internalOrganization":     SensitivityInternal,
			"internalOrganizationId":   SensitivityInternal,
			"invoice":                  SensitivityInternal,
			"invoiceId":                SensitivityInternal,
			"isaReceiverId":            SensitivityInternal,
			"isaReceiverQualifier":     SensitivityInternal,
			"isaSenderId":              SensitivityInternal,
			"isaSenderQualifier":       SensitivityInternal,
			"kind":                     SensitivityInternal,
			"loadTenderInbound":        SensitivityInternal,
			"loadTenderOutbound":       SensitivityInternal,
			"location":                 SensitivityInternal,
			"mappingSnapshot":          SensitivityRestricted,
			"message":                  SensitivityInternal,
			"messageCount":             SensitivityInternal,
			"messages":                 SensitivityInternal,
			"method":                   SensitivityInternal,
			"name":                     SensitivityInternal,
			"oldestGeneratedAt":        SensitivityInternal,
			"organizationId":           SensitivityInternal,
			"overdueAfterSeconds":      SensitivityInternal,
			"overdueByPartner":         SensitivityInternal,
			"overdueCount":             SensitivityInternal,
			"overdueNotListed":         SensitivityInternal,
			"partner":                  SensitivityInternal,
			"partnerCode":              SensitivityInternal,
			"partnerDocumentProfileId": SensitivityInternal,
			"partnerId":                SensitivityInternal,
			"partnerName":              SensitivityInternal,
			"partners":                 SensitivityInternal,
			"path":                     SensitivityInternal,
			"pieces":                   SensitivityInternal,
			"postalCode":               SensitivityInternal,
			"processedAt":              SensitivityInternal,
			"raw":                      SensitivityRestricted,
			"rawContent":               SensitivityRestricted,
			"rawPurgedAt":              SensitivityInternal,
			"rawX12":                   SensitivityRestricted,
			"receivedAt":               SensitivityInternal,
			"recentDeadLettered":       SensitivityInternal,
			"rejectedAt":               SensitivityInternal,
			"rejectionReason":          SensitivityInternal,
			"remotePath":               SensitivityInternal,
			"requestedAt":              SensitivityInternal,
			"revokedAt":                SensitivityInternal,
			"secretState":              SensitivityConfidential,
			"segmentCount":             SensitivityInternal,
			"segmentId":                SensitivityInternal,
			"sequence":                 SensitivityInternal,
			"serviceTypeLabel":         SensitivityInternal,
			"severity":                 SensitivityInternal,
			"shipmentId":               SensitivityInternal,
			"shipmentStatus":           SensitivityInternal,
			"shipmentTypeLabel":        SensitivityInternal,
			"since":                    SensitivityInternal,
			"sizeBytes":                SensitivityInternal,
			"source":                   SensitivityInternal,
			"sourceBusinessUnitId":     SensitivityInternal,
			"sourceOrganizationId":     SensitivityInternal,
			"sourcePartner":            SensitivityInternal,
			"sourcePartnerId":          SensitivityInternal,
			"sourceShipmentId":         SensitivityInternal,
			"state":                    SensitivityInternal,
			"status":                   SensitivityInternal,
			"stopCount":                SensitivityInternal,
			"stops":                    SensitivityInternal,
			"submittedAt":              SensitivityInternal,
			"suggestedFix":             SensitivityInternal,
			"suspendedAt":              SensitivityInternal,
			"targetBusinessUnitId":     SensitivityInternal,
			"targetOrganizationId":     SensitivityInternal,
			"targetPartner":            SensitivityInternal,
			"targetPartnerId":          SensitivityInternal,
			"targetShipmentId":         SensitivityInternal,
			"temperatureMax":           SensitivityInternal,
			"temperatureMin":           SensitivityInternal,
			"templateId":               SensitivityInternal,
			"templateVersionId":        SensitivityInternal,
			"tenderPayload":            SensitivityRestricted,
			"timezone":                 SensitivityInternal,
			"transactionControlNumber": SensitivityInternal,
			"transactionCount":         SensitivityInternal,
			"transactionSet":           SensitivityInternal,
			"transfer":                 SensitivityInternal,
			"transferId":               SensitivityInternal,
			"type":                     SensitivityInternal,
			"unresolvedMappings":       SensitivityInternal,
			"updatedAt":                SensitivityInternal,
			"version":                  SensitivityInternal,
			"weight":                   SensitivityInternal,
			"windowCloses":             SensitivityInternal,
			"windowOpens":              SensitivityInternal,
			"x12Version":               SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAPIKey.String(),
		DisplayName:        "API Key",
		Description:        "API key creation, rotation, revocation, and permission management",
		Category:           "Users & security",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourcePlatformCatalog.String(),
		DisplayName:        "Platform Catalog",
		Description:        "Platform catalog and entitlement metadata administration",
		Category:           "Organization settings",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceIdentityProvider.String(),
		DisplayName:        "Identity Provider",
		Description:        "OIDC identity provider configuration",
		Category:           "Users & security",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceSCIMDirectory.String(),
		DisplayName:        "SCIM Directory",
		Description:        "SCIM directory, token, and group mapping administration",
		Category:           "Users & security",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceProvisioningAudit.String(),
		DisplayName:        "Provisioning Audit",
		Description:        "SCIM provisioning activity",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAccessPolicy.String(),
		DisplayName:        "Access Policy",
		Description:        "Organization access policy decisions",
		Category:           "Users & security",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAuthEvent.String(),
		DisplayName:        "Authentication Event",
		Description:        "Authentication and sign-in activity",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceRiskDecision.String(),
		DisplayName:        "Risk Decision",
		Description:        "Authentication risk decisions",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceExternalIdentity.String(),
		DisplayName:        "External Identity",
		Description:        "Federated identity links",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceMFAAuthenticator.String(),
		DisplayName:        "MFA Authenticator",
		Description:        "MFA authenticator inventory",
		Category:           "Users & security",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceTableChangeAlert.String(),
		DisplayName: "Table Change Alert",
		Description: "Table change alert subscriptions and notifications",
		Category:    "Organization settings",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
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
		FieldSensitivities: map[string]FieldSensitivity{
			"conditionMatch": SensitivityInternal,
			"eventTypes":     SensitivityInternal,
			"status":         SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceHomeLayoutPreset.String(),
		DisplayName: "Home Screen Preset",
		Description: "Home screens administrators author and assign to roles",
		Category:    "Organization settings",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
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
		Category:    "Documents",
		Operations: relabel(append(slices.Clone(standardOpsWithDelete),
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
		), OpUpdate, "Update", "Modify templates and send test messages"),
		DefaultSensitivity: SensitivityInternal,
	})
}
