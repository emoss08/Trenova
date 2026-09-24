package permission

// agentAllowedPermissions is the whole permission model for an agent
// principal: there is no role behind it, so an entry missing here is a tool
// that works for a person and is refused to every agent, at the moment it
// tries to do the work it was woken for.
//
// It is therefore also the ceiling on what any agent can ever do, and it is
// kept to exactly what the registered tools need — no wider. The coverage
// test in agenttoolservice reads both halves and fails on either drift.
//
// OpApprove is deliberately absent: guardExecute refuses it to an agent
// principal outright, so approving is a person's, whatever any list says.
var agentAllowedPermissions = map[Resource]map[Operation]struct{}{
	// Creating a shipment from a document is the intake desk's whole job.
	// Editing or cancelling one that already exists is a person's: those
	// tools are offered in chat, where the actor is the person asking.
	ResourceShipment: {
		OpRead:   {},
		OpCreate: {},
	},
	ResourceShipmentComment: {
		OpCreate: {},
	},
	ResourceShipmentHold: {
		OpCreate: {},
		OpUpdate: {},
	},
	ResourceShipmentMove: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceWorker: {
		OpRead: {},
	},
	ResourceTractor: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceTrailer: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceWorkerPTO: {
		OpRead:   {},
		OpCancel: {},
		OpReject: {},
	},
	ResourceReport: {
		OpRead:   {},
		OpCreate: {},
		OpUpdate: {},
	},
	ResourceServiceFailure: {
		OpRead:   {},
		OpCreate: {},
		OpUpdate: {},
	},
	// The three surfaces an agent builds rather than reads: a saved view of a
	// table, a dashboard, and an alert that watches for a change. All three
	// are additive and reversible — nothing an agent makes here alters a
	// record, and a person can delete any of it.
	// No read on the other two: nothing lists them. A grant no tool claims is
	// one nobody can account for, which is what the coverage test holds.
	ResourceTableConfiguration: {
		OpCreate: {},
	},
	ResourceDashboard: {
		// list_dashboards claims the read: add_dashboard_tile needs an id and
		// nothing else hands one out.
		OpRead:   {},
		OpCreate: {},
		OpUpdate: {},
	},
	ResourceTableChangeAlert: {
		OpCreate: {},
	},
	ResourceCarrier: {
		OpRead: {},
	},
	ResourceCarrierIntelligence: {
		OpRead:   {},
		OpUpdate: {},
	},
	// The reference data every list and get tool resolves a name against.
	ResourceAccessorialCharge: {
		OpRead: {},
	},
	ResourceCommodity: {
		OpRead: {},
	},
	ResourceDocumentType: {
		OpRead: {},
	},
	ResourceEquipmentType: {
		OpRead: {},
	},
	ResourceFleetCode: {
		OpRead: {},
	},
	ResourceHazardousMaterial: {
		OpRead: {},
	},
	ResourceHoldReason: {
		OpRead: {},
	},
	ResourceLocationCategory: {
		OpRead: {},
	},
	ResourceServiceType: {
		OpRead: {},
	},
	ResourceShipmentType: {
		OpRead: {},
	},
	ResourceServiceFailureReasonCode: {
		OpRead: {},
	},
	ResourceDetentionPolicy: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceEmailProfile: {
		OpRead: {},
	},
	ResourceRateQuote: {
		OpRead: {},
	},
	ResourceTender: {
		OpCreate: {},
	},
	ResourceCustomer: {
		OpRead: {},
	},
	ResourceLocation: {
		OpRead: {},
	},
	ResourceBillingQueue: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceDocument: {
		OpRead: {},
		// Update, not create. An agent attaches a document that already exists
		// to the record it belongs to; it has no way to put bytes into storage,
		// so a create grant would be a permission nothing can use.
		OpUpdate: {},
	},
	ResourceAgentRun: {
		OpRead:   {},
		OpCreate: {},
	},
	ResourceAgentProposal: {
		OpRead:   {},
		OpCreate: {},
	},
	ResourceAgentException: {
		OpRead:   {},
		OpCreate: {},
	},
	ResourceAgentMemory: {
		OpRead:   {},
		OpCreate: {},
		OpUpdate: {},
	},
	ResourceInsight: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceAccountingIntegration: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceWatchtower: {
		OpRead: {},
	},
	// Searching and reading public pages, through an extension the
	// organization turned on. What the agent reads there can only lead to a
	// proposal for the rest of its run, never an automatic write.
	ResourceWebResearch: {
		OpRead: {},
	},
	ResourceBriefing: {
		OpRead: {},
	},
	ResourceInvoice: {
		OpRead: {},
	},
	ResourceCustomerPayment: {
		OpRead:   {},
		OpCreate: {},
	},
	ResourceBankReceipt: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceBankReceiptWorkItem: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceWorkerCredential: {
		OpRead:   {},
		OpUpdate: {},
	},
	// Holding a driver off the board is a dispatch decision, kept apart
	// from editing the worker record, which stays closed to an agent.
	ResourceWorkerDispatchHold: {
		OpCreate: {},
	},
	ResourceDriverMessage: {
		OpCreate: {},
	},
	ResourceCustomerCommunication: {
		OpCreate: {},
	},
	// The intake desk reads the inbox, files a message against its records
	// and settles it. Mailboxes stay closed: their addresses, tokens and
	// signing secrets are an administrator's.
	ResourceInboundMessage: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceAccountsReceivable: {
		OpRead: {},
	},
	ResourceJournalEntry: {
		OpRead: {},
	},
	ResourceGeneralLedgerAccount: {
		OpRead: {},
	},
	ResourceFiscalPeriod: {
		OpRead: {},
	},
	ResourceDriverSettlement: {
		OpRead: {},
	},
	ResourceSettlementDispute: {
		OpRead: {},
	},
	ResourceCarrierSettlement: {
		OpRead: {},
	},
	ResourceRateAgreement: {
		OpRead: {},
	},
	ResourceRateMatrix: {
		OpRead: {},
	},
	ResourceFuelSurchargeProgram: {
		OpRead: {},
	},
	ResourceOrder: {
		OpRead: {},
	},
	ResourceEDI: {
		OpRead: {},
	},
}

func IsAgentAllowed(resource Resource, operation Operation) bool {
	operations, ok := agentAllowedPermissions[resource]
	if !ok {
		return false
	}

	_, ok = operations[operation]
	return ok
}

func AgentAllowedPermissions() map[Resource][]Operation {
	out := make(map[Resource][]Operation, len(agentAllowedPermissions))
	for resource, operations := range agentAllowedPermissions {
		ops := make([]Operation, 0, len(operations))
		for operation := range operations {
			ops = append(ops, operation)
		}
		out[resource] = ops
	}

	return out
}
