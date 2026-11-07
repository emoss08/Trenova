package auditretention

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMonthsAhead_CountsWholeMonthsPastTheCurrentOne(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 7, 14, 0, 0, 0, time.UTC)
	monthStart := func(year int, month time.Month) int64 {
		return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Unix()
	}

	assert.Equal(t, 0, monthsAhead(now, 0), "no partitions")
	assert.Equal(t, 0, monthsAhead(now, monthStart(2026, time.December)), "only this month")
	assert.Equal(t, 1, monthsAhead(now, monthStart(2027, time.January)), "through December")
	assert.Equal(t, 24, monthsAhead(now, monthStart(2028, time.December)), "two years past this month")
	assert.Equal(t, 0, monthsAhead(now, monthStart(2026, time.June)), "partitions that already ended")
}
