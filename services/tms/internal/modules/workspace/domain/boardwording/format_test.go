package boardwording_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardwording"
	"github.com/stretchr/testify/assert"
)

func TestFormatDurationNeverSkipsAUnitBetweenTheTwoItWrites(t *testing.T) {
	t.Parallel()

	cases := map[int64]string{
		0:                     "0m",
		45 * 60:               "45m",
		100 * 60:              "1h 40m",
		48*3600 + 30*60:       "48h 30m",
		48 * 3600:             "48h",
		71*3600 + 59*60:       "71h 59m",
		72*3600 + 30*60:       "3d",
		7*86_400 + 19*3600:    "7d 19h",
		-(8*86_400 + 4*3600):  "8d 4h",
		2*86_400 + 30*60 + 10: "48h 30m",
	}
	for seconds, want := range cases {
		assert.Equal(t, want, boardwording.FormatDuration(seconds), seconds)
	}
}

func TestFormatHoursNeverRollsWorkHoursIntoDays(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "48h 30m", boardwording.FormatHours(48*3600+30*60))
	assert.Equal(t, "44h", boardwording.FormatHours(44*3600))
	assert.Equal(t, "42h 15m", boardwording.FormatHours(42*3600+15*60))
	assert.Equal(t, "120h", boardwording.FormatHours(120*3600))
	assert.Equal(t, "30m", boardwording.FormatHours(30*60))
	assert.Equal(t, "5d", boardwording.FormatDuration(120*3600),
		"an elapsed time of the same length reads in days")
}

func TestFormatDurationWritesTheLargestTwoUnits(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "3h 40m", boardwording.FormatDuration(220*60))
	assert.Equal(t, "6h", boardwording.FormatDuration(6*3600))
	assert.Equal(t, "52h", boardwording.FormatDuration(52*3600))
	assert.Equal(t, "0m", boardwording.FormatDuration(0))
}
