package accessorialcharge

type Method string

const (
	MethodFlat       = Method("Flat")       // Fixed amount (TONU, layover, tarp, hazmat)
	MethodPerUnit    = Method("PerUnit")    // Rate × units (detention/hr, storage/day, team/mile)
	MethodPercentage = Method("Percentage") // Percentage of linehaul (fuel surcharge)
)

// RateUnit defines what PerUnit method multiplies against
type RateUnit string

const (
	RateUnitMile = RateUnit("Mile") // Team drivers, per-mile fuel surcharge
	RateUnitHour = RateUnit("Hour") // Detention, driver assist
	RateUnitDay  = RateUnit("Day")  // Layover, storage
	RateUnitStop = RateUnit("Stop") // Stop-off charges
)
