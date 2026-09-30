package permission

import (
	"slices"
)

func (r *Registry) registerCarrierResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCarrier.String(),
		DisplayName: "Carrier",
		Description: "External carrier management for brokered freight",
		Category:    "Carriers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View carriers"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create carriers"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify carriers"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export carrier data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import carriers"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceRateConfirmation.String(),
		DisplayName:    "Rate Confirmation",
		Description:    "Outbound carrier rate confirmations",
		Category:       "Carriers",
		ParentResource: ResourceCarrier.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View rate confirmations"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Generate rate confirmations",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Send, confirm, and void rate confirmations",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCarrierSettlement.String(),
		DisplayName: "Carrier Settlement",
		Description: "Carrier settlement statements, batches, and cost events",
		Category:    "Carriers",
		Operations: append(slices.Clone(standardOps),
			OperationDefinition{
				Operation:   OpSubmit,
				DisplayName: "Submit",
				Description: "Submit carrier settlements for approval",
			},
			OperationDefinition{
				Operation:   OpApprove,
				DisplayName: "Approve",
				Description: "Approve and post carrier settlements",
			},
			OperationDefinition{
				Operation:   OpReject,
				DisplayName: "Reject",
				Description: "Send carrier settlements back to draft",
			},
			OperationDefinition{
				Operation:   OpCancel,
				DisplayName: "Void",
				Description: "Void carrier settlements",
			},
		),
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"id":                   SensitivityInternal,
			"businessUnitId":       SensitivityInternal,
			"organizationId":       SensitivityInternal,
			"carrierId":            SensitivityInternal,
			"carrierAssignmentId":  SensitivityInternal,
			"batchId":              SensitivityInternal,
			"settlementId":         SensitivityInternal,
			"shipmentId":           SensitivityInternal,
			"moveId":               SensitivityInternal,
			"settlementNumber":     SensitivityInternal,
			"status":               SensitivityInternal,
			"eventType":            SensitivityInternal,
			"idempotencyKey":       SensitivityInternal,
			"eventDate":            SensitivityInternal,
			"periodStart":          SensitivityInternal,
			"periodEnd":            SensitivityInternal,
			"payDate":              SensitivityInternal,
			"grossCostMinor":       SensitivityRestricted,
			"adjustmentsMinor":     SensitivityRestricted,
			"netPayableMinor":      SensitivityRestricted,
			"amountMinor":          SensitivityRestricted,
			"shipmentCount":        SensitivityInternal,
			"currencyCode":         SensitivityInternal,
			"description":          SensitivityInternal,
			"proNumber":            SensitivityInternal,
			"assignmentVersion":    SensitivityInternal,
			"notes":                SensitivityInternal,
			"submittedById":        SensitivityInternal,
			"submittedAt":          SensitivityInternal,
			"approvedById":         SensitivityInternal,
			"approvedAt":           SensitivityInternal,
			"postedById":           SensitivityInternal,
			"postedAt":             SensitivityInternal,
			"postedJournalBatchId": SensitivityInternal,
			"paidAt":               SensitivityInternal,
			"paidById":             SensitivityInternal,
			"paymentMethod":        SensitivityRestricted,
			"paymentReference":     SensitivityRestricted,
			"paidJournalBatchId":   SensitivityInternal,
			"voidedById":           SensitivityInternal,
			"voidedAt":             SensitivityInternal,
			"voidReason":           SensitivityInternal,
			"voidJournalBatchId":   SensitivityInternal,
			"version":              SensitivityInternal,
			"createdAt":            SensitivityInternal,
			"updatedAt":            SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceCarrierSettlementControl.String(),
		DisplayName:        "Carrier Settlement Control",
		Description:        "Carrier settlement pay period, trigger, and GL default configuration",
		Category:           "Carriers",
		ParentResource:     ResourceCarrierSettlement.String(),
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCarrierInvoiceMatch.String(),
		DisplayName: "Carrier Invoice Match",
		Description: "Carrier invoice matching against negotiated buy rates",
		Category:    "Carriers",
		Operations: append(slices.Clone(standardOps),
			OperationDefinition{
				Operation:   OpApprove,
				DisplayName: "Resolve",
				Description: "Accept or accept-with-variance carrier invoice matches",
			},
			OperationDefinition{
				Operation:   OpReject,
				DisplayName: "Reject",
				Description: "Reject carrier invoice matches",
			},
		),
		ParentResource:     ResourceCarrierSettlement.String(),
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceRoutingGuide.String(),
		DisplayName: "Routing Guide",
		Description: "Ranked carrier lists per freight lane that drive waterfall tenders",
		Category:    "Carriers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View routing guides"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create routing guides"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify routing guides"},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Delete routing guides"},
		},
		ParentResource:     ResourceCarrier.String(),
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceTender.String(),
		DisplayName: "Carrier Tender",
		Description: "Waterfall and spot tender offers made to carriers for brokered moves",
		Category:    "Carriers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View tenders and offers"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Start waterfall and spot tenders",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Record manual carrier responses",
			},
			{
				Operation:   OpCancel,
				DisplayName: "Cancel",
				Description: "Withdraw outstanding tenders",
			},
		},
		ParentResource:     ResourceCarrier.String(),
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCarrierIntelligence.String(),
		DisplayName: "Carrier Intelligence",
		Description: "Vetting, monitoring and compliance findings from carrier intelligence providers",
		Category:    "Carriers",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View carrier intelligence, findings and monitoring events",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Vet and refresh carriers, acknowledge events and apply suggested updates",
			},
			{
				Operation:   OpApprove,
				DisplayName: "Approve",
				Description: "Grant and revoke overrides of blocking findings",
			},
			{
				Operation:   OpManage,
				DisplayName: "Manage",
				Description: "Configure rules, monitoring enrollment, providers and spend caps",
			},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Download raw provider payloads",
			},
		},
		FieldSensitivities: map[string]FieldSensitivity{
			"rawPayload": SensitivityConfidential,
			"contacts":   SensitivityRestricted,
			"network":    SensitivityRestricted,
			"equipment":  SensitivityRestricted,
		},
		ParentResource:     ResourceCarrier.String(),
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCarrierSourcing.String(),
		DisplayName: "Carrier Sourcing",
		Description: "Search carrier intelligence providers for new capacity and import prospects",
		Category:    "Carriers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "Search for carriers"},
			{
				Operation:   OpImport,
				DisplayName: "Import",
				Description: "Import sourced carriers as pending carriers",
			},
		},
		ParentResource:     ResourceCarrier.String(),
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceEquipmentVerification.String(),
		DisplayName: "Equipment Verification",
		Description: "Driver-at-dock checks that arriving equipment belongs to the assigned carrier",
		Category:    "Carriers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View equipment verifications"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Verify a VIN, plate or unit number",
			},
			{
				Operation:   OpApprove,
				DisplayName: "Approve",
				Description: "Override a failed equipment verification",
			},
		},
		FieldSensitivities: map[string]FieldSensitivity{
			"vin":         SensitivityRestricted,
			"plateNumber": SensitivityRestricted,
		},
		ParentResource:     ResourceCarrier.String(),
		DefaultSensitivity: SensitivityInternal,
	})
}
