package manualjournal

type Status string

const (
	StatusDraft           = Status("Draft")
	StatusPendingApproval = Status("PendingApproval")
	StatusApproved        = Status("Approved")
	StatusRejected        = Status("Rejected")
	StatusCancelled       = Status("Cancelled")
	StatusPosted          = Status("Posted")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsEditable() bool {
	return s == StatusDraft
}

func (s Status) CanSubmit() bool {
	return s == StatusDraft
}

func (s Status) CanApprove() bool {
	return s == StatusPendingApproval
}

func (s Status) CanReject() bool {
	return s == StatusPendingApproval
}

func (s Status) CanCancel() bool {
	switch s {
	case StatusDraft, StatusPendingApproval, StatusApproved:
		return true
	default:
		return false
	}
}
