package cloudsignup

type Status string

const (
	StatusPending     = Status("pending")
	StatusProvisioned = Status("provisioned")
	StatusExpired     = Status("expired")
	StatusRejected    = Status("rejected")
)

func AllStatuses() []Status {
	return []Status{StatusPending, StatusProvisioned, StatusExpired, StatusRejected}
}

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusProvisioned, StatusExpired, StatusRejected:
		return true
	default:
		return false
	}
}

func (s Status) IsTerminal() bool {
	return s == StatusProvisioned || s == StatusExpired || s == StatusRejected
}
