package billingqueue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnlyDetentionRatingAndCreditIssuesWaitOnSomeoneElse(t *testing.T) {
	t.Parallel()

	waiting := map[IssueCode]bool{}
	for _, code := range WaitingIssueCodes() {
		waiting[code] = true
		assert.True(t, code.WaitsOnOthers(), code)
	}
	for _, code := range []IssueCode{
		IssueAccessorialNotOnRateCon,
		IssueChargeOverRateCon,
		IssuePODMissing,
		IssuePODUnsigned,
		IssueDetentionELDMismatch,
		IssueChargesUnsplittable,
		IssuePossibleDuplicate,
		IssueUnbilledAccessorial,
		IssuePONumberMissing,
	} {
		assert.False(t, code.WaitsOnOthers(), code)
		assert.False(t, waiting[code], code)
	}
	assert.Equal(t, []IssueCode{
		IssueDetentionAwaitingApproval,
		IssueRatingAwaitingReview,
		IssueBillToCreditHold,
	}, WaitingIssueCodes())
}

func TestLanesNameTheQueueOrder(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []Lane{LaneChecking, LaneNeedsYou, LaneWaiting, LaneReady}, ReviewLanes())
	assert.Equal(
		t,
		[]Lane{LaneChecking, LaneNeedsYou, LaneWaiting, LaneReady, LaneHeld, LaneApproved},
		OpenLanes(),
	)
	assert.True(t, LaneClosed.IsValid())
	assert.False(t, Lane("Archived").IsValid())
}
