package shipment

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func reviewed(texts ...string) *RatingDetail {
	bill := &tarifftypes.Bill{Currency: "USD"}
	for _, text := range texts {
		bill.Reviews = append(bill.Reviews, tarifftypes.Note{Text: text})
	}
	return &RatingDetail{Bill: bill}
}

func TestRatingDetail_ReviewReasonsSkipBlankText(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"Holiday pickup"}, reviewed(" ", "Holiday pickup ").ReviewReasons())
	assert.Empty(t, (&RatingDetail{}).ReviewReasons())
	assert.Empty(t, (*RatingDetail)(nil).ReviewReasons())
	assert.False(t, reviewed(" ").AwaitsReview())
}

func TestRatingDetail_ApproveReviewSettlesTheHold(t *testing.T) {
	t.Parallel()

	detail := reviewed("Holiday pickup", "Liftgate on a dock")
	require.True(t, detail.AwaitsReview())
	approver := pulid.MustNew("usr_")

	changed, err := detail.ApproveReview(approver, 1_800_000_000)
	require.NoError(t, err)

	assert.True(t, changed)
	assert.False(t, detail.AwaitsReview())
	assert.Equal(t, &RatingReviewApproval{
		Reviews:      []string{"Holiday pickup", "Liftgate on a dock"},
		ApprovedByID: approver,
		ApprovedAt:   1_800_000_000,
	}, detail.ReviewApproval)

	again, err := detail.ApproveReview(pulid.MustNew("usr_"), 1_900_000_000)
	require.NoError(t, err)
	assert.False(t, again, "approving twice changes nothing")
	assert.Equal(t, approver, detail.ReviewApproval.ApprovedByID)
}

func TestRatingDetail_ApproveReviewRefusesARatingWithNothingToReview(t *testing.T) {
	t.Parallel()

	for _, detail := range []*RatingDetail{nil, {}, reviewed()} {
		changed, err := detail.ApproveReview(pulid.MustNew("usr_"), 1)
		assert.False(t, changed)
		var business *errortypes.BusinessError
		assert.True(t, errors.As(err, &business), "%v", err)
	}
}

func TestRatingDetail_CarryReviewApprovalOnlyForTheSameReasons(t *testing.T) {
	t.Parallel()

	previous := reviewed("A", "B")
	_, err := previous.ApproveReview(pulid.MustNew("usr_"), 1)
	require.NoError(t, err)

	same := reviewed("B", "A")
	same.CarryReviewApproval(previous)
	assert.False(t, same.AwaitsReview())
	require.NotNil(t, same.ReviewApproval)
	assert.NotSame(t, previous.ReviewApproval, same.ReviewApproval)

	changed := reviewed("A", "C")
	changed.CarryReviewApproval(previous)
	assert.Nil(t, changed.ReviewApproval)
	assert.True(t, changed.AwaitsReview())

	fewer := reviewed("A")
	fewer.CarryReviewApproval(previous)
	assert.True(t, fewer.AwaitsReview())

	none := reviewed()
	none.CarryReviewApproval(previous)
	assert.Nil(t, none.ReviewApproval)

	unapproved := reviewed("A", "B")
	unapproved.CarryReviewApproval(reviewed("A", "B"))
	assert.True(t, unapproved.AwaitsReview())
}

func TestRatingReviewHoldError_NamesTheHeldShipments(t *testing.T) {
	t.Parallel()

	held := &Shipment{ID: pulid.MustNew("shp_"), ProNumber: "PRO-1", RatingDetail: reviewed("A")}
	other := &Shipment{ID: pulid.MustNew("shp_"), ProNumber: "PRO-2", RatingDetail: reviewed("B")}
	clean := &Shipment{ID: pulid.MustNew("shp_"), RatingDetail: reviewed()}

	require.NoError(t, RatingReviewHoldError([]*Shipment{clean, nil}))

	var one *errortypes.BusinessError
	require.ErrorAs(t, RatingReviewHoldError([]*Shipment{clean, held}), &one)
	assert.Equal(t, held.ID.String(), one.Params["shipmentIds"])

	var many *errortypes.BusinessError
	require.ErrorAs(t, RatingReviewHoldError([]*Shipment{held, other}), &many)
	assert.Equal(t, held.ID.String()+","+other.ID.String(), many.Params["shipmentIds"])
}
