package permission

import (
	"slices"
)

func (r *Registry) registerBillingResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceInvoiceRun.String(),
		DisplayName: "Invoice Run",
		Description: "Consolidated invoice batch preview and commit",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View invoice runs"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Build an invoice run preview",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Adjust invoice run membership",
			},
			// Issuing invoices to customers is a separable authority from building
			// a preview, so a four-eyes billing shop can grant one without the other.
			{
				Operation:   OpApprove,
				DisplayName: "Commit",
				Description: "Commit an invoice run and issue its invoices",
			},
			{Operation: OpCancel, DisplayName: "Cancel", Description: "Discard an invoice run"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceInvoiceDispute.String(),
		DisplayName: "Invoice Dispute",
		Description: "Dispute cases raised against posted invoices",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View invoice disputes"},
			{Operation: OpCreate, DisplayName: "Open", Description: "Open a dispute on an invoice"},
			{Operation: OpApprove, DisplayName: "Resolve", Description: "Resolve an open dispute"},
			{Operation: OpCancel, DisplayName: "Withdraw", Description: "Withdraw an open dispute"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceInvoice.String(),
		DisplayName: "Invoice",
		Description: "Invoice management",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View invoices"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create invoices"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify invoices"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export invoices"},
			{Operation: OpApprove, DisplayName: "Approve", Description: "Approve invoices"},
			{Operation: OpReject, DisplayName: "Reject", Description: "Reject invoices"},
			{
				Operation:   OpSubmit,
				DisplayName: "Submit",
				Description: "Submit invoices for approval",
			},
			{Operation: OpCancel, DisplayName: "Void", Description: "Void invoices"},
		},
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"id":                              SensitivityInternal,
			"businessUnitId":                  SensitivityInternal,
			"organizationId":                  SensitivityInternal,
			"number":                          SensitivityInternal,
			"status":                          SensitivityInternal,
			"type":                            SensitivityInternal,
			"billType":                        SensitivityInternal,
			"invoiceDate":                     SensitivityInternal,
			"dueDate":                         SensitivityInternal,
			"serviceDate":                     SensitivityInternal,
			"postedAt":                        SensitivityInternal,
			"sentAt":                          SensitivityInternal,
			"sentById":                        SensitivityInternal,
			"sendStatus":                      SensitivityInternal,
			"settlementStatus":                SensitivityInternal,
			"disputeStatus":                   SensitivityInternal,
			"currencyCode":                    SensitivityInternal,
			"paymentTerm":                     SensitivityInternal,
			"customerId":                      SensitivityInternal,
			"shipmentId":                      SensitivityInternal,
			"orderId":                         SensitivityInternal,
			"orderNumber":                     SensitivityInternal,
			"shipmentProNumber":               SensitivityInternal,
			"shipmentBol":                     SensitivityInternal,
			"invoiceId":                       SensitivityInternal,
			"lineNumber":                      SensitivityInternal,
			"description":                     SensitivityInternal,
			"billToName":                      SensitivityInternal,
			"billToCode":                      SensitivityInternal,
			"billToAddressLine1":              SensitivityInternal,
			"billToAddressLine2":              SensitivityInternal,
			"billToCity":                      SensitivityInternal,
			"billToState":                     SensitivityInternal,
			"billToCountry":                   SensitivityInternal,
			"billToPostalCode":                SensitivityInternal,
			"billingQueueItemId":              SensitivityInternal,
			"pdfDocumentId":                   SensitivityInternal,
			"correctionGroupId":               SensitivityInternal,
			"sourceInvoiceAdjustmentId":       SensitivityInternal,
			"supersededByInvoiceId":           SensitivityInternal,
			"supersedesInvoiceId":             SensitivityInternal,
			"isAdjustmentArtifact":            SensitivityInternal,
			"version":                         SensitivityInternal,
			"createdAt":                       SensitivityInternal,
			"updatedAt":                       SensitivityInternal,
			"amount":                          SensitivityRestricted,
			"amountMinor":                     SensitivityRestricted,
			"appliedAmount":                   SensitivityRestricted,
			"appliedAmountMinor":              SensitivityRestricted,
			"otherAmount":                     SensitivityRestricted,
			"otherAmountMinor":                SensitivityRestricted,
			"subtotalAmount":                  SensitivityRestricted,
			"subtotalAmountMinor":             SensitivityRestricted,
			"totalAmount":                     SensitivityRestricted,
			"totalAmountMinor":                SensitivityRestricted,
			"quantity":                        SensitivityRestricted,
			"unitPrice":                       SensitivityRestricted,
			"memo":                            SensitivityRestricted,
			"remittanceInstructions":          SensitivityRestricted,
			"emailToSnapshot":                 SensitivityRestricted,
			"emailCcSnapshot":                 SensitivityRestricted,
			"emailBccSnapshot":                SensitivityRestricted,
			"emailSubjectSnapshot":            SensitivityRestricted,
			"emailBodySnapshot":               SensitivityRestricted,
			"lastSendError":                   SensitivityRestricted,
			"lastSendWarning":                 SensitivityRestricted,
			"accountingDate":                  SensitivityInternal,
			"adjustmentId":                    SensitivityInternal,
			"approvalRequired":                SensitivityInternal,
			"approvalStatus":                  SensitivityInternal,
			"approvedAt":                      SensitivityInternal,
			"approvedById":                    SensitivityInternal,
			"batchId":                         SensitivityInternal,
			"creditMemoInvoiceId":             SensitivityInternal,
			"creditMemoLineId":                SensitivityInternal,
			"kind":                            SensitivityInternal,
			"originalInvoiceId":               SensitivityInternal,
			"originalLineId":                  SensitivityInternal,
			"rebillQueueItemId":               SensitivityInternal,
			"rebillStrategy":                  SensitivityInternal,
			"rejectedAt":                      SensitivityInternal,
			"rejectedById":                    SensitivityInternal,
			"replacementInvoiceId":            SensitivityInternal,
			"replacementLineId":               SensitivityInternal,
			"replacementReviewStatus":         SensitivityInternal,
			"requiresReconciliationException": SensitivityInternal,
			"submittedAt":                     SensitivityInternal,
			"submittedById":                   SensitivityInternal,
			"wouldCreateUnappliedCredit":      SensitivityInternal,
			"creditAmount":                    SensitivityRestricted,
			"creditAmountMinor":               SensitivityRestricted,
			"creditQuantity":                  SensitivityRestricted,
			"creditTotalAmount":               SensitivityRestricted,
			"creditTotalAmountMinor":          SensitivityRestricted,
			"netDeltaAmount":                  SensitivityRestricted,
			"netDeltaAmountMinor":             SensitivityRestricted,
			"policyReason":                    SensitivityRestricted,
			"reason":                          SensitivityRestricted,
			"rebillAmount":                    SensitivityRestricted,
			"rebillAmountMinor":               SensitivityRestricted,
			"rebillQuantity":                  SensitivityRestricted,
			"rebillTotalAmount":               SensitivityRestricted,
			"rebillTotalAmountMinor":          SensitivityRestricted,
			"rejectionReason":                 SensitivityRestricted,
			"remainingEligibleAmount":         SensitivityRestricted,
			"rerateVariancePercent":           SensitivityRestricted,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceBillingQueue.String(),
		DisplayName: "Billing Queue",
		Description: "Billing queue review and approval",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View billing queue items"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Transfer shipments to billing queue",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Update billing queue item status",
			},
			{
				Operation:   OpAssign,
				DisplayName: "Assign",
				Description: "Assign billers to queue items",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"id":                        SensitivityInternal,
			"businessUnitId":            SensitivityInternal,
			"organizationId":            SensitivityInternal,
			"shipmentId":                SensitivityInternal,
			"orderId":                   SensitivityInternal,
			"billToCustomerId":          SensitivityInternal,
			"assignedBillerId":          SensitivityInternal,
			"number":                    SensitivityInternal,
			"status":                    SensitivityInternal,
			"billType":                  SensitivityInternal,
			"exceptionReasonCode":       SensitivityInternal,
			"reviewNotes":               SensitivityInternal,
			"exceptionNotes":            SensitivityInternal,
			"reviewStartedAt":           SensitivityInternal,
			"reviewCompletedAt":         SensitivityInternal,
			"canceledById":              SensitivityInternal,
			"canceledAt":                SensitivityInternal,
			"cancelReason":              SensitivityInternal,
			"invoiceId":                 SensitivityInternal,
			"requiresReplacementReview": SensitivityInternal,
			"createdAt":                 SensitivityInternal,
			"updatedAt":                 SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentRun.String(),
		DisplayName: "Agent Run",
		Description: "Billing exception agent runs",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View agent runs"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Start agent runs"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentProposal.String(),
		DisplayName: "Agent Proposal",
		Description: "Agent-proposed resolutions awaiting human decision",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View agent proposals"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Accept, modify, or reject agent proposals",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentException.String(),
		DisplayName: "Agent Exception",
		Description: "Agent exceptions raised for human review",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View agent exceptions"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Raise agent exceptions"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Resolve agent exceptions",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentControl.String(),
		DisplayName: "Agent Control",
		Description: "Per-organization agent autonomy configuration",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View agent control settings"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Update agent control settings",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	// Configuring an agent decides which tools it may reach and how much it may do
	// What the organization has told its agents. Reading it shows what every
	// agent is told; writing it changes how every agent behaves, so it sits
	// with the agent definitions.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentMemory.String(),
		DisplayName: "Agent Memory",
		Description: "Standing instructions, facts and corrections kept for agents between runs",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View what agents remember"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Record a memory for agents"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Change, retire or restore a memory",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentEvalSuite.String(),
		DisplayName: "Agent Evaluation Suite",
		Description: "Evaluation cases agents are replayed against and scored on",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View evaluation cases and scores",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Capture or write an evaluation case and replay it",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Edit what a case expects, and activate, quarantine or retire it",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentFeedback.String(),
		DisplayName: "Agent Feedback",
		Description: "Everyone's ratings of AI output, with what they saw and why they rated it",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View everyone's ratings of AI output and each agent's satisfaction",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	// The sealed record of what every agent did and on whose behalf: each
	// model call, tool call, proposal, decision and write. It names people,
	// records and arguments across the whole organization, so it is kept for
	// those who answer for the organization's agents; reading an agent's runs
	// does not grant it, and no agent can hold it.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAIAuditTrail.String(),
		DisplayName: "AI Audit Trail",
		Description: "The tamper-evident record of what agents did, for whom, and who decided",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View the AI audit trail and verify its chain",
			},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export the AI audit trail and download exports",
			},
		},
		DefaultSensitivity: SensitivityConfidential,
	})

	// unattended, so it sits at the same sensitivity as provider configuration.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentDefinition.String(),
		DisplayName: "Agent Definition",
		Description: "Per-organization agent configurations and their tool access",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View configured agents"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add an agent"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Update an agent's tools and autonomy",
			},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Remove an agent"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	// Using the assistant is an ordinary operational act rather than an
	// administrative one, so it is held separately from configuring agents: a
	// dispatcher should be able to ask questions without being able to change
	// which tools an agent holds.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAssistant.String(),
		DisplayName: "Assistant",
		Description: "Conversations with configured agents",
		Category:    "Platform",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View own conversations"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Start conversations and send messages",
			},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Delete own conversations"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	// An insight names a customer, a location or a driver alongside a figure, so
	// reading the panel is a real disclosure. It is held separately from the
	// records it draws on because a reader still needs permission over those:
	// the panel filters to what its reader may already see, and this permission
	// only decides whether they get a panel at all.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceInsight.String(),
		DisplayName: "Insight",
		Description: "Operational findings surfaced on the home screen",
		Category:    "Platform",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View operational insights"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Dismiss an insight that is not worth acting on",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	// The watchtower is a feed over records a reader may already be allowed to
	// see: every item is filtered again by the permission on the source it came
	// from, so this one decides only whether a person gets the feed at all.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceWatchtower.String(),
		DisplayName: "Watchtower",
		Description: "The live feed of what needs attention across the operation",
		Category:    "Platform",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View the watchtower feed"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Mark the feed seen, dismiss an item, or hand one to an agent",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	// A briefing gathers the morning's figures across dispatch, billing and
	// compliance into one page, which is a wider disclosure than any single
	// screen it draws on.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceBriefing.String(),
		DisplayName: "Briefing",
		Description: "The daily briefing written for a role from the day's figures",
		Category:    "Platform",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "Read the daily briefing"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Write today's briefing again from current figures",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	// Configuring a provider decides which endpoint an organization's freight and
	// billing data is sent to, and whether that endpoint may be on the local
	// network, so it is held at the same sensitivity as credential management.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentExtension.String(),
		DisplayName: "Agent Extension",
		Description: "Capabilities turned on for agents only, such as web research, and their credentials",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View the extension marketplace"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Turn extensions on or off, change their settings and test them",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceWebResearch.String(),
		DisplayName: "Web Research",
		Description: "Agents searching and reading public web pages on a person's behalf, through an enabled extension",
		Category:    "Platform",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "Let agents search and read the web for this person",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAIProvider.String(),
		DisplayName: "AI Provider",
		Description: "Model endpoints, including self-hosted servers, and their task routing",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View AI providers"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add an AI provider"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Update an AI provider and its task routing",
			},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Remove an AI provider"},
			{
				Operation:   OpManage,
				DisplayName: "Manage",
				Description: "Test an AI provider connection",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAccessorialCharge.String(),
		DisplayName:        "Accessorial Charge",
		Description:        "Accessorial charge definitions",
		Category:           "Billing",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceChargeType.String(),
		DisplayName:        "Charge Type",
		Description:        "Charge type definitions",
		Category:           "Billing",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceRevenueCode.String(),
		DisplayName:        "Revenue Code",
		Description:        "Revenue code definitions",
		Category:           "Billing",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceFormulaTemplate.String(),
		DisplayName: "Formula Template",
		Description: "Rate formula templates",
		Category:    "Billing",
		Operations: append(slices.Clone(standardOps),
			OperationDefinition{
				Operation:   OpDuplicate,
				DisplayName: "Duplicate",
				Description: "Duplicate formula templates",
			},
			OperationDefinition{
				Operation:   OpSubmit,
				DisplayName: "Submit",
				Description: "Submit formula templates for review",
			},
			OperationDefinition{
				Operation:   OpApprove,
				DisplayName: "Approve",
				Description: "Approve formula templates and schedule rate changes",
			},
			OperationDefinition{
				Operation:   OpReject,
				DisplayName: "Reject",
				Description: "Reject formula templates under review",
			},
		),
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceFuelSurchargeProgram.String(),
		DisplayName:        "Fuel Surcharge Program",
		Description:        "Fuel surcharge programs, fuel price indices, and DOE weekly price data",
		Category:           "Billing",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDetentionPolicy.String(),
		DisplayName:        "Detention Policy",
		Description:        "Contractual detention terms, rate tiers, notice requirements, and per-stop occurrences",
		Category:           "Billing",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityInternal,
	})

	r.registerRateResources()
}
