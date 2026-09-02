package cronutils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const jan1of2026UTC = int64(1767225600)

func TestValidateAcceptsFiveFieldExpressions(t *testing.T) {
	t.Parallel()

	for _, expression := range []string{
		"* * * * *",
		"0 12 * * *",
		"*/15 0-6 1,15 * 1-5",
		"30 4 * * SUN",
	} {
		require.NoError(t, Validate(expression), "expression %q", expression)
	}
}

func TestValidateRejectsInvalidExpressions(t *testing.T) {
	t.Parallel()

	for _, expression := range []string{
		"",
		"not a cron",
		"0 0 12 * * *",
		"@hourly",
		"61 * * * *",
		"* 25 * * *",
	} {
		require.Error(t, Validate(expression), "expression %q", expression)
	}
}

func TestNextRunComputesNextFireTimeInUTC(t *testing.T) {
	t.Parallel()

	next, err := NextRun("0 12 * * *", "UTC", jan1of2026UTC)
	require.NoError(t, err)
	require.Equal(t, jan1of2026UTC+12*3600, next)
}

func TestNextRunHonorsTimezone(t *testing.T) {
	t.Parallel()

	next, err := NextRun("0 12 * * *", "America/New_York", jan1of2026UTC)
	require.NoError(t, err)
	require.Equal(t, jan1of2026UTC+17*3600, next)
}

func TestNextRunAdvancesPastTheGivenInstant(t *testing.T) {
	t.Parallel()

	atNoon := jan1of2026UTC + 12*3600
	next, err := NextRun("0 12 * * *", "UTC", atNoon)
	require.NoError(t, err)
	require.Equal(t, atNoon+24*3600, next)
}

func TestNextRunRejectsInvalidExpression(t *testing.T) {
	t.Parallel()

	_, err := NextRun("bogus", "UTC", jan1of2026UTC)
	require.Error(t, err)
}

func TestNextRunRejectsInvalidTimezone(t *testing.T) {
	t.Parallel()

	_, err := NextRun("0 12 * * *", "Not/AZone", jan1of2026UTC)
	require.Error(t, err)
}
