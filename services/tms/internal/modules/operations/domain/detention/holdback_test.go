package detention_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/stretchr/testify/assert"
)

func TestHoldBack_NamesTheFirstReasonAStoppedChargeShouldWait(t *testing.T) {
	t.Parallel()

	const now = int64(1_000_000)
	past, future := now-60, now+60
	cases := []struct {
		name  string
		facts detention.HoldBackFacts
		want  detention.HoldBack
	}{
		{"notice missed", detention.HoldBackFacts{
			Status:             detention.OccurrenceStatusPending,
			NotificationStatus: detention.NotificationStatusMissed,
			DepartureByHand:    true,
		}, detention.HoldBackNoticeMissed},
		{"notice window closed unsent", detention.HoldBackFacts{
			Status:             detention.OccurrenceStatusApproved,
			NotificationStatus: detention.NotificationStatusPending,
			NoticeDeadlineAt:   &past,
		}, detention.HoldBackNoticeMissed},
		{"notice window still open", detention.HoldBackFacts{
			Status:             detention.OccurrenceStatusApproved,
			NotificationStatus: detention.NotificationStatusPending,
			NoticeDeadlineAt:   &future,
		}, detention.HoldBackNone},
		{"notice failed", detention.HoldBackFacts{
			Status:             detention.OccurrenceStatusPending,
			NotificationStatus: detention.NotificationStatusFailed,
		}, detention.HoldBackNoticeFailed},
		{"notice late", detention.HoldBackFacts{
			Status:             detention.OccurrenceStatusPending,
			NotificationStatus: detention.NotificationStatusLate,
		}, detention.HoldBackNoticeLate},
		{"departure by hand before arrival by hand", detention.HoldBackFacts{
			Status:             detention.OccurrenceStatusApproved,
			NotificationStatus: detention.NotificationStatusSent,
			ArrivalByHand:      true,
			DepartureByHand:    true,
		}, detention.HoldBackDepartureByHand},
		{"arrival by hand", detention.HoldBackFacts{
			Status:             detention.OccurrenceStatusApproved,
			NotificationStatus: detention.NotificationStatusSent,
			ArrivalByHand:      true,
		}, detention.HoldBackArrivalByHand},
		{"awaiting approval", detention.HoldBackFacts{
			Status:             detention.OccurrenceStatusPending,
			NotificationStatus: detention.NotificationStatusSent,
			RequiresApproval:   true,
		}, detention.HoldBackAwaitingApproval},
		{"ready", detention.HoldBackFacts{
			Status:             detention.OccurrenceStatusApproved,
			NotificationStatus: detention.NotificationStatusSent,
		}, detention.HoldBackNone},
		{"billed is past holding", detention.HoldBackFacts{
			Status:             detention.OccurrenceStatusBilled,
			NotificationStatus: detention.NotificationStatusMissed,
		}, detention.HoldBackNone},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tc.facts.HoldBack(now), tc.name)
	}
}
