package agent

import "github.com/emoss08/trenova/internal/modules/identity/domain/permission"

type Type string

const (
	TypeBillingException   = Type("BillingException")
	TypeDispatchAssignment = Type("DispatchAssignment")
	// TypeAssistantChat covers proposals raised while someone was talking to the
	// assistant. Its runs are opened inline, since the reasoning happened in the
	// request rather than in a workflow.
	TypeAssistantChat = Type("AssistantChat")
	TypeGeneral       = Type("General")
)

func (t Type) IsValid() bool {
	switch t {
	case TypeBillingException, TypeDispatchAssignment, TypeAssistantChat, TypeGeneral:
		return true
	default:
		return false
	}
}

type SubjectType string

const (
	SubjectBillingQueueItem = SubjectType("BillingQueueItem")
	SubjectShipmentMove     = SubjectType("ShipmentMove")
	SubjectAssistantThread  = SubjectType("AssistantThread")
	SubjectShipment         = SubjectType("Shipment")
	SubjectDocument         = SubjectType("Document")
	SubjectOrganization     = SubjectType("Organization")
	SubjectInsight          = SubjectType("Insight")
	SubjectBankReceipt      = SubjectType("BankReceipt")
	// SubjectDetentionOccurrence is one clock at one stop: the desk's unit of
	// work, not the shipment it hangs off, so two clocks on the same shipment
	// are two subjects and two runs.
	SubjectDetentionOccurrence = SubjectType("DetentionOccurrence")
	// SubjectWorker is the person, not the credential, because a driver with
	// three papers coming due is one conversation with one renewal packet.
	SubjectWorker            = SubjectType("Worker")
	SubjectCarrierIntelEvent = SubjectType("CarrierIntelEvent")
	SubjectEDIInboundFile    = SubjectType("EDIInboundFile")
	// SubjectInboundMessage is the message, not the shipment or customer it
	// turned out to be about: what a desk works on is the piece of mail, and
	// two messages about one shipment are two things to answer.
	SubjectInboundMessage = SubjectType("InboundMessage")
	// SubjectReport is a saved report definition. A report that cannot be
	// built as asked is a case about the report, not about an insight.
	SubjectReport               = SubjectType("Report")
	SubjectDashboard            = SubjectType("Dashboard")
	SubjectAccountingConnection = SubjectType("AccountingConnection")
	SubjectFormulaTemplate      = SubjectType("FormulaTemplate")
	SubjectFormulaLibrary       = SubjectType("FormulaLibrary")
	SubjectAccountingSyncRecord = SubjectType("AccountingSyncRecord")
	SubjectAccountingInbound    = SubjectType("AccountingInboundChange")
	SubjectAccountingDrift      = SubjectType("AccountingDriftFinding")
	SubjectInvoice              = SubjectType("Invoice")
	SubjectInvoiceDispute       = SubjectType("InvoiceDispute")
	SubjectFactoringNoa         = SubjectType("FactoringNoa")
	SubjectFactoringSubmission  = SubjectType("FactoringSubmission")
	SubjectFactoringRemittance  = SubjectType("FactoringRemittance")
)

// Resource is the permission a person needs to read a record of this kind.
// The organization and a person's own conversation are no record of anyone
// else's, so they name none.
func (s SubjectType) Resource() (permission.Resource, bool) {
	switch s {
	case SubjectBillingQueueItem:
		return permission.ResourceBillingQueue, true
	case SubjectShipmentMove:
		return permission.ResourceShipmentMove, true
	case SubjectShipment:
		return permission.ResourceShipment, true
	case SubjectDocument:
		return permission.ResourceDocument, true
	case SubjectInsight:
		return permission.ResourceInsight, true
	case SubjectBankReceipt:
		return permission.ResourceBankReceipt, true
	case SubjectDetentionOccurrence:
		return permission.ResourceDetention, true
	case SubjectWorker:
		return permission.ResourceWorker, true
	case SubjectCarrierIntelEvent:
		return permission.ResourceCarrierIntelligence, true
	case SubjectEDIInboundFile:
		return permission.ResourceEDI, true
	case SubjectInboundMessage:
		return permission.ResourceInboundMessage, true
	case SubjectReport:
		return permission.ResourceReport, true
	case SubjectDashboard:
		return permission.ResourceDashboard, true
	case SubjectAccountingConnection:
		return permission.ResourceAccountingIntegration, true
	case SubjectFormulaTemplate:
		return permission.ResourceFormulaTemplate, true
	case SubjectFormulaLibrary:
		return permission.ResourceFormulaLibrary, true
	case SubjectAccountingSyncRecord, SubjectAccountingInbound, SubjectAccountingDrift:
		return permission.ResourceAccountingSync, true
	case SubjectInvoice:
		return permission.ResourceInvoice, true
	case SubjectInvoiceDispute:
		return permission.ResourceInvoiceDispute, true
	case SubjectFactoringNoa:
		return permission.ResourceFactoringNoa, true
	case SubjectFactoringSubmission:
		return permission.ResourceFactoringSubmission, true
	case SubjectFactoringRemittance:
		return permission.ResourceFactoringRemittance, true
	default:
		return "", false
	}
}

