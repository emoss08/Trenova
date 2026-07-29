package timeutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatDurationMs(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "3h 05m", FormatDurationMs(3*3_600_000+5*60_000))
	assert.Equal(t, "0h 00m", FormatDurationMs(-500))
}

func TestFormatLongDurationMs(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "45m", FormatLongDurationMs(45*60_000))
	assert.Equal(t, "3h 05m", FormatLongDurationMs(3*3_600_000+5*60_000))
	assert.Equal(t, "11d 12h", FormatLongDurationMs(11*86_400_000+12*3_600_000))
	assert.Equal(t, "0m", FormatLongDurationMs(-1))
}
