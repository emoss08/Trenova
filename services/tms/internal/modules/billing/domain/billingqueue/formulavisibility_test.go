package billingqueue

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBillingQueueItem_WithoutFormulaContent(t *testing.T) {
	t.Parallel()

	rated := &BillingQueueItem{
		ID: pulid.MustNew("bqi_"),
		Shipment: &shipment.Shipment{
			RatingDetail: &shipment.RatingDetail{
				Trace: rules.RawJSON(`{"statements":["miles * rate"]}`),
				Bill:  &tarifftypes.Bill{Total: decimal.NewFromInt(10)},
			},
		},
	}
	redacted := rated.WithoutFormulaContent()

	require.NotSame(t, rated, redacted)
	assert.Empty(t, redacted.Shipment.RatingDetail.Trace)
	assert.True(t, redacted.Shipment.RatingDetail.Bill.Total.Equal(decimal.NewFromInt(10)))
	assert.NotEmpty(t, rated.Shipment.RatingDetail.Trace)

	bare := &BillingQueueItem{ID: pulid.MustNew("bqi_")}
	assert.Same(t, bare, bare.WithoutFormulaContent())

	var missing *BillingQueueItem
	assert.Nil(t, missing.WithoutFormulaContent())

	items := []*BillingQueueItem{bare, rated}
	visible := WithoutFormulaContent(items)
	require.Len(t, visible, 2)
	assert.Same(t, bare, visible[0])
	assert.Empty(t, visible[1].Shipment.RatingDetail.Trace)
	assert.Same(t, rated, items[1])
}
