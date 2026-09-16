package allocationcheck

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// A split on a charge added in the same save points at the charge by position,
// because the charge has no id yet. The arithmetic check must still see it: a
// 60/30 split on a new charge is not 100% and has to be refused before insert.
func TestValidate_ChecksTheSplitOnAChargeAddedInTheSameSave(t *testing.T) {
	t.Parallel()

	idx := 0
	payerA, payerB := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	entity := &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		CustomerID:          payerA,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(1000)),
		AdditionalCharges: []*shipment.AdditionalCharge{{
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(100),
			Unit:                1,
		}},
		ChargeAllocations: []*shipment.ChargeAllocation{
			{
				ChargeKind:            shipment.ChargeAllocationKindAccessorial,
				AdditionalChargeIndex: &idx,
				BillToCustomerID:      payerA,
				Method:                shipment.ChargeAllocationMethodPercent,
				Percent:               decimal.NewNullDecimal(decimal.NewFromInt(60)),
			},
			{
				ChargeKind:            shipment.ChargeAllocationKindAccessorial,
				AdditionalChargeIndex: &idx,
				BillToCustomerID:      payerB,
				Method:                shipment.ChargeAllocationMethodPercent,
				Percent:               decimal.NewNullDecimal(decimal.NewFromInt(30)),
				Sequence:              1,
			},
		},
	}

	customers := mocks.NewMockCustomerRepository(t)
	customers.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return([]*customer.Customer{
		{ID: payerA, Status: domaintypes.StatusActive},
		{ID: payerB, Status: domaintypes.StatusActive},
	}, nil).Once()

	multiErr := Validate(t.Context(), Deps{CustomerRepo: customers}, entity, "")

	require.NotNil(t, multiErr)
	require.NotEmpty(t, multiErr.Errors)
	assert.Contains(t, multiErr.Errors[0].Message, "must total 100")
}

func TestValidate_NilAllocationsAreUntouchedAndPass(t *testing.T) {
	t.Parallel()

	entity := &shipment.Shipment{ID: pulid.MustNew("shp_"), CustomerID: pulid.MustNew("cus_")}

	assert.Nil(t, Validate(t.Context(), Deps{}, entity, ""))
}
