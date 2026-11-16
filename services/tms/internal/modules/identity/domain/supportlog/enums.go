package supportlog

import "time"

type Period string

const (
	PeriodHour = Period("hour")
	PeriodDay  = Period("day")
	PeriodWeek = Period("week")
)

func (p Period) IsValid() bool {
	return p == PeriodHour || p == PeriodDay || p == PeriodWeek
}

func (p Period) Duration() time.Duration {
	switch p {
	case PeriodHour:
		return time.Hour
	case PeriodDay:
		return 24 * time.Hour
	case PeriodWeek:
		return 7 * 24 * time.Hour
	default:
		return 0
	}
}

func (p Period) String() string {
	return string(p)
}

type Status string

const (
	StatusRequested  = Status("requested")
	StatusCollecting = Status("collecting")
	StatusSent       = Status("sent")
	StatusFailed     = Status("failed")
)

func (s Status) IsValid() bool {
	switch s {
	case StatusRequested, StatusCollecting, StatusSent, StatusFailed:
		return true
	default:
		return false
	}
}

func (s Status) InFlight() bool {
	return s == StatusRequested || s == StatusCollecting
}

func (s Status) Finished() bool {
	return s == StatusSent || s == StatusFailed
}

func (s Status) String() string {
	return string(s)
}

type PullKind string

const (
	PullLines  = PullKind("lines")
	PullBundle = PullKind("bundle")
)

func (k PullKind) IsValid() bool {
	return k == PullLines || k == PullBundle
}

func (k PullKind) String() string {
	return string(k)
}
