package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const day = int64(secondsPerDay)

func TestPromiseStatusOf(t *testing.T) {
	t.Parallel()

	promisedOn := 100 * day
	cases := []struct {
		name string
		paid int64
		now  int64
		want PromiseStatus
	}{
		{"paid in full before the date", 50_000, promisedOn - day, PromiseKept},
		{"paid in full after the date", 60_000, promisedOn + 5*day, PromiseKept},
		{"part paid with the promised day still running", 10_000, promisedOn + day - 1, PromiseOpen},
		{"part paid once the promised day is over", 10_000, promisedOn + day, PromiseBroken},
		{"nothing paid before the date", 0, promisedOn - 3*day, PromiseOpen},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, PromiseStatusOf(50_000, tc.paid, promisedOn, tc.now), tc.name)
	}
}

func TestFollowUpIsDue(t *testing.T) {
	t.Parallel()

	now := 10*day + 3600
	today := 10 * day
	assert.True(t, FollowUpIsDue(&today, now), "scheduled for today")
	past := 8 * day
	assert.True(t, FollowUpIsDue(&past, now), "overdue")
	tomorrow := 11 * day
	assert.False(t, FollowUpIsDue(&tomorrow, now), "scheduled for tomorrow")
	assert.False(t, FollowUpIsDue(nil, now), "nothing scheduled")
}

func TestPriorityOf(t *testing.T) {
	t.Parallel()

	assert.Equal(t, PriorityCritical, PriorityOf(PriorityFacts{
		OldestDaysPastDue: 10, OverdueMinor: 1, HasPromise: true, Promise: PromiseBroken,
	}), "a broken promise comes first")
	assert.Equal(t, PriorityCritical, PriorityOf(PriorityFacts{OldestDaysPastDue: 75, OverdueMinor: 1}))
	assert.Equal(t, PriorityLow, PriorityOf(PriorityFacts{
		OldestDaysPastDue: 40, OverdueMinor: 1, HasPromise: true, Promise: PromiseOpen,
	}), "an open promise waits for its date")
	assert.Equal(t, PriorityHigh, PriorityOf(PriorityFacts{
		OldestDaysPastDue: 40, OverdueMinor: 1, HasPromise: true, Promise: PromiseOpen,
		FollowUpDue: true, FollowUpScheduled: true,
	}), "unless its follow-up has come round")
	assert.Equal(t, PriorityLow, PriorityOf(PriorityFacts{
		OldestDaysPastDue: 40, OverdueMinor: 1, FollowUpScheduled: true,
	}), "a follow-up not yet due waits")
	assert.Equal(t, PriorityHigh, PriorityOf(PriorityFacts{OldestDaysPastDue: 31, OverdueMinor: 1}))
	assert.Equal(t, PriorityNormal, PriorityOf(PriorityFacts{OldestDaysPastDue: 3, OverdueMinor: 1}))
	assert.Less(t, PriorityCritical.Rank(), PriorityHigh.Rank())
}
