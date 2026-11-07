package dispatcheconomics

import (
	"errors"
	"maps"
)

type Preference string

const (
	PreferenceCustomerExperience = Preference("customerExperience")
	PreferenceOnTimeHistory      = Preference("onTimeHistory")
	PreferenceSafety             = Preference("safety")
	PreferenceAcceptance         = Preference("acceptance")
	PreferenceTrailerContinuity  = Preference("trailerContinuity")
	PreferenceHaulFit            = Preference("haulFit")
	PreferenceTimeOffBuffer      = Preference("timeOffBuffer")
)

func AllPreferences() []Preference {
	return []Preference{
		PreferenceCustomerExperience,
		PreferenceOnTimeHistory,
		PreferenceSafety,
		PreferenceAcceptance,
		PreferenceTrailerContinuity,
		PreferenceHaulFit,
		PreferenceTimeOffBuffer,
	}
}

func (p Preference) IsValid() bool {
	switch p {
	case PreferenceCustomerExperience,
		PreferenceOnTimeHistory,
		PreferenceSafety,
		PreferenceAcceptance,
		PreferenceTrailerContinuity,
		PreferenceHaulFit,
		PreferenceTimeOffBuffer:
		return true
	default:
		return false
	}
}

const (
	maxCentsPerUnit        = int64(1_000_000)
	maxPreferenceCents     = int64(1_000_000)
	maxCoverageCents       = int64(100_000_000)
	maxEarlyGraceMinutes   = int64(24 * 60)
	maxHomeTimeTargetDays  = int64(60)
	maxCoverageMargin      = int64(100)
	minCandidatesPerMove   = 5
	maxCandidatesPerMove   = 200
	defaultCandidatesCount = 40
)

var (
	ErrNegative            = errors.New("costs and prices cannot be negative")
	ErrOutOfRange          = errors.New("a cost or price is outside its allowed range")
	ErrUnknownPreference   = errors.New("unknown preference")
	ErrCandidatesPerMove   = errors.New("candidates per move must be between 5 and 200")
	ErrCoverageMarginRange = errors.New("coverage margin must be between 0 and 100 percent")
)

type Economics struct {
	EmptyMileCents        int64                `json:"emptyMileCents"`
	LateMinuteCents       int64                `json:"lateMinuteCents"`
	DriverHourCents       int64                `json:"driverHourCents"`
	EarlyGraceMinutes     int64                `json:"earlyGraceMinutes"`
	HomeTimeTargetDays    int64                `json:"homeTimeTargetDays"`
	HomeTimeDayCents      int64                `json:"homeTimeDayCents"`
	FairnessMileCents     int64                `json:"fairnessMileCents"`
	Preferences           map[Preference]int64 `json:"preferences"`
	CoverageMarginPercent int64                `json:"coverageMarginPercent"`
	CoverageFloorCents    int64                `json:"coverageFloorCents"`
	LatePremiumCents      int64                `json:"latePremiumCents"`
	DueNowPremiumCents    int64                `json:"dueNowPremiumCents"`
	CandidatesPerMove     int                  `json:"candidatesPerMove"`
	ReplanChangeCents     int64                `json:"replanChangeCents"`
	AnticipationMaxCents  int64                `json:"anticipationMaxCents"`
}

func Default() Economics {
	return Economics{
		EmptyMileCents:        200,
		LateMinuteCents:       100,
		DriverHourCents:       3_000,
		EarlyGraceMinutes:     120,
		HomeTimeTargetDays:    10,
		HomeTimeDayCents:      7_500,
		FairnessMileCents:     3,
		Preferences:           defaultPreferences(),
		CoverageMarginPercent: 30,
		CoverageFloorCents:    40_000,
		LatePremiumCents:      150_000,
		DueNowPremiumCents:    75_000,
		CandidatesPerMove:     defaultCandidatesCount,
		ReplanChangeCents:     2_500,
	}
}

func defaultPreferences() map[Preference]int64 {
	return map[Preference]int64{
		PreferenceCustomerExperience: 5_000,
		PreferenceOnTimeHistory:      7_500,
		PreferenceSafety:             7_500,
		PreferenceAcceptance:         3_000,
		PreferenceTrailerContinuity:  10_000,
		PreferenceHaulFit:            6_000,
		PreferenceTimeOffBuffer:      10_000,
	}
}

func (e *Economics) IsZero() bool {
	return e.EmptyMileCents == 0 && e.LateMinuteCents == 0 && e.DriverHourCents == 0 &&
		e.EarlyGraceMinutes == 0 && e.HomeTimeTargetDays == 0 && e.HomeTimeDayCents == 0 &&
		e.FairnessMileCents == 0 && len(e.Preferences) == 0 && e.CoverageMarginPercent == 0 &&
		e.CoverageFloorCents == 0 && e.LatePremiumCents == 0 && e.DueNowPremiumCents == 0 &&
		e.CandidatesPerMove == 0 && e.ReplanChangeCents == 0 && e.AnticipationMaxCents == 0
}

func (e *Economics) Normalized() Economics {
	if e.IsZero() {
		return Default()
	}
	out := *e
	out.Preferences = defaultPreferences()
	for key, value := range e.Preferences {
		if key.IsValid() {
			out.Preferences[key] = value
		}
	}
	if out.CandidatesPerMove == 0 {
		out.CandidatesPerMove = defaultCandidatesCount
	}

	return out
}

func (e *Economics) Preference(key Preference) int64 {
	if value, ok := e.Preferences[key]; ok {
		return value
	}

	return defaultPreferences()[key]
}

func (e *Economics) Validate() error {
	units := []int64{
		e.EmptyMileCents, e.LateMinuteCents, e.DriverHourCents, e.HomeTimeDayCents,
		e.FairnessMileCents, e.ReplanChangeCents,
	}
	for _, value := range units {
		if value < 0 {
			return ErrNegative
		}
		if value > maxCentsPerUnit {
			return ErrOutOfRange
		}
	}
	coverage := []int64{
		e.CoverageFloorCents, e.LatePremiumCents, e.DueNowPremiumCents, e.AnticipationMaxCents,
	}
	for _, value := range coverage {
		if value < 0 {
			return ErrNegative
		}
		if value > maxCoverageCents {
			return ErrOutOfRange
		}
	}
	if e.EarlyGraceMinutes < 0 || e.EarlyGraceMinutes > maxEarlyGraceMinutes ||
		e.HomeTimeTargetDays < 0 || e.HomeTimeTargetDays > maxHomeTimeTargetDays {
		return ErrOutOfRange
	}
	if e.CoverageMarginPercent < 0 || e.CoverageMarginPercent > maxCoverageMargin {
		return ErrCoverageMarginRange
	}
	if e.CandidatesPerMove != 0 &&
		(e.CandidatesPerMove < minCandidatesPerMove || e.CandidatesPerMove > maxCandidatesPerMove) {
		return ErrCandidatesPerMove
	}
	for key, value := range e.Preferences {
		if !key.IsValid() {
			return ErrUnknownPreference
		}
		if value < 0 {
			return ErrNegative
		}
		if value > maxPreferenceCents {
			return ErrOutOfRange
		}
	}

	return nil
}

func (e *Economics) Clone() Economics {
	out := *e
	out.Preferences = maps.Clone(e.Preferences)

	return out
}
