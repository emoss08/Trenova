package onboarding

type Status string

const (
	StatusPending   = Status("pending")
	StatusCompleted = Status("completed")
)

func AllStatuses() []Status {
	return []Status{StatusPending, StatusCompleted}
}

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusCompleted:
		return true
	default:
		return false
	}
}
