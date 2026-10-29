package carrierperms

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func Resources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		carrierResource(),
		carrierCapacityPostingResource(),
		rateConfirmationResource(),
		carrierSettlementResource(),
		carrierSettlementControlResource(),
		carrierInvoiceMatchResource(),
		routingGuideResource(),
		tenderResource(),
		carrierIntelligenceResource(),
		carrierSourcingResource(),
		equipmentVerificationResource(),
	}
}

func carrierResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourceCarrier.String(),
		DisplayName:   "Carrier",
		Description:   "External carrier management for brokered freight",
		Category:      "Carriers",
		ReferenceView: []string{"code", "name", "scac", "mcNumber", "dotNumber", "status"},
		Operations: []permissionspec.OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View carriers"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create carriers",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify carriers",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export carrier data",
			},
			{
				Operation:   permission.OpImport,
				DisplayName: "Import",
				Description: "Import carriers",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func carrierCapacityPostingResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceCarrierCapacityPosting.String(),
		DisplayName:        "Carrier Capacity Posting",
		Description:        "Trucks carriers say they have available, with lane, window and rate",
		Category:           "Carriers",
		ParentResource:     permission.ResourceCarrier.String(),
		Operations:         slices.Clone(permissionspec.StandardOpsWithDelete),
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func rateConfirmationResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceRateConfirmation.String(),
		DisplayName:    "Rate Confirmation",
		Description:    "Outbound carrier rate confirmations",
		Category:       "Carriers",
		ParentResource: permission.ResourceCarrier.String(),
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View rate confirmations",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Generate rate confirmations",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Send, confirm, and void rate confirmations",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func carrierSettlementResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourceCarrierSettlement.String(),
		DisplayName:   "Carrier Settlement",
		Description:   "Carrier settlement statements, batches, and cost events",
		Category:      "Carrier settlements",
		ReferenceView: []string{"settlementNumber", "status", "carrierId"},
		Operations: append(slices.Clone(permissionspec.StandardOps),
			permissionspec.OperationDefinition{
				Operation:   permission.OpSubmit,
				DisplayName: "Submit",
				Description: "Submit carrier settlements for approval",
			},
			permissionspec.OperationDefinition{
				Operation:   permission.OpApprove,
				DisplayName: "Approve",
				Description: "Approve and post carrier settlements",
			},
			permissionspec.OperationDefinition{
				Operation:   permission.OpReject,
				DisplayName: "Reject",
				Description: "Send carrier settlements back to draft",
			},
			permissionspec.OperationDefinition{
				Operation:   permission.OpCancel,
				DisplayName: "Void",
				Description: "Void carrier settlements",
			},
		),
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":                   permission.SensitivityInternal,
			"businessUnitId":       permission.SensitivityInternal,
			"organizationId":       permission.SensitivityInternal,
			"carrierId":            permission.SensitivityInternal,
			"carrierAssignmentId":  permission.SensitivityInternal,
			"batchId":              permission.SensitivityInternal,
			"settlementId":         permission.SensitivityInternal,
			"shipmentId":           permission.SensitivityInternal,
			"moveId":               permission.SensitivityInternal,
			"settlementNumber":     permission.SensitivityInternal,
			"status":               permission.SensitivityInternal,
			"eventType":            permission.SensitivityInternal,
			"lineNumber":           permission.SensitivityInternal,
			"costEventId":          permission.SensitivityInternal,
			"glAccountId":          permission.SensitivityInternal,
			"idempotencyKey":       permission.SensitivityInternal,
			"eventDate":            permission.SensitivityInternal,
			"periodStart":          permission.SensitivityInternal,
			"periodEnd":            permission.SensitivityInternal,
			"payDate":              permission.SensitivityInternal,
			"grossCostMinor":       permission.SensitivityRestricted,
			"adjustmentsMinor":     permission.SensitivityRestricted,
			"netPayableMinor":      permission.SensitivityRestricted,
			"amountMinor":          permission.SensitivityRestricted,
			"shipmentCount":        permission.SensitivityInternal,
			"currencyCode":         permission.SensitivityInternal,
			"description":          permission.SensitivityInternal,
			"proNumber":            permission.SensitivityInternal,
			"assignmentVersion":    permission.SensitivityInternal,
			"notes":                permission.SensitivityInternal,
			"submittedById":        permission.SensitivityInternal,
			"submittedAt":          permission.SensitivityInternal,
			"approvedById":         permission.SensitivityInternal,
			"approvedAt":           permission.SensitivityInternal,
			"postedById":           permission.SensitivityInternal,
			"postedAt":             permission.SensitivityInternal,
			"postedJournalBatchId": permission.SensitivityInternal,
			"paidAt":               permission.SensitivityInternal,
			"paidById":             permission.SensitivityInternal,
			"paymentMethod":        permission.SensitivityRestricted,
			"paymentReference":     permission.SensitivityRestricted,
			"paidJournalBatchId":   permission.SensitivityInternal,
			"voidedById":           permission.SensitivityInternal,
			"voidedAt":             permission.SensitivityInternal,
			"voidReason":           permission.SensitivityInternal,
			"voidJournalBatchId":   permission.SensitivityInternal,
			"version":              permission.SensitivityInternal,
			"createdAt":            permission.SensitivityInternal,
			"updatedAt":            permission.SensitivityInternal,
			"carrier":              permission.SensitivityInternal,
			"lines":                permission.SensitivityInternal,
		},
	}
}

func carrierSettlementControlResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceCarrierSettlementControl.String(),
		DisplayName:        "Carrier Settlement Control",
		Description:        "Carrier settlement pay period, trigger, and GL default configuration",
		Category:           "Carrier settlements",
		ParentResource:     permission.ResourceCarrierSettlement.String(),
		Operations:         permissionspec.StandardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func carrierInvoiceMatchResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourceCarrierInvoiceMatch.String(),
		DisplayName:   "Carrier Invoice Match",
		Description:   "Carrier invoice matching against negotiated buy rates",
		Category:      "Carrier settlements",
		ReferenceView: []string{"invoiceNumber", "status"},
		Operations: append(slices.Clone(permissionspec.StandardOps),
			permissionspec.OperationDefinition{
				Operation:   permission.OpApprove,
				DisplayName: "Resolve",
				Description: "Accept or accept-with-variance carrier invoice matches",
			},
			permissionspec.OperationDefinition{
				Operation:   permission.OpReject,
				DisplayName: "Reject",
				Description: "Reject carrier invoice matches",
			},
		),
		ParentResource:     permission.ResourceCarrierSettlement.String(),
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"billToName":             permission.SensitivityInternal,
			"bol":                    permission.SensitivityInternal,
			"businessUnitId":         permission.SensitivityInternal,
			"carrierId":              permission.SensitivityInternal,
			"createdAt":              permission.SensitivityInternal,
			"currencyCode":           permission.SensitivityInternal,
			"deliveryDate":           permission.SensitivityInternal,
			"duplicateOfId":          permission.SensitivityInternal,
			"expectedAmount":         permission.SensitivityRestricted,
			"invoiceDate":            permission.SensitivityInternal,
			"invoiceNumber":          permission.SensitivityInternal,
			"organizationId":         permission.SensitivityInternal,
			"proNumber":              permission.SensitivityInternal,
			"reconciliationNotes":    permission.SensitivityInternal,
			"reconciliationStatus":   permission.SensitivityInternal,
			"shipmentId":             permission.SensitivityInternal,
			"shipmentReference":      permission.SensitivityInternal,
			"suggestedCarrier":       permission.SensitivityInternal,
			"totalAmount":            permission.SensitivityRestricted,
			"updatedAt":              permission.SensitivityInternal,
			"varianceAmount":         permission.SensitivityRestricted,
			"version":                permission.SensitivityInternal,
			"ediCarrierInvoiceId":    permission.SensitivityInternal,
			"documentAiExtractionId": permission.SensitivityInternal,
			"carrierAssignmentId":    permission.SensitivityInternal,
			"carrierSettlementId":    permission.SensitivityInternal,
			"adjustmentCostEventId":  permission.SensitivityInternal,
			"duplicateOfMatchId":     permission.SensitivityInternal,
			"possibleDuplicateOfId":  permission.SensitivityInternal,
			"status":                 permission.SensitivityInternal,
			"matchedVia":             permission.SensitivityInternal,
			"resolutionNote":         permission.SensitivityInternal,
			"resolvedById":           permission.SensitivityInternal,
			"resolvedAt":             permission.SensitivityInternal,
			"carrier":                permission.SensitivityInternal,
			"carrierAssignment":      permission.SensitivityInternal,
			"invoiceTotalMinor":      permission.SensitivityRestricted,
			"expectedTotalMinor":     permission.SensitivityRestricted,
			"varianceMinor":          permission.SensitivityRestricted,
		},
	}
}

func routingGuideResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourceRoutingGuide.String(),
		DisplayName:   "Routing Guide",
		Description:   "Ranked carrier lists per freight lane that drive waterfall tenders",
		Category:      "Carriers",
		ReferenceView: []string{"name"},
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View routing guides",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Create routing guides",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify routing guides",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Delete routing guides",
			},
		},
		ParentResource:     permission.ResourceCarrier.String(),
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func tenderResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceTender.String(),
		DisplayName: "Carrier Tender",
		Description: "Waterfall and spot tender offers made to carriers for brokered moves",
		Category:    "Carriers",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View tenders and offers",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Start waterfall and spot tenders",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Record carrier responses, bounces and counter-offer decisions",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: "Cancel",
				Description: "Withdraw outstanding tenders",
			},
			{
				Operation:   permission.OpApprove,
				DisplayName: "Approve",
				Description: "Take carrier counter-offers that fall below the margin floor",
			},
		},
		ParentResource:     permission.ResourceCarrier.String(),
		DefaultSensitivity: permission.SensitivityInternal,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"counterMargin":        permission.SensitivityConfidential,
			"counterMarginPercent": permission.SensitivityConfidential,
		},
	}
}

func carrierIntelligenceResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceCarrierIntelligence.String(),
		DisplayName: "Carrier Intelligence",
		Description: "Vetting, monitoring and compliance findings from carrier intelligence providers",
		Category:    "Carriers",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View carrier intelligence, findings and monitoring events",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Vet and refresh carriers, acknowledge and resolve events, and apply suggested updates",
			},
			{
				Operation:   permission.OpApprove,
				DisplayName: "Approve",
				Description: "Grant and revoke overrides of blocking findings",
			},
			{
				Operation:   permission.OpManage,
				DisplayName: "Manage",
				Description: "Configure rules, monitoring enrollment, providers and spend caps",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Download raw provider payloads",
			},
		},
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"rawPayload": permission.SensitivityConfidential,
			"contacts":   permission.SensitivityRestricted,
			"network":    permission.SensitivityRestricted,
			"equipment":  permission.SensitivityRestricted,
		},
		ParentResource:     permission.ResourceCarrier.String(),
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func carrierSourcingResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceCarrierSourcing.String(),
		DisplayName: "Carrier Sourcing",
		Description: "Search carrier intelligence providers for new capacity and import prospects",
		Category:    "Carriers",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "Search for carriers",
			},
			{
				Operation:   permission.OpImport,
				DisplayName: "Import",
				Description: "Import sourced carriers as pending carriers",
			},
		},
		ParentResource:     permission.ResourceCarrier.String(),
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func equipmentVerificationResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceEquipmentVerification.String(),
		DisplayName: "Equipment Verification",
		Description: "Driver-at-dock checks that arriving equipment belongs to the assigned carrier",
		Category:    "Carriers",
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View equipment verifications",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Verify a VIN, plate or unit number",
			},
			{
				Operation:   permission.OpApprove,
				DisplayName: "Approve",
				Description: "Override a failed equipment verification",
			},
		},
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"vin":         permission.SensitivityRestricted,
			"plateNumber": permission.SensitivityRestricted,
		},
		ParentResource:     permission.ResourceCarrier.String(),
		DefaultSensitivity: permission.SensitivityInternal,
	}
}
