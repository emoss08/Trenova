package agent

import "github.com/emoss08/trenova/internal/core/domain/permission"

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
	SubjectReport          = SubjectType("Report")
	SubjectDashboard       = SubjectType("Dashboard")
	SubjectFormulaTemplate = SubjectType("FormulaTemplate")
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
		return permission.ResourceDetentionPolicy, true
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
	case SubjectFormulaTemplate:
		return permission.ResourceFormulaTemplate, true
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
)

func (t RunTrigger) IsValid() bool {
	switch t {
	case RunTriggerManual,
		RunTriggerChat,
		RunTriggerScheduled,
		RunTriggerEvent,
		RunTriggerContinuous:
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
		SubjectFormulaTemplate:
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
		SubjectFormulaTemplate,
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
	ProposalStatusPending    = ProposalStatus("Pending")
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

type PlanStatus string

const (
	PlanStatusPending   = PlanStatus("Pending")
	PlanStatusApproved  = PlanStatus("Approved")
	PlanStatusCompleted = PlanStatus("Completed")
	PlanStatusFailed    = PlanStatus("Failed")
	PlanStatusRejected  = PlanStatus("Rejected")
	PlanStatusExpired   = PlanStatus("Expired")
)

func (s PlanStatus) IsValid() bool {
	switch s {
	case PlanStatusPending,
		PlanStatusApproved,
		PlanStatusCompleted,
		PlanStatusFailed,
		PlanStatusRejected,
		PlanStatusExpired:
		return true
	default:
		return false
	}
}

// Decidable reports whether a plan is still waiting on a person.
func (s PlanStatus) Decidable() bool { return s == PlanStatusPending }

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

type Severity string

const (
	SeverityLow      = Severity("Low")
	SeverityMedium   = Severity("Medium")
	SeverityHigh     = Severity("High")
	SeverityCritical = Severity("Critical")
)

func (s Severity) IsValid() bool {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	default:
		return false
	}
}

type ResolutionState string

const (
	ResolutionStateOpen      = ResolutionState("Open")
	ResolutionStateInReview  = ResolutionState("InReview")
	ResolutionStateResolved  = ResolutionState("Resolved")
	ResolutionStateDismissed = ResolutionState("Dismissed")
)

func (s ResolutionState) IsValid() bool {
	switch s {
	case ResolutionStateOpen,
		ResolutionStateInReview,
		ResolutionStateResolved,
		ResolutionStateDismissed:
		return true
	default:
		return false
	}
}

type DecisionType string

const (
	DecisionAccepted = DecisionType("Accepted")
	DecisionModified = DecisionType("Modified")
	DecisionRejected = DecisionType("Rejected")
)

func (d DecisionType) IsValid() bool {
	switch d {
	case DecisionAccepted, DecisionModified, DecisionRejected:
		return true
	default:
		return false
	}
}

type ExceptionCategory string

const (
	CategoryMissingDocumentation       = ExceptionCategory("MissingDocumentation")
	CategoryIncorrectRates             = ExceptionCategory("IncorrectRates")
	CategoryWeightDiscrepancy          = ExceptionCategory("WeightDiscrepancy")
	CategoryAccessorialDispute         = ExceptionCategory("AccessorialDispute")
	CategoryDuplicateCharge            = ExceptionCategory("DuplicateCharge")
	CategoryMissingReferenceNumber     = ExceptionCategory("MissingReferenceNumber")
	CategoryCustomerInformationError   = ExceptionCategory("CustomerInformationError")
	CategoryServiceFailure             = ExceptionCategory("ServiceFailure")
	CategoryRateNotOnFile              = ExceptionCategory("RateNotOnFile")
	CategoryMissingBOL                 = ExceptionCategory("MissingBOL")
	CategoryRateMissingBasis           = ExceptionCategory("RateMissingBasis")
	CategoryRateVarianceRequiresAction = ExceptionCategory("RateVarianceRequiresAction")
	CategoryUnresolvedServiceFailures  = ExceptionCategory("UnresolvedServiceFailures")
	CategoryMissingRequiredDocument    = ExceptionCategory("MissingRequiredDocument")
	CategoryConfidenceBelowThreshold   = ExceptionCategory("ConfidenceBelowThreshold")
	CategoryUnableToDiagnose           = ExceptionCategory("UnableToDiagnose")
	CategoryOther                      = ExceptionCategory("Other")
)

func AllExceptionCategories() []ExceptionCategory {
	return []ExceptionCategory{
		CategoryMissingDocumentation,
		CategoryIncorrectRates,
		CategoryWeightDiscrepancy,
		CategoryAccessorialDispute,
		CategoryDuplicateCharge,
		CategoryMissingReferenceNumber,
		CategoryCustomerInformationError,
		CategoryServiceFailure,
		CategoryRateNotOnFile,
		CategoryMissingBOL,
		CategoryRateMissingBasis,
		CategoryRateVarianceRequiresAction,
		CategoryUnresolvedServiceFailures,
		CategoryMissingRequiredDocument,
		CategoryConfidenceBelowThreshold,
		CategoryUnableToDiagnose,
		CategoryOther,
	}
}

func (c ExceptionCategory) IsValid() bool {
	switch c {
	case CategoryMissingDocumentation,
		CategoryIncorrectRates,
		CategoryWeightDiscrepancy,
		CategoryAccessorialDispute,
		CategoryDuplicateCharge,
		CategoryMissingReferenceNumber,
		CategoryCustomerInformationError,
		CategoryServiceFailure,
		CategoryRateNotOnFile,
		CategoryMissingBOL,
		CategoryRateMissingBasis,
		CategoryRateVarianceRequiresAction,
		CategoryUnresolvedServiceFailures,
		CategoryMissingRequiredDocument,
		CategoryConfidenceBelowThreshold,
		CategoryUnableToDiagnose,
		CategoryOther:
		return true
	default:
		return false
	}
}

func ExceptionCategoryFromValidationCode(code string) ExceptionCategory {
	switch code {
	case "missing_bol":
		return CategoryMissingBOL
	case "rate_missing_basis":
		return CategoryRateMissingBasis
	case "rate_variance_requires_action":
		return CategoryRateVarianceRequiresAction
	case "unresolved_service_failures":
		return CategoryUnresolvedServiceFailures
	case "missing_required_document":
		return CategoryMissingRequiredDocument
	default:
		return CategoryOther
	}
}
