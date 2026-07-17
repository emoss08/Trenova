package cronutils

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

var parser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
)

// Validate parses a standard five-field cron expression.
func Validate(expression string) error {
	_, err := parser.Parse(expression)
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", expression, err)
	}
	return nil
}

// NextRun computes the next fire time for a five-field cron expression in the
// given IANA timezone, as epoch seconds after afterUnix.
func NextRun(expression, timezone string, afterUnix int64) (int64, error) {
	schedule, err := parser.Parse(expression)
	if err != nil {
		return 0, fmt.Errorf("invalid cron expression %q: %w", expression, err)
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return 0, fmt.Errorf("invalid timezone %q: %w", timezone, err)
	}

	return schedule.Next(time.Unix(afterUnix, 0).In(loc)).Unix(), nil
}
