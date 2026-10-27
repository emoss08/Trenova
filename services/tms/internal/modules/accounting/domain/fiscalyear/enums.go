package fiscalyear

type Status string

const (
	// Being set up; nothing can be posted to any of its periods.
	StatusDraft = Status("Draft")

	// Active and accepting transactions.
	StatusOpen = Status("Open")

	// Year-end close is done; it can be reopened with authorization and a
	// reason, which is audited.
	StatusClosed = Status("Closed")

	// Closed for good once the audit is signed off; it can never be reopened.
	StatusPermanentlyClosed = Status("PermanentlyClosed")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusDraft,
		StatusOpen,
		StatusClosed,
		StatusPermanentlyClosed:
		return true
	default:
		return false
	}
}

func (s Status) IsClosed() bool {
	return s == StatusClosed || s == StatusPermanentlyClosed
}
