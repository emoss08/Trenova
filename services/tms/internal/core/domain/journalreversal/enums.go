package journalreversal

type Status string

const (
	StatusRequested       = Status("Requested")
	StatusPendingApproval = Status("PendingApproval")
	StatusApproved        = Status("Approved")
	StatusRejected        = Status("Rejected")
	StatusCancelled       = Status("Cancelled")
	StatusPosted          = Status("Posted")
)

func (s Status) String() string { return string(s) }

func (s Status) CanApprove() bool { return s == StatusPendingApproval || s == StatusRequested }
func (s Status) CanReject() bool  { return s == StatusPendingApproval || s == StatusRequested }
func (s Status) CanCancel() bool {
	return s == StatusRequested || s == StatusPendingApproval || s == StatusApproved
}
func (s Status) CanPost() bool { return s == StatusApproved }
