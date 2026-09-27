package timeutils_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
)

func TestJitteredStaysWithinTenPercent(t *testing.T) {
	t.Parallel()

	for range 1000 {
		got := timeutils.Jittered(10 * time.Minute)
		assert.GreaterOrEqual(t, got, 9*time.Minute)
		assert.Less(t, got, 11*time.Minute)
	}
	assert.Equal(t, time.Duration(5), timeutils.Jittered(5))
}
