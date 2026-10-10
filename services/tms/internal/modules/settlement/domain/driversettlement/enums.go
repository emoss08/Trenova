package driversettlement

type Status string

const (
	StatusDraft           = Status("Draft")
	StatusPendingApproval = Status("PendingApproval")
	StatusApproved        = Status("Approved")
	StatusPosted          = Status("Posted")
	StatusPaid            = Status("Paid")
	StatusVoided          = Status("Voided")
)

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	switch s {
	case StatusDraft, StatusPendingApproval, StatusApproved, StatusPosted,
		StatusPaid, StatusVoided:
		return true
	default:
		return false
	}
}

func (s Status) IsTerminal() bool {
	return s == StatusPaid || s == StatusVoided
}

type LineCategory string

const (
	LineCategoryEarning            = LineCategory("Earning")
	LineCategoryReimbursement      = LineCategory("Reimbursement")
	LineCategoryDeduction          = LineCategory("Deduction")
	LineCategoryAdvanceRecovery    = LineCategory("AdvanceRecovery")
	LineCategoryEscrowContribution = LineCategory("EscrowContribution")
	LineCategoryGuaranteeTopUp     = LineCategory("GuaranteeTopUp")
	LineCategoryCarryForward       = LineCategory("CarryForward")
	LineCategoryAdjustment         = LineCategory("Adjustment")
)

func (l LineCategory) String() string { return string(l) }

func (l LineCategory) IsValid() bool {
	switch l {
	case LineCategoryEarning, LineCategoryReimbursement, LineCategoryDeduction,
		LineCategoryAdvanceRecovery, LineCategoryEscrowContribution,
		LineCategoryGuaranteeTopUp, LineCategoryCarryForward, LineCategoryAdjustment:
		return true
	default:
		return false
	}
}

func (l LineCategory) IsCredit() bool {
	switch l { //nolint:exhaustive // only credit categories matter; default covers the rest
	case LineCategoryEarning, LineCategoryReimbursement, LineCategoryGuaranteeTopUp:
		return true
	default:
		return false
	}
}

type BatchStatus string

const (
	BatchStatusOpen      = BatchStatus("Open")
	BatchStatusCompleted = BatchStatus("Completed")
	BatchStatusCanceled  = BatchStatus("Canceled")
)

func (b BatchStatus) String() string { return string(b) }

func (b BatchStatus) IsValid() bool {
	switch b {
	case BatchStatusOpen, BatchStatusCompleted, BatchStatusCanceled:
		return true
	default:
		return false
	}
}

type PayEventStatus string

const (
	PayEventStatusAccrued = PayEventStatus("Accrued")
	PayEventStatusSettled = PayEventStatus("Settled")
	PayEventStatusVoided  = PayEventStatus("Voided")
)

func (p PayEventStatus) String() string { return string(p) }

func (p PayEventStatus) IsValid() bool {
	switch p {
	case PayEventStatusAccrued, PayEventStatusSettled, PayEventStatusVoided:
		return true
	default:
		return false
	}
}

type ExceptionCode string

const (
	ExceptionCodeNegativeNet       = ExceptionCode("NegativeNet")
	ExceptionCodeHighVariance      = ExceptionCode("HighVariance")
	ExceptionCodeNoActivity        = ExceptionCode("NoActivity")
	ExceptionCodeMissingPayProfile = ExceptionCode("MissingPayProfile")
	ExceptionCodeGuaranteeApplied  = ExceptionCode("GuaranteeApplied")
	ExceptionCodeDeductionCapped   = ExceptionCode("DeductionCapped")
	ExceptionCodeEarningCapped     = ExceptionCode("EarningCapped")
	ExceptionCodeManualAdjustment  = ExceptionCode("ManualAdjustment")
)

func (e ExceptionCode) String() string { return string(e) }

type ExceptionSeverity string

const (
	ExceptionSeverityWarning  = ExceptionSeverity("Warning")
	ExceptionSeverityCritical = ExceptionSeverity("Critical")
)

func (e ExceptionSeverity) String() string { return string(e) }
