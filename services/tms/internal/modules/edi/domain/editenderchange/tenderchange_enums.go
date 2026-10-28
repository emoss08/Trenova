package editenderchange

type TenderRecipientKind string

const (
	TenderRecipientKindInternal = TenderRecipientKind("Internal")
	TenderRecipientKindExternal = TenderRecipientKind("External")
)

type TenderRecipientStatus string

const (
	TenderRecipientStatusActive = TenderRecipientStatus("Active")
	TenderRecipientStatusClosed = TenderRecipientStatus("Closed")
)

type TenderRecipientBaselineStatus string

const (
	TenderRecipientBaselineStatusSent     = TenderRecipientBaselineStatus("Sent")
	TenderRecipientBaselineStatusAccepted = TenderRecipientBaselineStatus("Accepted")
)

type TenderChangeStatus string

const (
	TenderChangeStatusPendingReview = TenderChangeStatus("PendingReview")
	TenderChangeStatusApplied       = TenderChangeStatus("Applied")
	TenderChangeStatusRejected      = TenderChangeStatus("Rejected")
	TenderChangeStatusQueued        = TenderChangeStatus("Queued")
	TenderChangeStatusSent          = TenderChangeStatus("Sent")
	TenderChangeStatusFailed        = TenderChangeStatus("Failed")
	TenderChangeStatusIgnored       = TenderChangeStatus("Ignored")
	TenderChangeStatusSuperseded    = TenderChangeStatus("Superseded")
)

const TenderChangeTypeLoadTender = "LoadTenderChange"

func (k TenderRecipientKind) IsValid() bool {
	switch k {
	case TenderRecipientKindInternal,
		TenderRecipientKindExternal:
		return true
	default:
		return false
	}
}

func (s TenderRecipientStatus) IsValid() bool {
	switch s {
	case TenderRecipientStatusActive,
		TenderRecipientStatusClosed:
		return true
	default:
		return false
	}
}

func (s TenderRecipientBaselineStatus) IsValid() bool {
	switch s {
	case TenderRecipientBaselineStatusSent,
		TenderRecipientBaselineStatusAccepted:
		return true
	default:
		return false
	}
}

func (s TenderChangeStatus) IsValid() bool {
	switch s {
	case TenderChangeStatusPendingReview,
		TenderChangeStatusApplied,
		TenderChangeStatusRejected,
		TenderChangeStatusQueued,
		TenderChangeStatusSent,
		TenderChangeStatusFailed,
		TenderChangeStatusIgnored,
		TenderChangeStatusSuperseded:
		return true
	default:
		return false
	}
}