type RunTrigger string

const (
	RunTriggerManual     = RunTrigger("Manual")
	RunTriggerChat       = RunTrigger("Chat")
	RunTriggerScheduled  = RunTrigger("Scheduled")
	RunTriggerEvent      = RunTrigger("Event")
	RunTriggerContinuous = RunTrigger("Continuous")
	// RunTriggerWait is a run that picks up work an earlier run parked on a
	// wait, once what it waited for happened or the wait ran out.
	RunTriggerWait = RunTrigger("Wait")
)

func (t RunTrigger) IsValid() bool {
	switch t {
	case RunTriggerManual,
		RunTriggerChat,
		RunTriggerScheduled,
		RunTriggerEvent,
		RunTriggerContinuous,
		RunTriggerWait:
		return true
	default:
		return false
	}
}

func (s SubjectType) IsValid() bool {
	switch s {
	case SubjectBillingQueueItem,
		SubjectShipmentMove,
		SubjectAssistantThread,
		SubjectShipment,
		SubjectDocument,
		SubjectOrganization,
		SubjectInsight,
		SubjectBankReceipt,
		SubjectDetentionOccurrence,
		SubjectWorker,
		SubjectCarrierIntelEvent,
		SubjectEDIInboundFile,
		SubjectInboundMessage,
		SubjectReport,
		SubjectDashboard,
		SubjectAccountingConnection,
		SubjectFormulaTemplate,
		SubjectFormulaLibrary,
		SubjectAccountingSyncRecord,
		SubjectAccountingInbound,
		SubjectAccountingDrift,
		SubjectInvoice,
		SubjectInvoiceDispute,
		SubjectFactoringNoa,
		SubjectFactoringSubmission,
		SubjectFactoringRemittance:
		return true
	default:
		return false
	}
}

func AllSubjectTypes() []SubjectType {
	return []SubjectType{
		SubjectBillingQueueItem,
		SubjectShipmentMove,
		SubjectAssistantThread,
		SubjectShipment,
		SubjectDocument,
		SubjectOrganization,
		SubjectInsight,
		SubjectBankReceipt,
		SubjectDetentionOccurrence,
		SubjectWorker,
		SubjectCarrierIntelEvent,
		SubjectEDIInboundFile,
		SubjectInboundMessage,
		SubjectReport,
		SubjectDashboard,
		SubjectAccountingConnection,
		SubjectFormulaTemplate,
		SubjectFormulaLibrary,
		SubjectAccountingSyncRecord,
		SubjectAccountingInbound,
		SubjectAccountingDrift,
		SubjectInvoice,
		SubjectInvoiceDispute,
		SubjectFactoringNoa,
		SubjectFactoringSubmission,
		SubjectFactoringRemittance,
	}
}

type RunStatus string

const (
	RunStatusPending          = RunStatus("Pending")
	RunStatusGatheringContext = RunStatus("GatheringContext")
	RunStatusDiagnosing       = RunStatus("Diagnosing")
	RunStatusAwaitingDecision = RunStatus("AwaitingDecision")
	RunStatusCompleted        = RunStatus("Completed")
	RunStatusShadowCompleted  = RunStatus("ShadowCompleted")
	RunStatusFailed           = RunStatus("Failed")
)

func (s RunStatus) IsValid() bool {
	switch s {
	case RunStatusPending,
		RunStatusGatheringContext,
		RunStatusDiagnosing,
		RunStatusAwaitingDecision,
		RunStatusCompleted,
		RunStatusShadowCompleted,
		RunStatusFailed:
		return true
	default:
		return false
	}
}

