package driversettlement

type ReleaseBlock string

const (
	ReleaseBlockHeld              = ReleaseBlock("Held")
	ReleaseBlockCriticalException = ReleaseBlock("CriticalException")
	ReleaseBlockDisputeOpen       = ReleaseBlock("DisputeOpen")
)

func ReleaseBlocks() []ReleaseBlock {
	return []ReleaseBlock{
		ReleaseBlockHeld,
		ReleaseBlockCriticalException,
		ReleaseBlockDisputeOpen,
	}
}

func (b ReleaseBlock) String() string { return string(b) }

func (b ReleaseBlock) IsValid() bool {
	switch b {
	case ReleaseBlockHeld, ReleaseBlockCriticalException, ReleaseBlockDisputeOpen:
		return true
	default:
		return false
	}
}

func UnreleasedStatuses() []Status {
	return []Status{StatusDraft, StatusPendingApproval, StatusApproved}
}

func ReleasedStatuses() []Status {
	return []Status{StatusPosted, StatusPaid}
}

func OpenDisputeStatuses() []DisputeStatus {
	return []DisputeStatus{DisputeStatusOpen, DisputeStatusInReview}
}
