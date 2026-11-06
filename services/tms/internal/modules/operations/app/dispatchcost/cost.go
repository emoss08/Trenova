package dispatchcost

import (
	"math"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
)

const (
	neutralQuality       = 0.5
	minutesPerHour       = 60.0
	timeOffComfortHours  = 48.0
	customerFamiliarRuns = 10.0
	safetyZeroCredit     = 5.0
)

type Signal struct {
	Known   bool
	Quality float64
}

func Known(quality float64) Signal {
	return Signal{Known: true, Quality: clamp01(quality)}
}

func (s Signal) quality() float64 {
	if !s.Known {
		return neutralQuality
	}

	return clamp01(s.Quality)
}

type Features struct {
	EmptyMiles          float64
	PickupLateMinutes   float64
	DeliveryLateMinutes float64
	EarlyMinutes        float64
	RestHours           float64
	HomeOverdueDays     float64
	HomewardShare       float64
	WeekMilesOverMedian float64
	CustomerRuns        Signal
	OnTimeHistory       Signal
	Safety              Signal
	Acceptance          Signal
	TrailerContinuity   Signal
	HaulFit             Signal
	TimeOffBuffer       Signal
	EndMarketCents      int64
}

type Multipliers map[dispatcheconomics.Component]float64

func (m Multipliers) of(component dispatcheconomics.Component) float64 {
	value, ok := m[component]
	if !ok || !component.Soft() || math.IsNaN(value) || math.IsInf(value, 0) {
		return 1
	}

	return math.Min(
		math.Max(value, dispatcheconomics.MinMultiplier),
		dispatcheconomics.MaxMultiplier,
	)
}

type Model struct {
	economics   dispatcheconomics.Economics
	multipliers Multipliers
}

func NewModel(economics *dispatcheconomics.Economics, multipliers Multipliers) Model {
	return Model{economics: economics.Normalized(), multipliers: multipliers}
}

func (m *Model) Economics() dispatcheconomics.Economics {
	return m.economics
}

func (m *Model) Total(f *Features) int64 {
	var total int64
	m.each(f, func(_ dispatcheconomics.Component, cents int64) {
		total += cents
	})

	return total
}

func (m *Model) Split(f *Features) (hard, soft int64) {
	m.each(f, func(component dispatcheconomics.Component, cents int64) {
		if component.Soft() {
			soft += cents
			return
		}
		hard += cents
	})

	return hard, soft
}

func (m *Model) Explain(f *Features) dispatcheconomics.Cost {
	cost := dispatcheconomics.Cost{Lines: make([]dispatcheconomics.Line, 0, 8)}
	m.each(f, func(component dispatcheconomics.Component, cents int64) {
		cost.TotalCents += cents
		if cents != 0 {
			cost.Lines = append(
				cost.Lines,
				dispatcheconomics.Line{Component: component, Cents: cents},
			)
		}
	})

	return cost
}

func (m *Model) each(f *Features, emit func(dispatcheconomics.Component, int64)) {
	e := m.economics

	emit(dispatcheconomics.ComponentEmptyMiles, cents(f.EmptyMiles*float64(e.EmptyMileCents)))
	emit(
		dispatcheconomics.ComponentLateness,
		cents((f.PickupLateMinutes+f.DeliveryLateMinutes)*float64(e.LateMinuteCents)),
	)
	idle := math.Max(0, f.EarlyMinutes-float64(e.EarlyGraceMinutes)) / minutesPerHour
	emit(dispatcheconomics.ComponentIdleWait, cents(idle*float64(e.DriverHourCents)))
	emit(dispatcheconomics.ComponentRest, cents(f.RestHours*float64(e.DriverHourCents)))
	homePressure := f.HomeOverdueDays * float64(e.HomeTimeDayCents) * (1 - clamp01(f.HomewardShare))
	emit(
		dispatcheconomics.ComponentHomeTime,
		cents(homePressure*m.multipliers.of(dispatcheconomics.ComponentHomeTime)),
	)
	emit(
		dispatcheconomics.ComponentFairness,
		cents(
			math.Max(0, f.WeekMilesOverMedian)*float64(e.FairnessMileCents)*
				m.multipliers.of(dispatcheconomics.ComponentFairness),
		),
	)

	m.preference(
		emit,
		dispatcheconomics.ComponentCustomerExperience,
		dispatcheconomics.PreferenceCustomerExperience,
		f.CustomerRuns,
	)
	m.preference(
		emit,
		dispatcheconomics.ComponentOnTimeHistory,
		dispatcheconomics.PreferenceOnTimeHistory,
		f.OnTimeHistory,
	)
	m.preference(
		emit,
		dispatcheconomics.ComponentSafety,
		dispatcheconomics.PreferenceSafety,
		f.Safety,
	)
	m.preference(
		emit,
		dispatcheconomics.ComponentAcceptance,
		dispatcheconomics.PreferenceAcceptance,
		f.Acceptance,
	)
	m.preference(
		emit,
		dispatcheconomics.ComponentTrailerContinuity,
		dispatcheconomics.PreferenceTrailerContinuity,
		f.TrailerContinuity,
	)
	m.preference(
		emit,
		dispatcheconomics.ComponentHaulFit,
		dispatcheconomics.PreferenceHaulFit,
		f.HaulFit,
	)
	m.preference(
		emit,
		dispatcheconomics.ComponentTimeOffBuffer,
		dispatcheconomics.PreferenceTimeOffBuffer,
		f.TimeOffBuffer,
	)

	emit(dispatcheconomics.ComponentEndMarket, -f.EndMarketCents)
}

func (m *Model) preference(
	emit func(dispatcheconomics.Component, int64),
	component dispatcheconomics.Component,
	key dispatcheconomics.Preference,
	signal Signal,
) {
	full := float64(m.economics.Preference(key)) * m.multipliers.of(component)
	emit(component, cents(full*(1-signal.quality())))
}

func CustomerRunsQuality(runs int) Signal {
	if runs <= 0 {
		return Known(0)
	}

	return Known(float64(runs) / customerFamiliarRuns)
}

func TimeOffBufferQuality(gapHours float64) Signal {
	return Known(gapHours / timeOffComfortHours)
}

func SafetyQuality(violations float64) Signal {
	return Known(1 - violations/safetyZeroCredit)
}

func cents(value float64) int64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}

	return int64(math.Round(value))
}

func clamp01(value float64) float64 {
	if math.IsNaN(value) || value < 0 {
		return 0
	}

	return math.Min(value, 1)
}