type ProposalStatus string

const (
	ProposalStatusPending = ProposalStatus("Pending")
	// ProposalStatusApproving is an approval that has not gone through yet:
	// approved from the person's own conversation, it commits a few seconds
	// later unless they undo it first. It is neither waiting on anyone nor
	// decided, so no queue lists it and no sweep expires it.
	ProposalStatusApproving  = ProposalStatus("Approving")
	ProposalStatusAccepted   = ProposalStatus("Accepted")
	ProposalStatusModified   = ProposalStatus("Modified")
	ProposalStatusRejected   = ProposalStatus("Rejected")
	ProposalStatusExpired    = ProposalStatus("Expired")
	ProposalStatusSuperseded = ProposalStatus("Superseded")
	// ProposalStatusExecuted and ProposalStatusExecutionFailed separate "a person
	// approved this" from "the tool actually ran". Without them an accepted
	// proposal is indistinguishable from a completed one, which is exactly the
	// ambiguity that let accepted proposals go unexecuted unnoticed.
	ProposalStatusExecuted        = ProposalStatus("Executed")
	ProposalStatusExecutionFailed = ProposalStatus("ExecutionFailed")
	// ProposalStatusSkipped is a plan step that never ran because a step
	// before it failed. Nobody decided against it and it did not expire.
	ProposalStatusSkipped = ProposalStatus("Skipped")
	// ProposalStatusSimulated is a write that was cleared to run while its
	// agent was in simulation: previewed and recorded, never made.
	ProposalStatusSimulated = ProposalStatus("Simulated")
)

func (s ProposalStatus) IsValid() bool {
	switch s {
	case ProposalStatusPending,
		ProposalStatusApproving,
		ProposalStatusAccepted,
		ProposalStatusModified,
		ProposalStatusRejected,
		ProposalStatusExpired,
		ProposalStatusSuperseded,
		ProposalStatusExecuted,
		ProposalStatusExecutionFailed,
		ProposalStatusSkipped,
		ProposalStatusSimulated:
		return true
	default:
		return false
	}
}

// Expirable reports whether the sweep may close a proposal's decision
// window. Only one still waiting on a person can expire: an approval in its
// undo window was decided inside its window and commits whatever the clock
// says now.
func (s ProposalStatus) Expirable() bool { return s == ProposalStatusPending }

type AutonomyTier string

const (
	TierPropose         = AutonomyTier("Propose")
	TierActWithApproval = AutonomyTier("ActWithApproval")
	TierAutoExecute     = AutonomyTier("AutoExecute")
)

func (t AutonomyTier) IsValid() bool {
	switch t {
	case TierPropose, TierActWithApproval, TierAutoExecute:
		return true
	default:
		return false
	}
}

// Rank orders the tiers by how much the agent may do on its own. An unknown
// tier ranks with Propose, the least an agent can be trusted with.
func (t AutonomyTier) Rank() int {
	switch t {
	case TierActWithApproval:
		return 1
	case TierAutoExecute:
		return 2
	default:
		return 0
	}
}

// Above reports whether t lets an agent do more on its own than other.
func (t AutonomyTier) Above(other AutonomyTier) bool { return t.Rank() > other.Rank() }

// AtMost is t, held down to limit when limit allows less.
func (t AutonomyTier) AtMost(limit AutonomyTier) AutonomyTier {
	if t.Above(limit) {
		return limit
	}

	return t
}

// Next is the tier one step up, and false from the top.
func (t AutonomyTier) Next() (AutonomyTier, bool) {
	switch t {
	case TierPropose:
		return TierActWithApproval, true
	case TierActWithApproval:
		return TierAutoExecute, true
	default:
		return "", false
	}
}

// Previous is the tier one step down, and false from the bottom.
func (t AutonomyTier) Previous() (AutonomyTier, bool) {
	switch t {
	case TierAutoExecute:
		return TierActWithApproval, true
	case TierActWithApproval:
		return TierPropose, true
	default:
		return "", false
	}
}

func (t AutonomyTier) Label() string {
	switch t {
	case TierActWithApproval:
		return "Ask first"
	case TierAutoExecute:
		return "Automatic"
	default:
		return "Propose"
	}
}
