package shipmentrating

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func reviewed(texts ...string) *shipment.RatingDetail {
	bill := &tarifftypes.Bill{Currency: "USD"}
	for _, text := range texts {
		bill.Reviews = append(bill.Reviews, tarifftypes.Note{Text: text})
	}
	return &shipment.RatingDetail{Bill: bill}
}

func TestRatingDetail_ReviewReasonsSkipBlankText(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"Holiday pickup"}, ReviewReasons(reviewed(" ", "Holiday pickup ")))
	assert.Empty(t, ReviewReasons(&shipment.RatingDetail{}))
	assert.Empty(t, ReviewReasons((*shipment.RatingDetail)(nil)))
	assert.False(t, DetailAwaitsReview(reviewed(" ")))
}

func TestRatingDetail_ApproveReviewSettlesTheHold(t *testing.T) {
	t.Parallel()

	detail := reviewed("Holiday pickup", "Liftgate on a dock")
	require.True(t, DetailAwaitsReview(detail))
	approver := pulid.MustNew("usr_")

	changed, err := ApproveReview(detail, approver, 1_800_000_000)
	require.NoError(t, err)

	assert.True(t, changed)
	assert.False(t, DetailAwaitsReview(detail))
	assert.Equal(t, &shipment.RatingReviewApproval{
		Reviews:      []string{"Holiday pickup", "Liftgate on a dock"},
		ApprovedByID: approver,
		ApprovedAt:   1_800_000_000,
	}, detail.ReviewApproval)

	again, err := ApproveReview(detail, pulid.MustNew("usr_"), 1_900_000_000)
	require.NoError(t, err)
	assert.False(t, again, "approving twice changes nothing")
	assert.Equal(t, approver, detail.ReviewApproval.ApprovedByID)
}

func TestRatingDetail_ApproveReviewRefusesARatingWithNothingToReview(t *testing.T) {
	t.Parallel()

	for _, detail := range []*shipment.RatingDetail{nil, {}, reviewed()} {
		changed, err := ApproveReview(detail, pulid.MustNew("usr_"), 1)
		assert.False(t, changed)
		var business *errortypes.BusinessError
		assert.True(t, errors.As(err, &business), "%v", err)
	}
}

func TestRatingDetail_CarryReviewApprovalOnlyForTheSameReasons(t *testing.T) {
	t.Parallel()

	previous := reviewed("A", "B")
	_, err := ApproveReview(previous, pulid.MustNew("usr_"), 1)
	require.NoError(t, err)

	same := reviewed("B", "A")
	CarryReviewApproval(same, previous)
	assert.False(t, DetailAwaitsReview(same))
	require.NotNil(t, same.ReviewApproval)
	assert.NotSame(t, previous.ReviewApproval, same.ReviewApproval)

	changed := reviewed("A", "C")
	CarryReviewApproval(changed, previous)
	assert.Nil(t, changed.ReviewApproval)
	assert.True(t, DetailAwaitsReview(changed))

	fewer := reviewed("A")
	CarryReviewApproval(fewer, previous)
	assert.True(t, DetailAwaitsReview(fewer))

	none := reviewed()
	CarryReviewApproval(none, previous)
	assert.Nil(t, none.ReviewApproval)

	unapproved := reviewed("A", "B")
	CarryReviewApproval(unapproved, reviewed("A", "B"))
	assert.True(t, DetailAwaitsReview(unapproved))
}

func TestRatingReviewHoldError_NamesTheHeldShipments(t *testing.T) {
	t.Parallel()

	held := &shipment.Shipment{
		ID:           pulid.MustNew("shp_"),
		ProNumber:    "PRO-1",
		RatingDetail: reviewed("A"),
	}
	other := &shipment.Shipment{
		ID:           pulid.MustNew("shp_"),
		ProNumber:    "PRO-2",
		RatingDetail: reviewed("B"),
	}
	clean := &shipment.Shipment{ID: pulid.MustNew("shp_"), RatingDetail: reviewed()}

	require.NoError(t, ReviewHoldError([]*shipment.Shipment{clean, nil}))

	var one *errortypes.BusinessError
	require.ErrorAs(t, ReviewHoldError([]*shipment.Shipment{clean, held}), &one)
	assert.Equal(t, held.ID.String(), one.Params["shipmentIds"])

	var many *errortypes.BusinessError
	require.ErrorAs(t, ReviewHoldError([]*shipment.Shipment{held, other}), &many)
	assert.Equal(t, held.ID.String()+","+other.ID.String(), many.Params["shipmentIds"])
}
