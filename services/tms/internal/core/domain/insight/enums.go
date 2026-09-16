package insight

import "time"

// DismissalSuppression is how long dismissing an insight keeps the same finding
// off the home screen.
//
// Forever would be wrong — an operation changes, and a condition someone waved
// away in spring can be the most expensive thing they have by autumn. A month is
// long enough that dismissing means something and short enough that a real and
// persistent problem comes back.
const DismissalSuppression = 30 * 24 * time.Hour

// Category groups insights by the kind of problem they describe. It is what a
// reader filters on, and it is fixed in code rather than chosen by a model: an
// insight's category follows from the detector that produced it.
type Category string

const (
	// CategoryServiceQuality covers service the customer feels — on-time
	// performance, service failures, appointment adherence.
	CategoryServiceQuality = Category("ServiceQuality")
	// CategoryCashFlow covers money earned but not yet collected.
	CategoryCashFlow = Category("CashFlow")
	// CategoryCostLeakage covers money the operation is losing quietly: dwell
	// time nobody billed for, empty miles nobody costed.
	CategoryCostLeakage = Category("CostLeakage")
	// CategoryCompliance covers exposure that takes capacity off the road —
	// credentials and qualifications running out.
	CategoryCompliance = Category("Compliance")
)

func (c Category) IsValid() bool {
	switch c {
	case CategoryServiceQuality, CategoryCashFlow, CategoryCostLeakage, CategoryCompliance:
		return true
	default:
		return false
	}
}

func (c Category) String() string {
	return string(c)
}

// Severity is how loudly an insight should present itself.
//
// A detector assigns this from its own thresholds, never a model. Letting
// generated prose decide urgency would make the loudest insight the one the
// model found most interesting rather than the one costing the most money.
type Severity string

const (
	SeverityInfo     = Severity("Info")
	SeverityWarning  = Severity("Warning")
	SeverityCritical = Severity("Critical")
)

func (s Severity) IsValid() bool {
	switch s {
	case SeverityInfo, SeverityWarning, SeverityCritical:
		return true
	default:
		return false
	}
}

func (s Severity) String() string {
	return string(s)
}

// Rank orders severities for sorting, highest first. Kept as an explicit method
// rather than relying on the string order, which would sort Critical after Info.
func (s Severity) Rank() int {
	switch s {
	case SeverityCritical:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

// Status is where an insight is in its life.
type Status string

const (
	// StatusActive means the condition was present at the last refresh.
	StatusActive = Status("Active")
	// StatusDismissed means a person judged it not worth acting on. The next
	// refresh will find the same condition and must not put the card straight
	// back, so a dismissal suppresses its finding for DismissalSuppression rather
	// than resolving it. It is "not now", not "never": a condition still present
	// months later has earned another look.
	StatusDismissed = Status("Dismissed")
	// StatusResolved means a later refresh no longer found the condition. It is
	// set by the system rather than a person.
	StatusResolved = Status("Resolved")
	// StatusSuperseded means a newer run of the same detector replaced this one
	// with fresher numbers.
	StatusSuperseded = Status("Superseded")
)

func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusDismissed, StatusResolved, StatusSuperseded:
		return true
	default:
		return false
	}
}

func (s Status) String() string {
	return string(s)
}

// Unit says how to render a metric's value. It exists because the same number
// means different things — 12 hours of dwell and 12 stops are not comparable —
// and the client must not guess from the key name.
type Unit string

const (
	UnitCount    = Unit("Count")
	UnitCurrency = Unit("Currency")
	UnitPercent  = Unit("Percent")
	UnitDays     = Unit("Days")
	UnitHours    = Unit("Hours")
	UnitMiles    = Unit("Miles")
)

func (u Unit) IsValid() bool {
	switch u {
	case UnitCount, UnitCurrency, UnitPercent, UnitDays, UnitHours, UnitMiles:
		return true
	default:
		return false
	}
}

func (u Unit) String() string {
	return string(u)
}

// Direction says which way is bad for a metric, so the client can colour a
// change without a table of metric keys. A detector knows this; a renderer
// cannot infer it, because a rising number is good for revenue and bad for
// detention.
type Direction string

const (
	// DirectionNeutral means the value carries no good-or-bad reading on its own.
	DirectionNeutral = Direction("Neutral")
	// DirectionHigherIsWorse marks a metric where an increase is the problem.
	DirectionHigherIsWorse = Direction("HigherIsWorse")
	// DirectionLowerIsWorse marks a metric where a decrease is the problem.
	DirectionLowerIsWorse = Direction("LowerIsWorse")
)

func (d Direction) IsValid() bool {
	switch d {
	case DirectionNeutral, DirectionHigherIsWorse, DirectionLowerIsWorse:
		return true
	default:
		return false
	}
}

func (d Direction) String() string {
	return string(d)
}
