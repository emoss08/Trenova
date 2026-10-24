package fiscalperiod

const (
	// MaxOperatingPeriodNumber is the highest ordinal an operating period may
	// take. Monthly calendars use 1–12 and 4-4-5 calendars fold their thirteenth
	// week-period into the twelfth ordinal.
	MaxOperatingPeriodNumber = 12

	// MaxAdjustingPeriodNumber is the highest ordinal an adjusting period may
	// take. Period 13 covers the year-end close; Period 14 is reserved for
	// post-audit entries.
	MaxAdjustingPeriodNumber = 14
)

type PeriodType string

const (
	// A calendar month, periods 1 to 12.
	PeriodTypeMonth PeriodType = "Month"

	// A calendar quarter.
	PeriodTypeQuarter PeriodType = "Quarter"

	// A run of weeks on a 4-4-5, 4-5-4 or 5-4-4 fiscal calendar.
	PeriodTypeWeek PeriodType = "Week"

	// The year-end adjustment period (period 13 or 14), sharing the final
	// period's dates so auditor entries stay out of operating results.
	PeriodTypeAdjusting PeriodType = "Adjusting"
)

func (p PeriodType) String() string {
	return string(p)
}

func (p PeriodType) IsValid() bool {
	switch p {
	case PeriodTypeMonth, PeriodTypeQuarter, PeriodTypeWeek, PeriodTypeAdjusting:
		return true
	default:
		return false
	}
}

type Status string

const (
	// Created but not yet open; nothing can be posted to it.
	StatusInactive = Status("Inactive")

	// Accepting every posting, from the subledgers and by hand.
	StatusOpen = Status("Open")

	// Soft-closed: subledger postings are blocked, manual journal entries are
	// still allowed.
	StatusLocked = Status("Locked")

	// Hard-closed: nothing can be posted; it can be reopened with
	// authorization and a reason, which is audited.
	StatusClosed = Status("Closed")

	// Closed for good once the audit is certified; it can never be reopened.
	StatusPermanentlyClosed = Status("PermanentlyClosed")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusInactive,
		StatusOpen,
		StatusLocked,
		StatusClosed,
		StatusPermanentlyClosed:
		return true
	default:
		return false
	}
}

func (s Status) CanActivate() bool {
	return s == StatusInactive
}

func (s Status) CanLock() bool {
	return s == StatusOpen
}

func (s Status) CanUnlock() bool {
	return s == StatusLocked
}

func (s Status) CanClose() bool {
	return s == StatusOpen || s == StatusLocked
}

func (s Status) CanReopen() bool {
	return s == StatusClosed
}

func (s Status) AcceptsPostings() bool {
	return s == StatusOpen || s == StatusLocked
}

func UnclosedStatuses() []Status {
	return []Status{StatusOpen, StatusLocked}
}
