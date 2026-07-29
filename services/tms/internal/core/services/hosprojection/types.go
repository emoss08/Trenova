package hosprojection

import "github.com/emoss08/trenova/internal/core/domain/telematics"

const (
	hourMs      = int64(3_600_000)
	breakRestMs = 30 * 60_000
	resetRestMs = 10 * hourMs
	restartMs   = 34 * hourMs

	splitLongSleeperMs = 7 * hourMs
	splitShortRestMs   = 2 * hourMs
	splitPairTotalMs   = 10 * hourMs

	sixtyHourCycleMs   = 60 * hourMs
	sevenDayWindowSec  = int64(7 * 86400)
	eightDayWindowSec  = int64(8 * 86400)
	standardDriveMs    = 11 * hourMs
	maxEnRouteResets   = 2
	canadaSouthJurisCS = "CS"
	canadaNorthJurisCN = "CN"
)

// Strategy names the rest plan that makes a trip legal, ordered from least to most
// disruptive for the driver.
type Strategy string

const (
	StrategyCurrentClocks = Strategy("currentClocks")
	StrategySplitSleeper  = Strategy("splitSleeper")
	StrategyTenHourReset  = Strategy("tenHourReset")
	StrategyRestart34     = Strategy("restart34")
)

// Limiter identifies the clock that makes a trip infeasible.
type Limiter string

const (
	LimiterNone  = Limiter("")
	LimiterDrive = Limiter("drive")
	LimiterShift = Limiter("shift")
	LimiterCycle = Limiter("cycle")
	LimiterBreak = Limiter("break")
)

// Limits are the ruleset ceilings for each clock in milliseconds.
type Limits struct {
	DriveMs int64
	ShiftMs int64
	CycleMs int64
	BreakMs int64
}

// Clocks are the remaining milliseconds on each clock at a point in time.
type Clocks struct {
	DriveMs int64
	ShiftMs int64
	CycleMs int64
	BreakMs int64
}

// DutyInterval is one closed span of the driver's duty timeline in unix seconds.
type DutyInterval struct {
	Status  telematics.DutyStatus
	StartAt int64
	EndAt   int64
}

// Input carries everything Project needs; all times are unix seconds and the engine
// never reads the wall clock, so results are fully deterministic.
type Input struct {
	Now             int64
	Departure       int64
	TripDriveMs     int64
	Clocks          Clocks
	ClocksAt        int64
	CycleTomorrowMs int64
	DutyStatus      telematics.DutyStatus
	Limits          Limits
	Jurisdiction    string
	Timeline        []DutyInterval
}

// TripPlan is the simulated execution of the trip's drive time, including mandated
// 30-minute breaks and any en-route 10-hour resets.
type TripPlan struct {
	TotalMs  int64
	Breaks   int
	Resets   int
	Arrival  int64
	MarginMs int64
}

// Result is the best legal option for taking the trip at its departure time.
type Result struct {
	Feasible          bool
	Strategy          Strategy
	Limiter           Limiter
	DriveAvailableMs  int64
	ShiftAvailableMs  int64
	CycleAvailableMs  int64
	RestStartDeadline int64
	Trip              TripPlan
	Detail            string
}
