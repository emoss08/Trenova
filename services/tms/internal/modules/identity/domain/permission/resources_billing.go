package permission

import (
	"slices"
)

const displayDiscard = "Discard"

func (r *Registry) registerBillingResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceInvoiceRun.String(),
		DisplayName: "Invoice Run",
		Description: "Consolidated invoice batch preview and commit",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View invoice runs"},
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
		Resource:      ResourceInvoiceDispute.String(),
		DisplayName:   "Invoice Dispute",
		Description:   "Dispute cases raised against posted invoices",
		Category:      "Billing",
		ReferenceView: []string{"invoiceId", "status", "reasonCode", "openedAt", "resolvedAt"},
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View invoice disputes"},
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
		ReferenceView: []string{
			"number",
			"status",
			"invoiceDate",
			"dueDate",
			"billToName",
			"settlementStatus",
			"disputeStatus",
			"customerId",
		},
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View invoices"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create invoices"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update and post",
				Description: "Modify invoices, submit adjustments and post invoices to the ledger",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export invoices"},
			{
				Operation:   OpApprove,
				DisplayName: "Approve",
				Description: "Approve invoices, and approve or reject invoice adjustments",
			},
			{Operation: OpReject, DisplayName: "Reject", Description: "Reject invoices"},
			{
				Operation:   OpSubmit,
				DisplayName: "Submit and send",
				Description: "Submit invoices for approval, and send or resend them by email or EDI",
			},
			{Operation: OpCancel, DisplayName: "Void", Description: "Void invoices"},
			{
				Operation:   OpPost,
				DisplayName: "Post",
				Description: "Post invoices to the ledger, by hand or from the billing queue",
			},
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
			"verdict":                         SensitivityInternal,
			"failedChecks":                    SensitivityInternal,
			"readySince":                      SensitivityInternal,
			"evaluatedAt":                     SensitivityInternal,
			"checks":                          SensitivityInternal,
			"code":                            SensitivityInternal,
			"passed":                          SensitivityInternal,
			"message":                         SensitivityInternal,
			"fix":                             SensitivityInternal,
			"documentTypeId":                  SensitivityInternal,
			"documentTypeName":                SensitivityInternal,
			"shipmentPro":                     SensitivityInternal,
			"factoringCompanyId":              SensitivityInternal,
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
		Scopes: []ScopeSupport{
			{Scope: DataScopeOwn, EnforcedBy: ScopeEnforcedByDatabase},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      ResourceBillingQueue.String(),
		DisplayName:   "Billing Queue",
		Description:   "Billing queue review and approval",
		Category:      "Billing",
		ReferenceView: []string{"number", "status"},
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View billing queue items"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Transfer shipments to billing queue",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Update billing queue items: post them, release them and resolve their issues",
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
			"shipment.proNumber":        SensitivityInternal,
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
		Category:    "AI & agents",
		ReferenceView: []string{
			"agentDefinitionId",
			"subjectType",
			"subjectId",
			"trigger",
			"definition",
		},
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View agent runs"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Start agent runs"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      ResourceAgentProposal.String(),
		DisplayName:   "Decisions",
		Description:   "What the agents propose and ask a person to approve or decline",
		Category:      "Desk",
		ReferenceView: []string{"toolName", "status"},
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View agent proposals"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Accept, modify or reject agent proposals, and resolve proposals and plans",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentException.String(),
		DisplayName: "Agent Exception",
		Description: "Agent exceptions raised for human review",
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View agent exceptions"},
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
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "View agent control settings, and dismiss or restore provider failure notices",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Update agent control settings, and apply or restore tune-ups",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	// What the organization has told its agents. Reading it shows what every
	// agent is told. A person's own memories need only the assistant, and
	// anyone may offer one for their team or the organization; keeping,
	// changing, retiring and approving what reaches other people's
	// conversations is manage, which the role templates never hand out, so it
	// stays an administrator's unless a role is given it on purpose.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentMemory.String(),
		DisplayName: "Agent Memory",
		Description: "Standing instructions, facts, corrections and profile guidance kept for agents between runs",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View what agents remember"},
			{
				Operation:   OpManage,
				DisplayName: operationDefinitions[OpManage].DisplayName,
				Description: "Keep, change, retire, review and approve memories and profile guidance shared with a team, the organization or every agent",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentEvalSuite.String(),
		DisplayName: "Agent Evaluation Suite",
		Description: "Evaluation cases agents are replayed against and scored on",
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
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
				Description: "Edit what a case expects, activate, quarantine or retire it, and cancel runs",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAgentFeedback.String(),
		DisplayName: "Agent Feedback",
		Description: "Everyone's ratings of AI output, with what they saw and why they rated it",
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
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
		Category:    "Users & security",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
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
		Resource:      ResourceAgentDefinition.String(),
		DisplayName:   "Agent Definition",
		Description:   "Per-organization agent configurations and their tool access",
		Category:      "AI & agents",
		ReferenceView: []string{"name", "icon", "accent", "template"},
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View configured agents"},
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
		DisplayName: "Desk",
		Description: "Conversations with the agents and what they produce",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "View own conversations and pin artifacts in them",
			},
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
		Category:    "Reports",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View operational insights"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Dismiss an insight that is not worth acting on, or restore one",
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
		Category:    "Desk",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View the watchtower feed"},
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
		Category:    "Desk",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "Read the daily briefing"},
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
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View the extension marketplace"},
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
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "Let agents search and read the web for this person",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourcePortalAutomation.String(),
		DisplayName: "Portal Automation",
		Description: "Agents reading and acting on the third-party web portals the organization set up, through an enabled extension",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "Let agents look things up on portals for this person",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Act",
				Description: "Let agents propose booking, uploading or accepting on portals for this person",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDataAnalysis.String(),
		DisplayName: "Data Analysis",
		Description: "Agents running code over the report results and files a person gave them, through an enabled extension",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "Let agents analyze this person's report results and attached files",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCodeMode.String(),
		DisplayName: "Read-only scripts",
		Description: "Agents answering a question that needs many reads with one short read-only script, through an enabled extension",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "Let agents run read-only scripts over what this person may read",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceAIProvider.String(),
		DisplayName: "AI Provider",
		Description: "Model endpoints, including self-hosted servers, and their task routing",
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View AI providers"},
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
		ReferenceView:      []string{"code", "description", "status", "method", "rateUnit"},
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
		Resource:    ResourceFormulaLibrary.String(),
		DisplayName: "Formula Library",
		Description: "Organization formula libraries that rate formula templates import",
		Category:    "Billing",
		Operations: append(slices.Clone(standardOps),
			OperationDefinition{
				Operation:   OpDuplicate,
				DisplayName: "Duplicate",
				Description: "Duplicate formula libraries",
			},
			OperationDefinition{
				Operation:   OpSubmit,
				DisplayName: "Submit",
				Description: "Submit formula libraries for review",
			},
			OperationDefinition{
				Operation:   OpApprove,
				DisplayName: "Approve",
				Description: "Approve formula library versions templates can pin",
			},
			OperationDefinition{
				Operation:   OpReject,
				DisplayName: "Reject",
				Description: "Reject formula libraries under review",
			},
			OperationDefinition{
				Operation:   OpArchive,
				DisplayName: "Archive",
				Description: "Archive formula libraries so no new template imports them",
			},
		),
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceFuelSurchargeProgram.String(),
		DisplayName:        "Fuel Surcharge Program",
		Description:        "Fuel surcharge programs, fuel price indices, and DOE weekly price data",
		Category:           "Fuel & tax",
		ReferenceView:      []string{"code", "name", "status"},
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDetentionPolicy.String(),
		DisplayName: "Detention Policy",
		Description: "Contractual detention terms, rate tiers and notice requirements",
		Category:    "Detention",
		Operations: relabel(
			standardOpsWithDelete,
			OpUpdate,
			"Update",
			"Modify detention policies and the terms they apply",
		),
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDetention.String(),
		DisplayName: "Detention",
		Description: "Detention at stops: the detention desk, its occurrences and the intelligence over them",
		Category:    "Detention",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "View the detention desk, occurrences, dispute packets and detention intelligence",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Review",
				Description: "Approve, waive and dispute occurrences, and send detention notices",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	r.registerRateResources()

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceFactoringAgreement.String(),
		DisplayName: "Factoring Agreement",
		Description: "Advance, reserve, fee and recourse terms agreed with a factor",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View factoring agreements"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Draft factoring agreements"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Change, activate and end factoring agreements",
			},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export factoring agreements",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"accountingTreatment":    SensitivityInternal,
			"activatedAt":            SensitivityInternal,
			"businessUnitId":         SensitivityInternal,
			"cashAccountId":          SensitivityInternal,
			"chargebackWindowDays":   SensitivityInternal,
			"createdAt":              SensitivityInternal,
			"daysFrom":               SensitivityInternal,
			"daysTo":                 SensitivityInternal,
			"documentTypes":          SensitivityInternal,
			"dueFromFactorAccountId": SensitivityInternal,
			"effectiveFrom":          SensitivityInternal,
			"effectiveTo":            SensitivityInternal,
			"endedAt":                SensitivityInternal,
			"factoringCompany":       SensitivityInternal,
			"factoringCompanyId":     SensitivityInternal,
			"feeExpenseAccountId":    SensitivityInternal,
			"feeTiers":               SensitivityInternal,
			"fundingDays":            SensitivityInternal,
			"id":                     SensitivityInternal,
			"liabilityAccountId":     SensitivityInternal,
			"organizationId":         SensitivityInternal,
			"recourse":               SensitivityInternal,
			"status":                 SensitivityInternal,
			"submissionTime":         SensitivityInternal,
			"updatedAt":              SensitivityInternal,
			"version":                SensitivityInternal,
			"advanceRateBp":          SensitivityRestricted,
			"flatMinor":              SensitivityRestricted,
			"minimumFeeMinor":        SensitivityRestricted,
			"notes":                  SensitivityRestricted,
			"rateBp":                 SensitivityRestricted,
			"reserveRateBp":          SensitivityRestricted,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceFactoringNoa.String(),
		DisplayName: "Notice of Assignment",
		Description: "Notices telling customers to pay a factor instead",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View notices of assignment"},
			{Operation: OpCreate, DisplayName: "Draft", Description: "Draft notices of assignment"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Send, record and release notices of assignment",
			},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export notices of assignment",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"acknowledgedAt":     SensitivityInternal,
			"businessUnitId":     SensitivityInternal,
			"createdAt":          SensitivityInternal,
			"customer":           SensitivityInternal,
			"customerId":         SensitivityInternal,
			"expiresAt":          SensitivityInternal,
			"factoringCompany":   SensitivityInternal,
			"factoringCompanyId": SensitivityInternal,
			"id":                 SensitivityInternal,
			"organizationId":     SensitivityInternal,
			"releaseReason":      SensitivityInternal,
			"releasedAt":         SensitivityInternal,
			"sentAt":             SensitivityInternal,
			"sentTo":             SensitivityInternal,
			"signedDocumentId":   SensitivityInternal,
			"status":             SensitivityInternal,
			"updatedAt":          SensitivityInternal,
			"version":            SensitivityInternal,
			"notes":              SensitivityRestricted,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      ResourceFactoringSubmission.String(),
		DisplayName:   "Factoring Schedule",
		Description:   "Schedules of accounts sent to a factor, with each invoice's expected funding",
		Category:      "Billing",
		ReferenceView: []string{"number"},
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View factoring schedules"},
			{
				Operation:   OpSubmit,
				DisplayName: "Submit",
				Description: "Send invoices to a factor on a schedule",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Retry a schedule that failed to send",
			},
			{
				Operation:   OpCancel,
				DisplayName: "Cancel",
				Description: "Cancel a schedule that failed to send and release its invoices",
			},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export factoring schedules",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"agreementId":           SensitivityInternal,
			"attempts":              SensitivityInternal,
			"businessUnitId":        SensitivityInternal,
			"cancelReason":          SensitivityInternal,
			"cancelledAt":           SensitivityInternal,
			"chargebackReason":      SensitivityInternal,
			"chargebackWarnedAt":    SensitivityInternal,
			"chargedBackOn":         SensitivityInternal,
			"createdAt":             SensitivityInternal,
			"currencyCode":          SensitivityInternal,
			"customer":              SensitivityInternal,
			"customerId":            SensitivityInternal,
			"expectedDays":          SensitivityInternal,
			"factoringCompany":      SensitivityInternal,
			"factoringCompanyId":    SensitivityInternal,
			"failedAt":              SensitivityInternal,
			"fundedAt":              SensitivityInternal,
			"fundedOn":              SensitivityInternal,
			"id":                    SensitivityInternal,
			"invoice":               SensitivityInternal,
			"invoiceCount":          SensitivityInternal,
			"invoiceId":             SensitivityInternal,
			"invoiceNumber":         SensitivityInternal,
			"items":                 SensitivityInternal,
			"lastError":             SensitivityInternal,
			"number":                SensitivityInternal,
			"organizationId":        SensitivityInternal,
			"packetDocumentId":      SensitivityInternal,
			"position":              SensitivityInternal,
			"rejectionReason":       SensitivityInternal,
			"releasedAt":            SensitivityInternal,
			"scheduleCsvDocumentId": SensitivityInternal,
			"schedulePdfDocumentId": SensitivityInternal,
			"sendAttemptGroup":      SensitivityInternal,
			"sentAt":                SensitivityInternal,
			"settledOn":             SensitivityInternal,
			"status":                SensitivityInternal,
			"submissionId":          SensitivityInternal,
			"submittedById":         SensitivityInternal,
			"transportMethod":       SensitivityInternal,
			"transportReference":    SensitivityInternal,
			"transportTarget":       SensitivityInternal,
			"updatedAt":             SensitivityInternal,
			"version":               SensitivityInternal,
			"advanceMinor":          SensitivityRestricted,
			"expectedAdvanceMinor":  SensitivityRestricted,
			"expectedFeeMinor":      SensitivityRestricted,
			"expectedReserveMinor":  SensitivityRestricted,
			"faceMinor":             SensitivityRestricted,
			"feeMinor":              SensitivityRestricted,
			"reserveMinor":          SensitivityRestricted,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceFactoringReserve.String(),
		DisplayName: "Factoring Reserve",
		Description: "What factors funded, held in reserve, released and charged back, booked to the ledger",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "View factor funding, reserve balances and chargebacks",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Book",
				Description: "Record a factor's funding, release a reserve, and record or reverse a chargeback",
			},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export factoring reserve entries",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"entryCount":         SensitivityInternal,
			"factoringCompanyId": SensitivityInternal,
			"lastOccurredOn":     SensitivityInternal,
			"advancedMinor":      SensitivityRestricted,
			"chargedBackMinor":   SensitivityRestricted,
			"feeMinor":           SensitivityRestricted,
			"liabilityMinor":     SensitivityRestricted,
			"releasedMinor":      SensitivityRestricted,
			"reserveMinor":       SensitivityRestricted,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceFactoringRemittance.String(),
		DisplayName:    "Factoring Remittance",
		Description:    "A factor's funding or reserve report, matched to schedule items before it is booked",
		Category:       "Billing",
		ParentResource: ResourceFactoringReserve.String(),
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "View",
				Description: "View remittances and their lines",
			},
			{
				Operation:   OpCreate,
				DisplayName: operationDefinitions[OpImport].DisplayName,
				Description: "Upload a factor's remittance and read it",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Resolve",
				Description: "Match, ignore or restore a remittance line",
			},
			{
				Operation:   OpImport,
				DisplayName: "Apply",
				Description: "Book a remittance's matched lines to the ledger",
			},
			{
				Operation:   OpCancel,
				DisplayName: displayDiscard,
				Description: "Discard a remittance that will not be booked",
			},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export remittances and their lines",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"appliedAt":            SensitivityInternal,
			"appliedById":          SensitivityInternal,
			"appliedCount":         SensitivityInternal,
			"businessUnitId":       SensitivityInternal,
			"column":               SensitivityInternal,
			"createdAt":            SensitivityInternal,
			"createdById":          SensitivityInternal,
			"currencyCode":         SensitivityInternal,
			"debtorName":           SensitivityInternal,
			"defaultKind":          SensitivityInternal,
			"discardReason":        SensitivityInternal,
			"discardedAt":          SensitivityInternal,
			"documentId":           SensitivityInternal,
			"error":                SensitivityInternal,
			"exceptionCount":       SensitivityInternal,
			"factorReference":      SensitivityInternal,
			"factoringCompany":     SensitivityInternal,
			"factoringCompanyId":   SensitivityInternal,
			"field":                SensitivityInternal,
			"fileName":             SensitivityInternal,
			"headers":              SensitivityInternal,
			"id":                   SensitivityInternal,
			"ignoredReason":        SensitivityInternal,
			"inboundMessageId":     SensitivityInternal,
			"invoiceNumber":        SensitivityInternal,
			"itemId":               SensitivityInternal,
			"kind":                 SensitivityInternal,
			"lineCount":            SensitivityInternal,
			"lines":                SensitivityInternal,
			"mapping":              SensitivityInternal,
			"matchStatus":          SensitivityInternal,
			"matchedBy":            SensitivityInternal,
			"matchedCount":         SensitivityInternal,
			"matchedInvoiceNumber": SensitivityInternal,
			"matchedSubmissionId":  SensitivityInternal,
			"occurredOn":           SensitivityInternal,
			"organizationId":       SensitivityInternal,
			"reason":               SensitivityInternal,
			"remittanceId":         SensitivityInternal,
			"remittedOn":           SensitivityInternal,
			"reserveEntryId":       SensitivityInternal,
			"rowNumber":            SensitivityInternal,
			"score":                SensitivityInternal,
			"source":               SensitivityInternal,
			"sourceFormat":         SensitivityInternal,
			"stagedAt":             SensitivityInternal,
			"status":               SensitivityInternal,
			"submissionId":         SensitivityInternal,
			"submissionItemId":     SensitivityInternal,
			"suggestions":          SensitivityInternal,
			"unmappedHeaders":      SensitivityInternal,
			"updatedAt":            SensitivityInternal,
			"version":              SensitivityInternal,
			"advanceMinor":         SensitivityRestricted,
			"cashMinor":            SensitivityRestricted,
			"cells":                SensitivityRestricted,
			"faceMinor":            SensitivityRestricted,
			"feeMinor":             SensitivityRestricted,
			"reserveMinor":         SensitivityRestricted,
			"totalCashMinor":       SensitivityRestricted,
		},
	})
}
