package permissioncatalog

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const displayDiscard = "Discard"

func (r *Registry) registerBillingResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceInvoiceRun.String(),
		DisplayName: "Invoice Run",
		Description: "Consolidated invoice batch preview and commit",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View invoice runs"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Build an invoice run preview",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Adjust invoice run membership",
			},
			// Issuing invoices to customers is a separable authority from building
			// a preview, so a four-eyes billing shop can grant one without the other.
			{
				Operation:   permission.OpApprove,
				DisplayName: "Commit",
				Description: "Commit an invoice run and issue its invoices",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: "Cancel",
				Description: "Discard an invoice run",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceInvoiceDispute.String(),
		DisplayName:   "Invoice Dispute",
		Description:   "Dispute cases raised against posted invoices",
		Category:      "Billing",
		ReferenceView: []string{"invoiceId", "status", "reasonCode", "openedAt", "resolvedAt"},
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View invoice disputes",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Open",
				Description: "Open a dispute on an invoice",
			},
			{
				Operation:   permission.OpApprove,
				DisplayName: "Resolve",
				Description: "Resolve an open dispute",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: "Withdraw",
				Description: "Withdraw an open dispute",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":                     permission.SensitivityInternal,
			"organizationId":         permission.SensitivityInternal,
			"businessUnitId":         permission.SensitivityInternal,
			"invoiceId":              permission.SensitivityInternal,
			"customerId":             permission.SensitivityInternal,
			"status":                 permission.SensitivityInternal,
			"reasonCode":             permission.SensitivityInternal,
			"openedById":             permission.SensitivityInternal,
			"openedAt":               permission.SensitivityInternal,
			"resolvedById":           permission.SensitivityInternal,
			"resolvedAt":             permission.SensitivityInternal,
			"resolution":             permission.SensitivityInternal,
			"resolutionAdjustmentId": permission.SensitivityInternal,
			"version":                permission.SensitivityInternal,
			"createdAt":              permission.SensitivityInternal,
			"updatedAt":              permission.SensitivityInternal,
			"disputedAmount":         permission.SensitivityRestricted,
			"disputedAmountMinor":    permission.SensitivityRestricted,
			"notes":                  permission.SensitivityRestricted,
			"resolutionNotes":        permission.SensitivityRestricted,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceInvoice.String(),
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
			{Operation: permission.OpRead, DisplayName: "View", Description: "View invoices"},
			{Operation: permission.OpCreate, DisplayName: "Create", Description: "Create invoices"},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update and post",
				Description: "Modify invoices, submit adjustments and post invoices to the ledger",
			},
			{Operation: permission.OpExport, DisplayName: "Export", Description: "Export invoices"},
			{
				Operation:   permission.OpApprove,
				DisplayName: "Approve",
				Description: "Approve invoices, and approve or reject invoice adjustments",
			},
			{Operation: permission.OpReject, DisplayName: "Reject", Description: "Reject invoices"},
			{
				Operation:   permission.OpSubmit,
				DisplayName: "Submit and send",
				Description: "Submit invoices for approval, and send or resend them by email or EDI",
			},
			{Operation: permission.OpCancel, DisplayName: "Void", Description: "Void invoices"},
			{
				Operation:   permission.OpPost,
				DisplayName: "Post",
				Description: "Post invoices to the ledger, by hand or from the billing queue",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":                              permission.SensitivityInternal,
			"businessUnitId":                  permission.SensitivityInternal,
			"organizationId":                  permission.SensitivityInternal,
			"number":                          permission.SensitivityInternal,
			"status":                          permission.SensitivityInternal,
			"type":                            permission.SensitivityInternal,
			"billType":                        permission.SensitivityInternal,
			"invoiceDate":                     permission.SensitivityInternal,
			"dueDate":                         permission.SensitivityInternal,
			"serviceDate":                     permission.SensitivityInternal,
			"postedAt":                        permission.SensitivityInternal,
			"sentAt":                          permission.SensitivityInternal,
			"sentById":                        permission.SensitivityInternal,
			"sendStatus":                      permission.SensitivityInternal,
			"settlementStatus":                permission.SensitivityInternal,
			"disputeStatus":                   permission.SensitivityInternal,
			"currencyCode":                    permission.SensitivityInternal,
			"paymentTerm":                     permission.SensitivityInternal,
			"customerId":                      permission.SensitivityInternal,
			"shipmentId":                      permission.SensitivityInternal,
			"orderId":                         permission.SensitivityInternal,
			"orderNumber":                     permission.SensitivityInternal,
			"shipmentProNumber":               permission.SensitivityInternal,
			"shipmentBol":                     permission.SensitivityInternal,
			"invoiceId":                       permission.SensitivityInternal,
			"lineNumber":                      permission.SensitivityInternal,
			"description":                     permission.SensitivityInternal,
			"billToName":                      permission.SensitivityInternal,
			"billToCode":                      permission.SensitivityInternal,
			"billToAddressLine1":              permission.SensitivityInternal,
			"billToAddressLine2":              permission.SensitivityInternal,
			"billToCity":                      permission.SensitivityInternal,
			"billToState":                     permission.SensitivityInternal,
			"billToCountry":                   permission.SensitivityInternal,
			"billToPostalCode":                permission.SensitivityInternal,
			"billingQueueItemId":              permission.SensitivityInternal,
			"pdfDocumentId":                   permission.SensitivityInternal,
			"correctionGroupId":               permission.SensitivityInternal,
			"sourceInvoiceAdjustmentId":       permission.SensitivityInternal,
			"supersededByInvoiceId":           permission.SensitivityInternal,
			"supersedesInvoiceId":             permission.SensitivityInternal,
			"isAdjustmentArtifact":            permission.SensitivityInternal,
			"scope":                           permission.SensitivityInternal,
			"shipmentCount":                   permission.SensitivityInternal,
			"voidedAt":                        permission.SensitivityInternal,
			"ediSendStatus":                   permission.SensitivityInternal,
			"chargeCode":                      permission.SensitivityInternal,
			"accessorialChargeId":             permission.SensitivityInternal,
			"executionError":                  permission.SensitivityInternal,
			"supportingDocumentIds":           permission.SensitivityInternal,
			"lines":                           permission.SensitivityInternal,
			"verdict":                         permission.SensitivityInternal,
			"failedChecks":                    permission.SensitivityInternal,
			"readySince":                      permission.SensitivityInternal,
			"evaluatedAt":                     permission.SensitivityInternal,
			"checks":                          permission.SensitivityInternal,
			"code":                            permission.SensitivityInternal,
			"passed":                          permission.SensitivityInternal,
			"message":                         permission.SensitivityInternal,
			"fix":                             permission.SensitivityInternal,
			"documentTypeId":                  permission.SensitivityInternal,
			"documentTypeName":                permission.SensitivityInternal,
			"shipmentPro":                     permission.SensitivityInternal,
			"factoringCompanyId":              permission.SensitivityInternal,
			"version":                         permission.SensitivityInternal,
			"createdAt":                       permission.SensitivityInternal,
			"updatedAt":                       permission.SensitivityInternal,
			"amount":                          permission.SensitivityRestricted,
			"amountMinor":                     permission.SensitivityRestricted,
			"appliedAmount":                   permission.SensitivityRestricted,
			"appliedAmountMinor":              permission.SensitivityRestricted,
			"otherAmount":                     permission.SensitivityRestricted,
			"otherAmountMinor":                permission.SensitivityRestricted,
			"subtotalAmount":                  permission.SensitivityRestricted,
			"subtotalAmountMinor":             permission.SensitivityRestricted,
			"totalAmount":                     permission.SensitivityRestricted,
			"totalAmountMinor":                permission.SensitivityRestricted,
			"quantity":                        permission.SensitivityRestricted,
			"unitPrice":                       permission.SensitivityRestricted,
			"memo":                            permission.SensitivityRestricted,
			"remittanceInstructions":          permission.SensitivityRestricted,
			"emailToSnapshot":                 permission.SensitivityRestricted,
			"emailCcSnapshot":                 permission.SensitivityRestricted,
			"emailBccSnapshot":                permission.SensitivityRestricted,
			"emailSubjectSnapshot":            permission.SensitivityRestricted,
			"emailBodySnapshot":               permission.SensitivityRestricted,
			"lastSendError":                   permission.SensitivityRestricted,
			"lastSendWarning":                 permission.SensitivityRestricted,
			"accountingDate":                  permission.SensitivityInternal,
			"adjustmentId":                    permission.SensitivityInternal,
			"approvalRequired":                permission.SensitivityInternal,
			"approvalStatus":                  permission.SensitivityInternal,
			"approvedAt":                      permission.SensitivityInternal,
			"approvedById":                    permission.SensitivityInternal,
			"batchId":                         permission.SensitivityInternal,
			"creditMemoInvoiceId":             permission.SensitivityInternal,
			"creditMemoLineId":                permission.SensitivityInternal,
			"kind":                            permission.SensitivityInternal,
			"originalInvoiceId":               permission.SensitivityInternal,
			"originalLineId":                  permission.SensitivityInternal,
			"rebillQueueItemId":               permission.SensitivityInternal,
			"rebillStrategy":                  permission.SensitivityInternal,
			"rejectedAt":                      permission.SensitivityInternal,
			"rejectedById":                    permission.SensitivityInternal,
			"replacementInvoiceId":            permission.SensitivityInternal,
			"replacementLineId":               permission.SensitivityInternal,
			"replacementReviewStatus":         permission.SensitivityInternal,
			"requiresReconciliationException": permission.SensitivityInternal,
			"submittedAt":                     permission.SensitivityInternal,
			"submittedById":                   permission.SensitivityInternal,
			"wouldCreateUnappliedCredit":      permission.SensitivityInternal,
			"creditAmount":                    permission.SensitivityRestricted,
			"creditAmountMinor":               permission.SensitivityRestricted,
			"creditQuantity":                  permission.SensitivityRestricted,
			"creditTotalAmount":               permission.SensitivityRestricted,
			"creditTotalAmountMinor":          permission.SensitivityRestricted,
			"netDeltaAmount":                  permission.SensitivityRestricted,
			"netDeltaAmountMinor":             permission.SensitivityRestricted,
			"policyReason":                    permission.SensitivityRestricted,
			"reason":                          permission.SensitivityRestricted,
			"rebillAmount":                    permission.SensitivityRestricted,
			"rebillAmountMinor":               permission.SensitivityRestricted,
			"rebillQuantity":                  permission.SensitivityRestricted,
			"rebillTotalAmount":               permission.SensitivityRestricted,
			"rebillTotalAmountMinor":          permission.SensitivityRestricted,
			"rejectionReason":                 permission.SensitivityRestricted,
			"remainingEligibleAmount":         permission.SensitivityRestricted,
			"rerateVariancePercent":           permission.SensitivityRestricted,
		},
		Scopes: []ScopeSupport{
			{Scope: permission.DataScopeOwn, EnforcedBy: ScopeEnforcedByDatabase},
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceBillingQueue.String(),
		DisplayName:   "Billing Queue",
		Description:   "Billing queue review and approval",
		Category:      "Billing",
		ReferenceView: []string{"number", "status"},
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View billing queue items",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Transfer shipments to billing queue",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Update billing queue items: post them, release them, resolve their issues and approve ratings a tariff held for review",
			},
			{
				Operation:   permission.OpAssign,
				DisplayName: "Assign",
				Description: "Assign billers to queue items",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":                        permission.SensitivityInternal,
			"businessUnitId":            permission.SensitivityInternal,
			"organizationId":            permission.SensitivityInternal,
			"shipmentId":                permission.SensitivityInternal,
			"shipment.proNumber":        permission.SensitivityInternal,
			"orderId":                   permission.SensitivityInternal,
			"billToCustomerId":          permission.SensitivityInternal,
			"assignedBillerId":          permission.SensitivityInternal,
			"number":                    permission.SensitivityInternal,
			"status":                    permission.SensitivityInternal,
			"billType":                  permission.SensitivityInternal,
			"exceptionReasonCode":       permission.SensitivityInternal,
			"reviewNotes":               permission.SensitivityInternal,
			"exceptionNotes":            permission.SensitivityInternal,
			"reviewStartedAt":           permission.SensitivityInternal,
			"reviewCompletedAt":         permission.SensitivityInternal,
			"canceledById":              permission.SensitivityInternal,
			"canceledAt":                permission.SensitivityInternal,
			"cancelReason":              permission.SensitivityInternal,
			"invoiceId":                 permission.SensitivityInternal,
			"requiresReplacementReview": permission.SensitivityInternal,
			"createdAt":                 permission.SensitivityInternal,
			"updatedAt":                 permission.SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceAgentRun.String(),
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
			{Operation: permission.OpRead, DisplayName: "View", Description: "View agent runs"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Start agent runs",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceAgentProposal.String(),
		DisplayName:   "Decisions",
		Description:   "What the agents propose and ask a person to approve or decline",
		Category:      "Desk",
		ReferenceView: []string{"toolName", "status"},
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View agent proposals",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Accept, modify or reject agent proposals, and resolve proposals and plans",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceAgentException.String(),
		DisplayName: "Agent Exception",
		Description: "Agent exceptions raised for human review",
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View agent exceptions",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Raise agent exceptions",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Resolve agent exceptions",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceAgentControl.String(),
		DisplayName: "Agent Control",
		Description: "Per-organization agent autonomy configuration",
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View agent control settings, and dismiss or restore provider failure notices",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Update agent control settings, and apply or restore tune-ups",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	// What the organization has told its agents. Reading it shows what every
	// agent is told. A person's own memories need only the assistant, and
	// anyone may offer one for their team or the organization; keeping,
	// changing, retiring and approving what reaches other people's
	// conversations is manage, which the role templates never hand out, so it
	// stays an administrator's unless a role is given it on purpose.
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceAgentMemory.String(),
		DisplayName: "Agent Memory",
		Description: "Standing instructions, facts, corrections and profile guidance kept for agents between runs",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View what agents remember",
			},
			{
				Operation:   permission.OpManage,
				DisplayName: operationDefinitions[permission.OpManage].DisplayName,
				Description: "Keep, change, retire, review and approve memories and profile guidance shared with a team, the organization or every agent",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceAgentEvalSuite.String(),
		DisplayName: "Agent Evaluation Suite",
		Description: "Evaluation cases agents are replayed against and scored on",
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View evaluation cases and scores",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Capture or write an evaluation case and replay it",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Edit what a case expects, activate, quarantine or retire it, and cancel runs",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceAgentFeedback.String(),
		DisplayName: "Agent Feedback",
		Description: "Everyone's ratings of AI output, with what they saw and why they rated it",
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View everyone's ratings of AI output and each agent's satisfaction",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	// The sealed record of what every agent did and on whose behalf: each
	// model call, tool call, proposal, decision and write. It names people,
	// records and arguments across the whole organization, so it is kept for
	// those who answer for the organization's agents; reading an agent's runs
	// does not grant it, and no agent can hold it.
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceAIAuditTrail.String(),
		DisplayName: "AI Audit Trail",
		Description: "The tamper-evident record of what agents did, for whom, and who decided",
		Category:    "Users & security",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View the AI audit trail and verify its chain",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export the AI audit trail and download exports",
			},
		},
		DefaultSensitivity: permission.SensitivityConfidential,
	})

	// unattended, so it sits at the same sensitivity as provider configuration.
	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceAgentDefinition.String(),
		DisplayName:   "Agent Definition",
		Description:   "Per-organization agent configurations and their tool access",
		Category:      "AI & agents",
		ReferenceView: []string{"name", "icon", "accent", "template"},
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View configured agents",
			},
			{Operation: permission.OpCreate, DisplayName: "Create", Description: "Add an agent"},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Update an agent's tools and autonomy",
			},
			{Operation: permission.OpDelete, DisplayName: "Delete", Description: "Remove an agent"},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	// Using the assistant is an ordinary operational act rather than an
	// administrative one, so it is held separately from configuring agents: a
	// dispatcher should be able to ask questions without being able to change
	// which tools an agent holds.
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceAssistant.String(),
		DisplayName: "Desk",
		Description: "Conversations with the agents and what they produce",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View own conversations and pin artifacts in them",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Start conversations and send messages",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Delete own conversations",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	// An insight names a customer, a location or a driver alongside a figure, so
	// reading the panel is a real disclosure. It is held separately from the
	// records it draws on because a reader still needs permission over those:
	// the panel filters to what its reader may already see, and this permission
	// only decides whether they get a panel at all.
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceInsight.String(),
		DisplayName: "Insight",
		Description: "Operational findings surfaced on the home screen",
		Category:    "Reports",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View operational insights",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Dismiss an insight that is not worth acting on, or restore one",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	// The watchtower is a feed over records a reader may already be allowed to
	// see: every item is filtered again by the permission on the source it came
	// from, so this one decides only whether a person gets the feed at all.
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceWatchtower.String(),
		DisplayName: "Watchtower",
		Description: "The live feed of what needs attention across the operation",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View the watchtower feed",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Mark the feed seen, dismiss an item, or hand one to an agent",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	// A briefing gathers the morning's figures across dispatch, billing and
	// compliance into one page, which is a wider disclosure than any single
	// screen it draws on.
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceBriefing.String(),
		DisplayName: "Briefing",
		Description: "The daily briefing written for a role from the day's figures",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "Read the daily briefing",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Write today's briefing again from current figures",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	// Configuring a provider decides which endpoint an organization's freight and
	// billing data is sent to, and whether that endpoint may be on the local
	// network, so it is held at the same sensitivity as credential management.
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceAgentExtension.String(),
		DisplayName: "Agent Extension",
		Description: "Capabilities turned on for agents only, such as web research, and their credentials",
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View the extension marketplace",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Turn extensions on or off, change their settings and test them",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceWebResearch.String(),
		DisplayName: "Web Research",
		Description: "Agents searching and reading public web pages on a person's behalf, through an enabled extension",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "Let agents search and read the web for this person",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourcePortalAutomation.String(),
		DisplayName: "Portal Automation",
		Description: "Agents reading and acting on the third-party web portals the organization set up, through an enabled extension",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "Let agents look things up on portals for this person",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Act",
				Description: "Let agents propose booking, uploading or accepting on portals for this person",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceCodeMode.String(),
		DisplayName: "Scripts and analysis",
		Description: "Agents answering questions across many records, and analyzing report results and attached files, with short read-only scripts",
		Category:    "Desk",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "Let agents run read-only scripts over what this person may read, the reports they ran and the files they attached",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceAIProvider.String(),
		DisplayName: "AI Provider",
		Description: "Model endpoints, including self-hosted servers, and their task routing",
		Category:    "AI & agents",
		Operations: []OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View AI providers"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Add an AI provider",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Update an AI provider and its task routing",
			},
			{
				Operation:   permission.OpDelete,
				DisplayName: "Delete",
				Description: "Remove an AI provider",
			},
			{
				Operation:   permission.OpManage,
				DisplayName: "Manage",
				Description: "Test an AI provider connection",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceAccessorialCharge.String(),
		DisplayName:        "Accessorial Charge",
		Description:        "Accessorial charge definitions",
		Category:           "Billing",
		ReferenceView:      []string{"code", "description", "status", "method", "rateUnit"},
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceChargeType.String(),
		DisplayName:        "Charge Type",
		Description:        "Charge type definitions",
		Category:           "Billing",
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceRevenueCode.String(),
		DisplayName:        "Revenue Code",
		Description:        "Revenue code definitions",
		Category:           "Billing",
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceFormulaTemplate.String(),
		DisplayName: "Formula Template",
		Description: "Rate formula templates",
		Category:    "Billing",
		Operations: append(slices.Clone(standardOps),
			OperationDefinition{
				Operation:   permission.OpDuplicate,
				DisplayName: "Duplicate",
				Description: "Duplicate formula templates",
			},
			OperationDefinition{
				Operation:   permission.OpSubmit,
				DisplayName: "Submit",
				Description: "Submit formula templates for review",
			},
			OperationDefinition{
				Operation:   permission.OpApprove,
				DisplayName: "Approve",
				Description: "Approve formula templates and schedule rate changes",
			},
			OperationDefinition{
				Operation:   permission.OpReject,
				DisplayName: "Reject",
				Description: "Reject formula templates under review",
			},
			OperationDefinition{
				Operation:   permission.OpApproveOwn,
				DisplayName: "Approve own changes",
				Description: "Approve a formula template revision you submitted or edited, without a second reviewer",
			},
		),
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceFormulaLibrary.String(),
		DisplayName: "Formula Library",
		Description: "Organization formula libraries that rate formula templates import",
		Category:    "Billing",
		Operations: append(slices.Clone(standardOps),
			OperationDefinition{
				Operation:   permission.OpDuplicate,
				DisplayName: "Duplicate",
				Description: "Duplicate formula libraries",
			},
			OperationDefinition{
				Operation:   permission.OpSubmit,
				DisplayName: "Submit",
				Description: "Submit formula libraries for review",
			},
			OperationDefinition{
				Operation:   permission.OpApprove,
				DisplayName: "Approve",
				Description: "Approve formula library versions templates can pin",
			},
			OperationDefinition{
				Operation:   permission.OpReject,
				DisplayName: "Reject",
				Description: "Reject formula libraries under review",
			},
			OperationDefinition{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Archive formula libraries so no new template imports them",
			},
			OperationDefinition{
				Operation:   permission.OpApproveOwn,
				DisplayName: "Approve own changes",
				Description: "Approve a formula library revision you submitted or edited, without a second reviewer",
			},
		),
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceFuelSurchargeProgram.String(),
		DisplayName:        "Fuel Surcharge Program",
		Description:        "Fuel surcharge programs, fuel price indices, and DOE weekly price data",
		Category:           "Fuel & tax",
		ReferenceView:      []string{"code", "name", "status"},
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceDetentionPolicy.String(),
		DisplayName: "Detention Policy",
		Description: "Contractual detention terms, rate tiers and notice requirements",
		Category:    "Detention",
		Operations: relabel(
			standardOpsWithDelete,
			permission.OpUpdate,
			"Update",
			"Modify detention policies and the terms they apply",
		),
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceDetention.String(),
		DisplayName: "Detention",
		Description: "Detention at stops: the detention desk, its occurrences and the intelligence over them",
		Category:    "Detention",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View the detention desk, occurrences, dispute packets and detention intelligence",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Review",
				Description: "Approve, waive and dispute occurrences, and send detention notices",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	r.registerRateResources()

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceFactoringAgreement.String(),
		DisplayName: "Factoring Agreement",
		Description: "Advance, reserve, fee and recourse terms agreed with a factor",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View factoring agreements",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Draft factoring agreements",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Change, activate and end factoring agreements",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export factoring agreements",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"accountingTreatment":    permission.SensitivityInternal,
			"activatedAt":            permission.SensitivityInternal,
			"businessUnitId":         permission.SensitivityInternal,
			"cashAccountId":          permission.SensitivityInternal,
			"chargebackWindowDays":   permission.SensitivityInternal,
			"createdAt":              permission.SensitivityInternal,
			"daysFrom":               permission.SensitivityInternal,
			"daysTo":                 permission.SensitivityInternal,
			"documentTypes":          permission.SensitivityInternal,
			"dueFromFactorAccountId": permission.SensitivityInternal,
			"effectiveFrom":          permission.SensitivityInternal,
			"effectiveTo":            permission.SensitivityInternal,
			"endedAt":                permission.SensitivityInternal,
			"factoringCompany":       permission.SensitivityInternal,
			"factoringCompanyId":     permission.SensitivityInternal,
			"feeExpenseAccountId":    permission.SensitivityInternal,
			"feeTiers":               permission.SensitivityInternal,
			"fundingDays":            permission.SensitivityInternal,
			"id":                     permission.SensitivityInternal,
			"liabilityAccountId":     permission.SensitivityInternal,
			"organizationId":         permission.SensitivityInternal,
			"recourse":               permission.SensitivityInternal,
			"status":                 permission.SensitivityInternal,
			"submissionTime":         permission.SensitivityInternal,
			"updatedAt":              permission.SensitivityInternal,
			"version":                permission.SensitivityInternal,
			"advanceRateBp":          permission.SensitivityRestricted,
			"flatMinor":              permission.SensitivityRestricted,
			"minimumFeeMinor":        permission.SensitivityRestricted,
			"notes":                  permission.SensitivityRestricted,
			"rateBp":                 permission.SensitivityRestricted,
			"reserveRateBp":          permission.SensitivityRestricted,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceFactoringNoa.String(),
		DisplayName: "Notice of Assignment",
		Description: "Notices telling customers to pay a factor instead",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View notices of assignment",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Draft",
				Description: "Draft notices of assignment",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Send, record and release notices of assignment",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export notices of assignment",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"acknowledgedAt":     permission.SensitivityInternal,
			"businessUnitId":     permission.SensitivityInternal,
			"createdAt":          permission.SensitivityInternal,
			"customer":           permission.SensitivityInternal,
			"customerId":         permission.SensitivityInternal,
			"expiresAt":          permission.SensitivityInternal,
			"factoringCompany":   permission.SensitivityInternal,
			"factoringCompanyId": permission.SensitivityInternal,
			"id":                 permission.SensitivityInternal,
			"organizationId":     permission.SensitivityInternal,
			"releaseReason":      permission.SensitivityInternal,
			"releasedAt":         permission.SensitivityInternal,
			"sentAt":             permission.SensitivityInternal,
			"sentTo":             permission.SensitivityInternal,
			"signedDocumentId":   permission.SensitivityInternal,
			"status":             permission.SensitivityInternal,
			"updatedAt":          permission.SensitivityInternal,
			"version":            permission.SensitivityInternal,
			"notes":              permission.SensitivityRestricted,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceFactoringSubmission.String(),
		DisplayName:   "Factoring Schedule",
		Description:   "Schedules of accounts sent to a factor, with each invoice's expected funding",
		Category:      "Billing",
		ReferenceView: []string{"number"},
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View factoring schedules",
			},
			{
				Operation:   permission.OpSubmit,
				DisplayName: "Submit",
				Description: "Send invoices to a factor on a schedule",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Retry a schedule that failed to send",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: "Cancel",
				Description: "Cancel a schedule that failed to send and release its invoices",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export factoring schedules",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"agreementId":           permission.SensitivityInternal,
			"attempts":              permission.SensitivityInternal,
			"businessUnitId":        permission.SensitivityInternal,
			"cancelReason":          permission.SensitivityInternal,
			"cancelledAt":           permission.SensitivityInternal,
			"chargebackReason":      permission.SensitivityInternal,
			"chargebackWarnedAt":    permission.SensitivityInternal,
			"chargedBackOn":         permission.SensitivityInternal,
			"createdAt":             permission.SensitivityInternal,
			"currencyCode":          permission.SensitivityInternal,
			"customer":              permission.SensitivityInternal,
			"customerId":            permission.SensitivityInternal,
			"expectedDays":          permission.SensitivityInternal,
			"factoringCompany":      permission.SensitivityInternal,
			"factoringCompanyId":    permission.SensitivityInternal,
			"failedAt":              permission.SensitivityInternal,
			"fundedAt":              permission.SensitivityInternal,
			"fundedOn":              permission.SensitivityInternal,
			"id":                    permission.SensitivityInternal,
			"invoice":               permission.SensitivityInternal,
			"invoiceCount":          permission.SensitivityInternal,
			"invoiceId":             permission.SensitivityInternal,
			"invoiceNumber":         permission.SensitivityInternal,
			"items":                 permission.SensitivityInternal,
			"lastError":             permission.SensitivityInternal,
			"number":                permission.SensitivityInternal,
			"organizationId":        permission.SensitivityInternal,
			"packetDocumentId":      permission.SensitivityInternal,
			"position":              permission.SensitivityInternal,
			"rejectionReason":       permission.SensitivityInternal,
			"releasedAt":            permission.SensitivityInternal,
			"scheduleCsvDocumentId": permission.SensitivityInternal,
			"schedulePdfDocumentId": permission.SensitivityInternal,
			"sendAttemptGroup":      permission.SensitivityInternal,
			"sentAt":                permission.SensitivityInternal,
			"settledOn":             permission.SensitivityInternal,
			"status":                permission.SensitivityInternal,
			"submissionId":          permission.SensitivityInternal,
			"submittedById":         permission.SensitivityInternal,
			"transportMethod":       permission.SensitivityInternal,
			"transportReference":    permission.SensitivityInternal,
			"transportTarget":       permission.SensitivityInternal,
			"updatedAt":             permission.SensitivityInternal,
			"version":               permission.SensitivityInternal,
			"advanceMinor":          permission.SensitivityRestricted,
			"expectedAdvanceMinor":  permission.SensitivityRestricted,
			"expectedFeeMinor":      permission.SensitivityRestricted,
			"expectedReserveMinor":  permission.SensitivityRestricted,
			"faceMinor":             permission.SensitivityRestricted,
			"feeMinor":              permission.SensitivityRestricted,
			"reserveMinor":          permission.SensitivityRestricted,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceFactoringReserve.String(),
		DisplayName: "Factoring Reserve",
		Description: "What factors funded, held in reserve, released and charged back, booked to the ledger",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View factor funding, reserve balances and chargebacks",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Book",
				Description: "Record a factor's funding, release a reserve, and record or reverse a chargeback",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export factoring reserve entries",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"entryCount":         permission.SensitivityInternal,
			"factoringCompanyId": permission.SensitivityInternal,
			"lastOccurredOn":     permission.SensitivityInternal,
			"advancedMinor":      permission.SensitivityRestricted,
			"chargedBackMinor":   permission.SensitivityRestricted,
			"feeMinor":           permission.SensitivityRestricted,
			"liabilityMinor":     permission.SensitivityRestricted,
			"releasedMinor":      permission.SensitivityRestricted,
			"reserveMinor":       permission.SensitivityRestricted,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       permission.ResourceFactoringRemittance.String(),
		DisplayName:    "Factoring Remittance",
		Description:    "A factor's funding or reserve report, matched to schedule items before it is booked",
		Category:       "Billing",
		ParentResource: permission.ResourceFactoringReserve.String(),
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View remittances and their lines",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: operationDefinitions[permission.OpImport].DisplayName,
				Description: "Upload a factor's remittance and read it",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Resolve",
				Description: "Match, ignore or restore a remittance line",
			},
			{
				Operation:   permission.OpImport,
				DisplayName: "Apply",
				Description: "Book a remittance's matched lines to the ledger",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: displayDiscard,
				Description: "Discard a remittance that will not be booked",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export remittances and their lines",
			},
		},
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"appliedAt":            permission.SensitivityInternal,
			"appliedById":          permission.SensitivityInternal,
			"appliedCount":         permission.SensitivityInternal,
			"businessUnitId":       permission.SensitivityInternal,
			"column":               permission.SensitivityInternal,
			"createdAt":            permission.SensitivityInternal,
			"createdById":          permission.SensitivityInternal,
			"currencyCode":         permission.SensitivityInternal,
			"debtorName":           permission.SensitivityInternal,
			"defaultKind":          permission.SensitivityInternal,
			"discardReason":        permission.SensitivityInternal,
			"discardedAt":          permission.SensitivityInternal,
			"documentId":           permission.SensitivityInternal,
			"error":                permission.SensitivityInternal,
			"exceptionCount":       permission.SensitivityInternal,
			"factorReference":      permission.SensitivityInternal,
			"factoringCompany":     permission.SensitivityInternal,
			"factoringCompanyId":   permission.SensitivityInternal,
			"field":                permission.SensitivityInternal,
			"fileName":             permission.SensitivityInternal,
			"headers":              permission.SensitivityInternal,
			"id":                   permission.SensitivityInternal,
			"ignoredReason":        permission.SensitivityInternal,
			"inboundMessageId":     permission.SensitivityInternal,
			"invoiceNumber":        permission.SensitivityInternal,
			"itemId":               permission.SensitivityInternal,
			"kind":                 permission.SensitivityInternal,
			"lineCount":            permission.SensitivityInternal,
			"lines":                permission.SensitivityInternal,
			"mapping":              permission.SensitivityInternal,
			"matchStatus":          permission.SensitivityInternal,
			"matchedBy":            permission.SensitivityInternal,
			"matchedCount":         permission.SensitivityInternal,
			"matchedInvoiceNumber": permission.SensitivityInternal,
			"matchedSubmissionId":  permission.SensitivityInternal,
			"occurredOn":           permission.SensitivityInternal,
			"organizationId":       permission.SensitivityInternal,
			"reason":               permission.SensitivityInternal,
			"remittanceId":         permission.SensitivityInternal,
			"remittedOn":           permission.SensitivityInternal,
			"reserveEntryId":       permission.SensitivityInternal,
			"rowNumber":            permission.SensitivityInternal,
			"score":                permission.SensitivityInternal,
			"source":               permission.SensitivityInternal,
			"sourceFormat":         permission.SensitivityInternal,
			"stagedAt":             permission.SensitivityInternal,
			"status":               permission.SensitivityInternal,
			"submissionId":         permission.SensitivityInternal,
			"submissionItemId":     permission.SensitivityInternal,
			"suggestions":          permission.SensitivityInternal,
			"unmappedHeaders":      permission.SensitivityInternal,
			"updatedAt":            permission.SensitivityInternal,
			"version":              permission.SensitivityInternal,
			"advanceMinor":         permission.SensitivityRestricted,
			"cashMinor":            permission.SensitivityRestricted,
			"cells":                permission.SensitivityRestricted,
			"faceMinor":            permission.SensitivityRestricted,
			"feeMinor":             permission.SensitivityRestricted,
			"reserveMinor":         permission.SensitivityRestricted,
			"totalCashMinor":       permission.SensitivityRestricted,
		},
	})
}
