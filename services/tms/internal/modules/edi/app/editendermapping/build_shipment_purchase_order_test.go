package editendermapping

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestATenderedPurchaseOrderReachesTheShipmentsOrder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		tendered string
		want     string
	}{
		{name: "as tendered", tendered: " 8978300638 ", want: "8978300638"},
		{name: "none tendered", tendered: "   ", want: ""},
		{
			name:     "longer than an order holds",
			tendered: strings.Repeat("P", order.MaxPONumberLength+20),
			want:     strings.Repeat("P", order.MaxPONumberLength),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			transfer := &editransfer.EDITransfer{
				TargetOrganizationID: pulid.MustNew("org_"),
				TenderPayload:        editransfer.LoadTenderPayload{PONumber: tc.tendered},
			}
			target, err := (&Service{}).BuildTargetShipment(
				transfer,
				pulid.MustNew("bu_"),
				nil,
				pulid.MustNew("usr_"),
			)
			require.NoError(t, err)
			assert.Equal(t, tc.want, target.OrderPONumber)
		})
	}
}
