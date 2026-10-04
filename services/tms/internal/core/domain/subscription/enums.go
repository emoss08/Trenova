package subscription

type Status string

const (
	StatusTrialing = Status("trialing")
	StatusActive   = Status("active")
	StatusReadOnly = Status("read_only")
	StatusExpired  = Status("expired")
)

func AllStatuses() []Status {
	return []Status{StatusTrialing, StatusActive, StatusReadOnly, StatusExpired}
}

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusTrialing, StatusActive, StatusReadOnly, StatusExpired:
		return true
	default:
		return false
	}
}

func (s Status) AllowsWrites() bool {
	return s == StatusTrialing || s == StatusActive
}

func (s Status) AllowsLogin() bool {
	return s.IsValid() && s != StatusExpired
}
