package hosprojection

import (
	"regexp"
	"strconv"
)

var cycleHoursPattern = regexp.MustCompile(`(\d+)\s*hour`)

// LimitsForRuleset derives clock ceilings from the provider's free-text ruleset labels
// (e.g. "70 hour/8 day", "US Interstate Property", jurisdiction "CS"/"CN"), falling
// back to US interstate property limits when the labels don't say otherwise.
func LimitsForRuleset(cycle, shift, jurisdiction string) Limits {
	limits := Limits{
		DriveMs: standardDriveMs,
		ShiftMs: 14 * hourMs,
		CycleMs: 70 * hourMs,
		BreakMs: 8 * hourMs,
	}

	if match := cycleHoursPattern.FindStringSubmatch(cycle); len(match) == 2 {
		if hours, err := strconv.ParseInt(match[1], 10, 64); err == nil && hours > 0 {
			limits.CycleMs = hours * hourMs
		}
	}

	switch shift {
	case "US Interstate Passenger":
		limits.DriveMs = 10 * hourMs
		limits.ShiftMs = 15 * hourMs
	case "Texas Intrastate":
		limits.DriveMs = 12 * hourMs
		limits.ShiftMs = 15 * hourMs
	}

	if jurisdiction == canadaSouthJurisCS || jurisdiction == canadaNorthJurisCN {
		limits.DriveMs = 13 * hourMs
		limits.ShiftMs = 16 * hourMs
	}

	return limits
}
