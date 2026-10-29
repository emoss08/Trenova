package billingqueuereview

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func reviewedItem(reviews ...string) *billingqueue.BillingQueueItem {
	shipmentID := pulid.MustNew("shp_")
	bill := &tarifftypes.Bill{Currency: "USD"}
	for _, text := range reviews {
		bill.Reviews = append(bill.Reviews, tarifftypes.Note{Text: text})
	}
	return &billingqueue.BillingQueueItem{
		ID:         pulid.MustNew("bqi_"),
		ShipmentID: shipmentID,
		Status:     billingqueue.StatusInReview,
		BillType:   billingqueue.BillTypeInvoice,
		Shipment: &shipment.Shipment{
			ID: shipmentID,
			RatingDetail: &shipment.RatingDetail{
				Bill:                bill,
				RatedAt:             1_800_000_000,
				FormulaTemplateName: "Dock rates",
			},
		},
	}
}

func TestNewRatingReviewHold_DescribesAHeldRating(t *testing.T) {
	t.Parallel()

	item := reviewedItem("Holiday pickup", "Liftgate on a dock")

	hold := billingqueue.NewRatingReviewHold(item)

	require.NotNil(t, hold)
	assert.Equal(t, item.ShipmentID, hold.ShipmentID)
	assert.Equal(t, []string{"Holiday pickup", "Liftgate on a dock"}, hold.Reasons)
	assert.Equal(t, "Holiday pickup; Liftgate on a dock", hold.Summary())
	assert.Equal(t, int64(1_800_000_000), hold.RatedAt)
	assert.Equal(t, "Dock rates", hold.FormulaTemplateName)
}

func TestNewRatingReviewHold_HoldsNothingWhenNothingBills(t *testing.T) {
	t.Parallel()

	approved := reviewedItem("Holiday pickup")
	_, err := approved.Shipment.RatingDetail.ApproveReview(pulid.MustNew("usr_"), 1)
	require.NoError(t, err)

	credit := reviewedItem("Holiday pickup")
	credit.BillType = billingqueue.BillTypeCreditMemo

	posted := reviewedItem("Holiday pickup")
	posted.Status = billingqueue.StatusPosted

	unread := reviewedItem("Holiday pickup")
	unread.Shipment = nil

	for name, item := range map[string]*billingqueue.BillingQueueItem{
		"no reviews":   reviewedItem(),
		"approved":     approved,
		"credit memo":  credit,
		"posted":       posted,
		"not expanded": unread,
		"no item":      nil,
	} {
		assert.Nil(t, billingqueue.NewRatingReviewHold(item), name)
	}
}

func TestFindings_RaiseTheRatingReviewAsABlockingCheck(t *testing.T) {
	t.Parallel()

	hold := billingqueue.NewRatingReviewHold(reviewedItem("Holiday pickup"))

	issues := Findings(FindingInput{RatingReview: hold})

	require.Len(t, issues, 1)
	assert.Equal(t, billingqueue.CheckCharges, issues[0].CheckKey)
	assert.Equal(t, billingqueue.IssueRatingAwaitingReview, issues[0].Code)
	assert.Equal(t, hold.ShipmentID.String(), issues[0].SubjectKey)
	assert.Contains(t, issues[0].Reasoning, "Holiday pickup")
	assert.Empty(t, issues[0].Options, "only approving the rating settles it")
	assert.Empty(t, Findings(FindingInput{}))
}
