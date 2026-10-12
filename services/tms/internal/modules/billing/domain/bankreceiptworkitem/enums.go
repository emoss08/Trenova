package bankreceiptworkitem

type Status string

const (
	StatusOpen      = Status("Open")
	StatusAssigned  = Status("Assigned")
	StatusInReview  = Status("InReview")
	StatusResolved  = Status("Resolved")
	StatusDismissed = Status("Dismissed")
)

func (s Status) String() string { return string(s) }

func (s Status) IsActive() bool {
	switch s {
	case StatusOpen, StatusAssigned, StatusInReview:
		return true
	case StatusResolved, StatusDismissed:
		return false
	default:
		return false
	}
}

type ResolutionType string

const (
	ResolutionMatchedToPayment         = ResolutionType("MatchedToPayment")
	ResolutionMarkedFalsePositive      = ResolutionType("MarkedFalsePositive")
	ResolutionRequiresExternalFollowUp = ResolutionType("RequiresExternalFollowUp")
	ResolutionSuperseded               = ResolutionType("Superseded")
)

func (r ResolutionType) String() string { return string(r) }
