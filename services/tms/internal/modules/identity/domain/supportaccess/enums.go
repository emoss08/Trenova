package supportaccess

import "time"

type Scope string

const (
	ScopeReadOnly = Scope("ReadOnly")
	ScopeFull     = Scope("Full")
)

func (s Scope) IsValid() bool {
	return s == ScopeReadOnly || s == ScopeFull
}

func (s Scope) String() string {
	return string(s)
}

func (s Scope) ControlPlane() string {
	if s == ScopeFull {
		return "full"
	}
	return "read_only"
}

type Status string

const (
	StatusLive    = Status("Live")
	StatusExpired = Status("Expired")
	StatusRevoked = Status("Revoked")
)

func (s Status) String() string {
	return string(s)
}

// Durations are the only lengths a grant may have. There is no open-ended grant and nothing
// longer than three days: an administrator who still needs Trenova after that grants again,
// and the audit log shows each grant's reason.
var Durations = []int{1, 4, 24, 72}

const MaxDuration = 72 * time.Hour

func ValidDuration(hours int) bool {
	for _, d := range Durations {
		if d == hours {
			return true
		}
	}
	return false
}
