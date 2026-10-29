package agentceiling

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

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
var agentAllowedPermissions = map[permission.Resource]map[permission.Operation]struct{}{ //nolint:exhaustive // an allow-list: a missing resource is refused
	// The factoring clerk reads the agreements it works under and drafts,
	// sends, records and releases notices of assignment. Agreement terms are
	// copied from the signed contract by a person, so writing them is not
	// an agent's.
	permission.ResourceFactoringAgreement: {
		permission.OpRead: {},
	},
	permission.ResourceFactoringNoa: {
		permission.OpRead:   {},
		permission.OpCreate: {},
		permission.OpUpdate: {},
	},
	// The clerk proposes each schedule and each retry or cancellation of a
	// failed one; a person approves every one of them before it runs.
	permission.ResourceFactoringSubmission: {
		permission.OpRead:   {},
		permission.OpSubmit: {},
		permission.OpUpdate: {},
		permission.OpCancel: {},
	},
	// The clerk proposes recording a factor's funding and releasing a reserve;
	// both book to the ledger, so each runs only from a person's approval.
	permission.ResourceFactoringReserve: {
		permission.OpRead:   {},
		permission.OpCreate: {},
	},
	// The clerk resolves remittance lines it is sure of and proposes applying
	// the matched ones; applying books to the ledger, so it runs only from a
	// person's approval.
	permission.ResourceFactoringRemittance: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
		permission.OpImport: {},
		permission.OpCancel: {},
	},
	// Creating a shipment from a document is the intake desk's whole job.
	// Editing or cancelling one that already exists is a person's: those
	// tools are offered in chat, where the actor is the person asking.
	permission.ResourceShipment: {
		permission.OpRead:   {},
		permission.OpCreate: {},
	},
	permission.ResourceShipmentComment: {
		permission.OpRead:   {},
		permission.OpCreate: {},
	},
	permission.ResourceShipmentHold: {
		permission.OpRead:   {},
		permission.OpCreate: {},
		permission.OpUpdate: {},
	},
	permission.ResourceShipmentMove: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
		// The coverage desk may put a contract carrier on a move no driver
		// fits. Unassigning stays a person's: no desk runs dispatch_unassign_move or
		// dispatch_cancel_carrier_assignment.
		permission.OpAssign: {},
	},
	permission.ResourceWorker: {
		permission.OpRead: {},
	},
	permission.ResourceTractor: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	permission.ResourceTrailer: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	permission.ResourceWorkerPTO: {
		permission.OpRead:   {},
		permission.OpCancel: {},
		permission.OpReject: {},
	},
	permission.ResourceReport: {
		permission.OpRead:   {},
		permission.OpCreate: {},
		permission.OpUpdate: {},
	},
	permission.ResourceServiceFailure: {
		permission.OpRead:   {},
		permission.OpCreate: {},
		permission.OpUpdate: {},
		// The service failure desk may propose voiding a failure it finds was
		// opened in error; exception_void_service_failure never runs until a person
		// approves the proposal.
		permission.OpArchive: {},
	},
	// The three surfaces an agent builds rather than reads: a saved view of a
	// table, a dashboard, and an alert that watches for a change. All three
	// are additive and reversible — nothing an agent makes here alters a
	// record, and a person can delete any of it.
	// No read on the other two: nothing lists them. A grant no tool claims is
	// one nobody can account for, which is what the coverage test holds.
	permission.ResourceTableConfiguration: {
		permission.OpCreate: {},
	},
	permission.ResourceDashboard: {
		// report_list_dashboards claims the read: report_add_dashboard_tile needs an id and
		// nothing else hands one out.
		permission.OpRead:   {},
		permission.OpCreate: {},
		permission.OpUpdate: {},
	},
	permission.ResourceTableChangeAlert: {
		permission.OpCreate: {},
	},
	permission.ResourceCarrier: {
		permission.OpRead: {},
	},
	permission.ResourceCarrierCapacityPosting: {
		permission.OpRead:   {},
		permission.OpCreate: {},
		permission.OpUpdate: {},
	},
	permission.ResourceCarrierIntelligence: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	// The reference data every list and get tool resolves a name against.
	permission.ResourceAccessorialCharge: {
		permission.OpRead: {},
	},
	permission.ResourceCommodity: {
		permission.OpRead: {},
	},
	permission.ResourceDocumentType: {
		permission.OpRead: {},
	},
	permission.ResourceEquipmentType: {
		permission.OpRead: {},
	},
	permission.ResourceFleetCode: {
		permission.OpRead: {},
	},
	permission.ResourceHazardousMaterial: {
		permission.OpRead: {},
	},
	permission.ResourceHoldReason: {
		permission.OpRead: {},
	},
	permission.ResourceLocationCategory: {
		permission.OpRead: {},
	},
	permission.ResourceServiceType: {
		permission.OpRead: {},
	},
	permission.ResourceShipmentType: {
		permission.OpRead: {},
	},
	permission.ResourceServiceFailureReasonCode: {
		permission.OpRead: {},
	},
	permission.ResourceDetention: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	permission.ResourceEmailProfile: {
		permission.OpRead: {},
	},
	permission.ResourceRateQuote: {
		permission.OpRead: {},
	},
	permission.ResourceTender: {
		// dispatch_list_tenders, so the coverage desk sees a live tender
		// before it covers a move another way.
		permission.OpRead:   {},
		permission.OpCreate: {},
	},
	// The coverage desk generates the rate confirmation for a carrier it
	// covered a move with and proposes sending it; rate_send_confirmation
	// never runs without a person, whatever the desk's tier. Voiding and
	// recording a carrier's confirmation share the update grant, and neither
	// runs before a person approves it either.
	permission.ResourceRateConfirmation: {
		permission.OpRead:   {},
		permission.OpCreate: {},
		permission.OpUpdate: {},
	},
	permission.ResourceCustomer: {
		permission.OpRead: {},
	},
	permission.ResourceLocation: {
		permission.OpRead:   {},
		permission.OpCreate: {},
	},
	permission.ResourceBillingQueue: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	permission.ResourceDocument: {
		permission.OpRead: {},
		// Update, not create. An agent attaches a document that already exists
		// to the record it belongs to; it has no way to put bytes into storage,
		// so a create grant would be a permission nothing can use.
		permission.OpUpdate: {},
	},
	permission.ResourceAgentRun: {
		permission.OpRead:   {},
		permission.OpCreate: {},
	},
	permission.ResourceAgentProposal: {
		permission.OpRead:   {},
		permission.OpCreate: {},
	},
	permission.ResourceAgentException: {
		permission.OpRead:   {},
		permission.OpCreate: {},
	},
	// Memory and profile guidance are used under the assistant, as on the
	// Desk: a run reads what its reader may read and offers what it would keep
	// for the organization, which an administrator approves.
	permission.ResourceAssistant: {
		permission.OpRead:   {},
		permission.OpCreate: {},
	},
	permission.ResourceInsight: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	permission.ResourceAccountingIntegration: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	// Reading the sync ledger, and retrying, skipping, pausing and resuming
	// what it sends. Connecting, disconnecting and backfilling are manage on
	// the integration, which stays a person's.
	permission.ResourceAccountingSync: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	// The load monitor works the exceptions on the loads it watches: it notes
	// what a carrier said, snoozes what cannot move yet, routes an exception to
	// its owner and proposes closing what is dealt with. Each write starts as a
	// proposal a person approves.
	permission.ResourceWatchtower: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	// Searching and reading public pages, through an extension the
	// organization turned on. What the agent reads there can only lead to a
	// proposal for the rest of its run, never an automatic write.
	permission.ResourceWebResearch: {
		permission.OpRead: {},
	},
	// Reading the portals an administrator set up, and acting on one. What an
	// agent reads there holds every later write for a person, and acting on a
	// portal is always a proposal a person approves.
	permission.ResourcePortalAutomation: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	// Running a short script whose every read is one of the agent's own reads,
	// checked as the person, over the reports they ran and the files they
	// attached, in an interpreter that reaches no file or network. It changes
	// nothing.
	permission.ResourceCodeMode: {
		permission.OpRead: {},
	},
	permission.ResourceBriefing: {
		permission.OpRead: {},
	},
	permission.ResourceInvoice: {
		permission.OpRead: {},
	},
	permission.ResourceCustomerPayment: {
		permission.OpRead:   {},
		permission.OpCreate: {},
	},
	permission.ResourceBankReceipt: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	permission.ResourceBankReceiptWorkItem: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	permission.ResourceWorkerCredential: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	// Holding a driver off the board is a dispatch decision, kept apart
	// from editing the worker record, which stays closed to an agent.
	permission.ResourceWorkerDispatchHold: {
		permission.OpCreate: {},
	},
	permission.ResourceDriverMessage: {
		permission.OpCreate: {},
	},
	permission.ResourceCustomerCommunication: {
		permission.OpCreate: {},
	},
	// The intake desk reads the inbox, files a message against its records
	// and settles it, and proposes who owns a message and the end-of-day
	// hand-off to the on-call person; both assign tools stop at Ask first.
	// Mailboxes stay closed: their addresses, tokens and signing secrets are
	// an administrator's.
	permission.ResourceInboundMessage: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
		permission.OpAssign: {},
	},
	permission.ResourceAccountsReceivable: {
		permission.OpRead: {},
	},
	permission.ResourceJournalEntry: {
		permission.OpRead: {},
	},
	permission.ResourceGeneralLedgerAccount: {
		permission.OpRead: {},
	},
	// The books keeper proposes locking and then closing a period whose
	// blockers are clear. Both tools stop at a proposal and run only as the
	// person who approves it; reopening, unlocking and opening a period are
	// offered to no desk.
	permission.ResourceFiscalPeriod: {
		permission.OpRead:  {},
		permission.OpLock:  {},
		permission.OpClose: {},
	},
	permission.ResourceDriverSettlement: {
		permission.OpRead: {},
	},
	permission.ResourceSettlementDispute: {
		permission.OpRead: {},
	},
	permission.ResourceCarrierSettlement: {
		permission.OpRead: {},
	},
	permission.ResourceRateAgreement: {
		permission.OpRead: {},
	},
	permission.ResourceRateMatrix: {
		permission.OpRead: {},
	},
	permission.ResourceFuelSurchargeProgram: {
		permission.OpRead: {},
	},
	permission.ResourceOrder: {
		permission.OpRead: {},
	},
	permission.ResourceEDI: {
		permission.OpRead:   {},
		permission.OpUpdate: {},
	},
	permission.ResourceFormulaTemplate: {
		permission.OpRead: {},
	},
	permission.ResourceFormulaLibrary: {
		permission.OpRead: {},
	},
}

func Allows(resource permission.Resource, operation permission.Operation) bool {
	operations, ok := agentAllowedPermissions[resource]
	if !ok {
		return false
	}

	_, ok = operations[operation]
	return ok
}

func Permissions() map[permission.Resource][]permission.Operation {
	out := make(map[permission.Resource][]permission.Operation, len(agentAllowedPermissions))
	for resource, operations := range agentAllowedPermissions {
		ops := make([]permission.Operation, 0, len(operations))
		for operation := range operations {
			ops = append(ops, operation)
		}
		out[resource] = ops
	}

	return out
}
