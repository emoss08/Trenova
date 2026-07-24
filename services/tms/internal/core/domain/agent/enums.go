package agent

type Type string

const (
	TypeBillingException = Type("BillingException")
)

func (t Type) IsValid() bool {
	switch t {
	case TypeBillingException:
		return true
	default:
		return false
	}
}

type SubjectType string

const (
	SubjectBillingQueueItem = SubjectType("BillingQueueItem")
)

func (s SubjectType) IsValid() bool {
	switch s {
	case SubjectBillingQueueItem:
		return true
	default:
		return false
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
)

func (s ProposalStatus) IsValid() bool {
	switch s {
	case ProposalStatusPending,
		ProposalStatusAccepted,
		ProposalStatusModified,
		ProposalStatusRejected,
		ProposalStatusExpired,
		ProposalStatusSuperseded:
		return true
	default:
		return false
	}
}

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
