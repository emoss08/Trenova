package timeutils_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
)

func TestYearsAgoUnix(t *testing.T) {
	t.Parallel()

	t.Run("returns timestamp in the past", func(t *testing.T) {
		t.Parallel()
		result := timeutils.YearsAgoUnix(1)
		now := time.Now().Unix()

		assert.Less(t, result, now)
	})

	t.Run("approximate one year ago", func(t *testing.T) {
		t.Parallel()
		result := timeutils.YearsAgoUnix(1)
		expected := time.Now().AddDate(-1, 0, 0).Unix()

		assert.InDelta(t, expected, result, 2)
	})
}

func TestMonthsAgoUnix(t *testing.T) {
	t.Parallel()

	t.Run("returns timestamp in the past", func(t *testing.T) {
		t.Parallel()
		result := timeutils.MonthsAgoUnix(6)
		now := time.Now().Unix()

		assert.Less(t, result, now)
	})

	t.Run("approximate six months ago", func(t *testing.T) {
		t.Parallel()
		result := timeutils.MonthsAgoUnix(6)
		expected := time.Now().AddDate(0, -6, 0).Unix()

		assert.InDelta(t, expected, result, 2)
	})
}

func TestDaysAgoUnix(t *testing.T) {
	t.Parallel()

	t.Run("returns timestamp in the past", func(t *testing.T) {
		t.Parallel()
		result := timeutils.DaysAgoUnix(30)
		now := time.Now().Unix()

		assert.Less(t, result, now)
	})

	t.Run("approximate 30 days ago", func(t *testing.T) {
		t.Parallel()
		result := timeutils.DaysAgoUnix(30)
		expected := time.Now().AddDate(0, 0, -30).Unix()

		assert.InDelta(t, expected, result, 2)
	})
}

func TestIsAtLeastAge(t *testing.T) {
	t.Parallel()

	t.Run("returns true for someone old enough", func(t *testing.T) {
		t.Parallel()
		dob := time.Now().AddDate(-25, 0, 0).Unix()
		assert.True(t, timeutils.IsAtLeastAge(dob, 18))
	})

	t.Run("returns false for someone too young", func(t *testing.T) {
		t.Parallel()
		dob := time.Now().AddDate(-16, 0, 0).Unix()
		assert.False(t, timeutils.IsAtLeastAge(dob, 18))
	})

	t.Run("returns true at exact age boundary", func(t *testing.T) {
		t.Parallel()
		dob := time.Now().AddDate(-18, 0, 0).Unix()
		assert.True(t, timeutils.IsAtLeastAge(dob, 18))
	})

	t.Run("returns false for zero dob", func(t *testing.T) {
		t.Parallel()
		assert.False(t, timeutils.IsAtLeastAge(0, 18))
	})

	t.Run("returns false for negative dob", func(t *testing.T) {
		t.Parallel()
		assert.False(t, timeutils.IsAtLeastAge(-100, 18))
	})
}

func TestIsExpired(t *testing.T) {
	t.Parallel()

	t.Run("returns true for past timestamp", func(t *testing.T) {
		t.Parallel()
		past := time.Now().Add(-time.Hour).Unix()
		assert.True(t, timeutils.IsExpired(past))
	})

	t.Run("returns false for future timestamp", func(t *testing.T) {
		t.Parallel()
		future := time.Now().Add(time.Hour).Unix()
		assert.False(t, timeutils.IsExpired(future))
	})

	t.Run("returns true for zero", func(t *testing.T) {
		t.Parallel()
		assert.True(t, timeutils.IsExpired(0))
	})

	t.Run("returns true for negative", func(t *testing.T) {
		t.Parallel()
		assert.True(t, timeutils.IsExpired(-1))
	})
}

func TestIsOverdue(t *testing.T) {
	t.Parallel()

	t.Run("returns true for past due date", func(t *testing.T) {
		t.Parallel()
		past := time.Now().Add(-time.Hour).Unix()
		assert.True(t, timeutils.IsOverdue(past))
	})

	t.Run("returns false for future due date", func(t *testing.T) {
		t.Parallel()
		future := time.Now().Add(time.Hour).Unix()
		assert.False(t, timeutils.IsOverdue(future))
	})

	t.Run("returns false for zero", func(t *testing.T) {
		t.Parallel()
		assert.False(t, timeutils.IsOverdue(0))
	})

	t.Run("returns false for negative", func(t *testing.T) {
		t.Parallel()
		assert.False(t, timeutils.IsOverdue(-1))
	})
}

func TestIsDueSoon(t *testing.T) {
	t.Parallel()

	t.Run("returns true when within warning period", func(t *testing.T) {
		t.Parallel()
		dueDate := time.Now().Add(2 * 24 * time.Hour).Unix()
		assert.True(t, timeutils.IsDueSoon(dueDate, 7))
	})

	t.Run("returns false when outside warning period", func(t *testing.T) {
		t.Parallel()
		dueDate := time.Now().Add(30 * 24 * time.Hour).Unix()
		assert.False(t, timeutils.IsDueSoon(dueDate, 7))
	})

	t.Run("returns true for past due date within threshold", func(t *testing.T) {
		t.Parallel()
		past := time.Now().Add(-time.Hour).Unix()
		assert.True(t, timeutils.IsDueSoon(past, 7))
	})

	t.Run("returns false for zero due date", func(t *testing.T) {
		t.Parallel()
		assert.False(t, timeutils.IsDueSoon(0, 7))
	})
}

func TestIsWithinMonths(t *testing.T) {
	t.Parallel()

	t.Run("returns true for recent timestamp", func(t *testing.T) {
		t.Parallel()
		recent := time.Now().Add(-time.Hour).Unix()
		assert.True(t, timeutils.IsWithinMonths(recent, 3))
	})

	t.Run("returns false for old timestamp", func(t *testing.T) {
		t.Parallel()
		old := time.Now().AddDate(-1, 0, 0).Unix()
		assert.False(t, timeutils.IsWithinMonths(old, 3))
	})

	t.Run("returns false for zero", func(t *testing.T) {
		t.Parallel()
		assert.False(t, timeutils.IsWithinMonths(0, 3))
	})
}

func TestMaxAllowedUnix(t *testing.T) {
	t.Parallel()

	t.Run("adds years to timestamp", func(t *testing.T) {
		t.Parallel()
		from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
		result := timeutils.MaxAllowedUnix(from, 5)
		expected := time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

		assert.Equal(t, expected, result)
	})

	t.Run("handles zero years", func(t *testing.T) {
		t.Parallel()
		from := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC).Unix()
		result := timeutils.MaxAllowedUnix(from, 0)

		assert.Equal(t, from, result)
	})
}
