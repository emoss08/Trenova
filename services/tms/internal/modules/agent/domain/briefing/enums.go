package briefing

// RoleKey is who a briefing is written for. A dispatcher and a biller open
// the same morning on different numbers, so the day is gathered once and
// written several times rather than one page trying to serve everybody.
type RoleKey string

const (
	RoleDispatch   = RoleKey("Dispatch")
	RoleBilling    = RoleKey("Billing")
	RoleCompliance = RoleKey("Compliance")
	RoleLeadership = RoleKey("Leadership")
	RoleGeneral    = RoleKey("General")
)

func (r RoleKey) IsValid() bool {
	switch r {
	case RoleDispatch, RoleBilling, RoleCompliance, RoleLeadership, RoleGeneral:
		return true
	default:
		return false
	}
}

func (r RoleKey) String() string { return string(r) }

// Label is the role in the reader's words.
func (r RoleKey) Label() string {
	switch r {
	case RoleDispatch:
		return "Dispatch"
	case RoleBilling:
		return "Billing"
	case RoleCompliance:
		return "Compliance"
	case RoleLeadership:
		return "Leadership"
	case RoleGeneral:
		return "Everyone"
	default:
		return string(r)
	}
}

func AllRoleKeys() []RoleKey {
	return []RoleKey{RoleDispatch, RoleBilling, RoleCompliance, RoleLeadership, RoleGeneral}
}

// Status is how far the morning's writing got. A briefing exists from the
// moment the facts are gathered, so a reader who opens the page while the
// model is still writing sees the figures rather than an empty screen.
type Status string

const (
	// StatusPending means the facts are gathered and the prose is not written.
	StatusPending = Status("Pending")
	// StatusReady means the briefing is complete, narrated or not.
	StatusReady = Status("Ready")
	// StatusFailed means the facts could not be gathered at all.
	StatusFailed = Status("Failed")
)

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusReady, StatusFailed:
		return true
	default:
		return false
	}
}

func (s Status) String() string { return string(s) }

// SectionKey names a block of the page. The keys are fixed so a client can
// give each one its own shape and icon, and so a model cannot invent a
// section that nothing computed.
type SectionKey string

const (
	SectionAttention  = SectionKey("Attention")
	SectionToday      = SectionKey("Today")
	SectionCoverage   = SectionKey("Coverage")
	SectionExceptions = SectionKey("Exceptions")
	SectionDecisions  = SectionKey("Decisions")
	SectionCompliance = SectionKey("Compliance")
	SectionBilling    = SectionKey("Billing")
	SectionCash       = SectionKey("Cash")
)

func (k SectionKey) IsValid() bool {
	switch k {
	case SectionAttention, SectionToday, SectionCoverage, SectionExceptions,
		SectionDecisions, SectionCompliance, SectionBilling, SectionCash:
		return true
	default:
		return false
	}
}

func (k SectionKey) String() string { return string(k) }
