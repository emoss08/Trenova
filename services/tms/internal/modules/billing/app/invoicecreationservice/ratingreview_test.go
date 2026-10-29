package invoicecreationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentrating"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func heldRating() *shipment.Shipment {
	return &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "PRO-5",
		RatingDetail: &shipment.RatingDetail{Bill: &tarifftypes.Bill{
			Reviews: []tarifftypes.Note{{Text: "Holiday pickup"}},
		}},
	}
}

func TestGuardItemRatingReview_RefusesToInvoiceAHeldRating(t *testing.T) {
	t.Parallel()

	held := heldRating()
	err := guardItemRatingReview(
		&billingqueue.BillingQueueItem{BillType: billingqueue.BillTypeInvoice},
		held,
	)

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Equal(t, held.ID.String(), business.Params["shipmentIds"])
}

func TestGuardItemRatingReview_LetsACreditMemoAndAnApprovedRatingThrough(t *testing.T) {
	t.Parallel()

	approved := heldRating()
	_, err := shipmentrating.ApproveReview(approved.RatingDetail, pulid.MustNew("usr_"), 1)
	require.NoError(t, err)

	invoiceItem := &billingqueue.BillingQueueItem{BillType: billingqueue.BillTypeInvoice}
	creditItem := &billingqueue.BillingQueueItem{BillType: billingqueue.BillTypeCreditMemo}

	require.NoError(t, guardItemRatingReview(creditItem, heldRating()))
	require.NoError(t, guardItemRatingReview(invoiceItem, approved))
	require.NoError(t, guardItemRatingReview(invoiceItem, nil))
}
