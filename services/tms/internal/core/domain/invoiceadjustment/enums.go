package invoiceadjustment

type Kind string

const (
	KindCreditOnly   = Kind("CreditOnly")
	KindCreditRebill = Kind("CreditAndRebill")
	KindFullReversal = Kind("FullReversal")
	KindWriteOff     = Kind("WriteOff")
)

type Status string

const (
	StatusDraft           = Status("Draft")
	StatusPendingApproval = Status("PendingApproval")
	StatusApproved        = Status("Approved")
	StatusRejected        = Status("Rejected")
	StatusExecuting       = Status("Executing")
	StatusExecuted        = Status("Executed")
	StatusExecutionFailed = Status("ExecutionFailed")
)

type RebillStrategy string

const (
	RebillStrategyCloneExact = RebillStrategy("CloneExact")
	RebillStrategyRerate     = RebillStrategy("Rerate")
	RebillStrategyManual     = RebillStrategy("Manual")
)

type SnapshotKind string

const (
	SnapshotKindSubmission = SnapshotKind("Submission")
	SnapshotKindExecution  = SnapshotKind("Execution")
)

type ApprovalStatus string

const (
	ApprovalStatusNotRequired = ApprovalStatus("NotRequired")
	ApprovalStatusPending     = ApprovalStatus("Pending")
	ApprovalStatusApproved    = ApprovalStatus("Approved")
	ApprovalStatusRejected    = ApprovalStatus("Rejected")
)

type ReplacementReviewStatus string

const (
	ReplacementReviewStatusNotRequired = ReplacementReviewStatus("NotRequired")
	ReplacementReviewStatusRequired    = ReplacementReviewStatus("Required")
	ReplacementReviewStatusCompleted   = ReplacementReviewStatus("Completed")
)

type ExceptionStatus string

const (
	ExceptionStatusOpen     = ExceptionStatus("Open")
	ExceptionStatusResolved = ExceptionStatus("Resolved")
)

type BatchStatus string

const (
	BatchStatusPending   = BatchStatus("Pending")
	BatchStatusRunning   = BatchStatus("Running")
	BatchStatusCompleted = BatchStatus("Completed")
	BatchStatusFailed    = BatchStatus("Failed")
	BatchStatusPartial   = BatchStatus("PartialSuccess")
	BatchStatusSubmitted = BatchStatus("Submitted")
	BatchStatusQueued    = BatchStatus("Queued")
)

type BatchItemStatus string

const (
	BatchItemStatusPending         = BatchItemStatus("Pending")
	BatchItemStatusPreviewed       = BatchItemStatus("Previewed")
	BatchItemStatusSubmitted       = BatchItemStatus("Submitted")
	BatchItemStatusPendingApproval = BatchItemStatus("PendingApproval")
	BatchItemStatusExecuting       = BatchItemStatus("Executing")
	BatchItemStatusExecuted        = BatchItemStatus("Executed")
	BatchItemStatusRejected        = BatchItemStatus("Rejected")
	BatchItemStatusFailed          = BatchItemStatus("Failed")
)

type SupportingDocumentPolicySource string

const (
	SupportingDocumentPolicySourceCustomerBillingProfile = SupportingDocumentPolicySource(
		"CustomerBillingProfile",
	)
	SupportingDocumentPolicySourceOrganizationControl = SupportingDocumentPolicySource(
		"OrganizationControl",
	)
	SupportingDocumentPolicySourceDefaultOptional = SupportingDocumentPolicySource(
		"DefaultOptional",
	)
)

func (v Kind) IsValid() bool {
	switch v {
	case KindCreditOnly,
		KindCreditRebill,
		KindFullReversal,
		KindWriteOff:
		return true
	default:
		return false
	}
}

func (v Status) IsValid() bool {
	switch v {
	case StatusDraft,
		StatusPendingApproval,
		StatusApproved,
		StatusRejected,
		StatusExecuting,
		StatusExecuted,
		StatusExecutionFailed:
		return true
	default:
		return false
	}
}

func (v RebillStrategy) IsValid() bool {
	switch v {
	case RebillStrategyCloneExact,
		RebillStrategyRerate,
		RebillStrategyManual:
		return true
	default:
		return false
	}
}

func (v SnapshotKind) IsValid() bool {
	switch v {
	case SnapshotKindSubmission,
		SnapshotKindExecution:
		return true
	default:
		return false
	}
}

func (v ApprovalStatus) IsValid() bool {
	switch v {
	case ApprovalStatusNotRequired,
		ApprovalStatusPending,
		ApprovalStatusApproved,
		ApprovalStatusRejected:
		return true
	default:
		return false
	}
}

func (v ReplacementReviewStatus) IsValid() bool {
	switch v {
	case ReplacementReviewStatusNotRequired,
		ReplacementReviewStatusRequired,
		ReplacementReviewStatusCompleted:
		return true
	default:
		return false
	}
}

func (v ExceptionStatus) IsValid() bool {
	switch v {
	case ExceptionStatusOpen,
		ExceptionStatusResolved:
		return true
	default:
		return false
	}
}

func (v BatchStatus) IsValid() bool {
	switch v {
	case BatchStatusPending,
		BatchStatusRunning,
		BatchStatusCompleted,
		BatchStatusFailed,
		BatchStatusPartial,
		BatchStatusSubmitted,
		BatchStatusQueued:
		return true
	default:
		return false
	}
}

func (v BatchItemStatus) IsValid() bool {
	switch v {
	case BatchItemStatusPending,
		BatchItemStatusPreviewed,
		BatchItemStatusSubmitted,
		BatchItemStatusPendingApproval,
		BatchItemStatusExecuting,
		BatchItemStatusExecuted,
		BatchItemStatusRejected,
		BatchItemStatusFailed:
		return true
	default:
		return false
	}
}

func (v SupportingDocumentPolicySource) IsValid() bool {
	switch v {
	case SupportingDocumentPolicySourceCustomerBillingProfile,
		SupportingDocumentPolicySourceOrganizationControl,
		SupportingDocumentPolicySourceDefaultOptional:
		return true
	default:
		return false
	}
}
