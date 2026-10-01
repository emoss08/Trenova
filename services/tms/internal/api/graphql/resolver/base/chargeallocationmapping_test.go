package base

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChargeAllocationsFromInput_RejectsBadValues(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.AllocationAuthCtx()
	payer := pulid.MustNew("cus_").String()
	negative := -1

	_, err := ChargeAllocationsFromInput([]*gqlmodel.ChargeAllocationInput{
		{
			BillToCustomerID: payer,
			Method:           shipmentdomain.ChargeAllocationMethodPercent,
			Percent:          new("sixty"),
		},
	}, shipmentdomain.ChargeAllocationKindFreight, "freightAllocations", authCtx, nil)
	require.Error(t, err)

	_, err = ChargeAllocationsFromInput([]*gqlmodel.ChargeAllocationInput{
		{
			BillToCustomerID: payer,
			Method:           shipmentdomain.ChargeAllocationMethodAmount,
			Amount:           new("ten"),
		},
	}, shipmentdomain.ChargeAllocationKindFreight, "freightAllocations", authCtx, nil)
	require.Error(t, err)

	_, err = ChargeAllocationsFromInput([]*gqlmodel.ChargeAllocationInput{
		{
			BillToCustomerID: payer,
			Method:           shipmentdomain.ChargeAllocationMethodPercent,
			Percent:          new("60"),
			Sequence:         &negative,
		},
	}, shipmentdomain.ChargeAllocationKindFreight, "freightAllocations", authCtx, nil)
	require.Error(t, err)

	_, err = ChargeAllocationsFromInput([]*gqlmodel.ChargeAllocationInput{
		{
			BillToCustomerID: "",
			Method:           shipmentdomain.ChargeAllocationMethodPercent,
			Percent:          new("60"),
		},
	}, shipmentdomain.ChargeAllocationKindFreight, "freightAllocations", authCtx, nil)
	require.Error(t, err)

	rows, err := ChargeAllocationsFromInput(
		[]*gqlmodel.ChargeAllocationInput{nil},
		shipmentdomain.ChargeAllocationKindFreight,
		"freightAllocations",
		authCtx,
		nil,
	)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestBillingSplitSummaryToModel(t *testing.T) {
	t.Parallel()

	t.Run("empty when the charges are not loaded", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, billingSplitSummaryToModel(nil))
		assert.Empty(t, billingSplitSummaryToModel(&shipmentdomain.Shipment{
			CustomerID:          pulid.MustNew("cus_"),
			FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(500)),
		}))
	})

	t.Run("labels payers from the loaded relations", func(t *testing.T) {
		t.Parallel()

		intel := &customer.Customer{ID: pulid.MustNew("cus_"), Name: "Intel", Code: "INTEL"}
		amd := &customer.Customer{ID: pulid.MustNew("cus_"), Name: "AMD", Code: "AMD"}
		charge := &shipmentdomain.AdditionalCharge{
			ID:     pulid.MustNew("ac_"),
			Method: "Flat",
			Amount: decimal.NewFromInt(120),
			Unit:   1,
		}
		chargeID := charge.ID
		entity := &shipmentdomain.Shipment{
			CustomerID:          intel.ID,
			Customer:            intel,
			FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(1000)),
			AdditionalCharges:   []*shipmentdomain.AdditionalCharge{charge},
			ChargeAllocations: []*shipmentdomain.ChargeAllocation{{
				ID:                 pulid.MustNew("chal_"),
				ChargeKind:         shipmentdomain.ChargeAllocationKindAccessorial,
				AdditionalChargeID: &chargeID,
				BillToCustomerID:   amd.ID,
				BillToCustomer:     amd,
				Method:             shipmentdomain.ChargeAllocationMethodPercent,
				Percent:            decimal.NewNullDecimal(decimal.NewFromInt(100)),
			}},
		}

		rows := billingSplitSummaryToModel(entity)
		require.Len(t, rows, 2)

		assert.Equal(t, intel.ID.String(), rows[0].PayerID)
		assert.Equal(t, "Intel", rows[0].PayerName)
		assert.Equal(t, "INTEL", rows[0].PayerCode)
		assert.True(t, rows[0].IsPrimary)
		assert.Equal(t, "1000.00", rows[0].FreightAmount)
		assert.Equal(t, "0.00", rows[0].AccessorialAmount)
		assert.Equal(t, "1000.00", rows[0].TotalAmount)
		assert.True(t, rows[0].IsSplit)

		assert.Equal(t, amd.ID.String(), rows[1].PayerID)
		assert.Equal(t, "AMD", rows[1].PayerName)
		assert.False(t, rows[1].IsPrimary)
		assert.Equal(t, "0.00", rows[1].FreightAmount)
		assert.Equal(t, "120.00", rows[1].AccessorialAmount)
		assert.Equal(t, "120.00", rows[1].TotalAmount)
	})

	t.Run("an unresolvable split yields nothing rather than a wrong figure", func(t *testing.T) {
		t.Parallel()

		entity := &shipmentdomain.Shipment{
			CustomerID:          pulid.MustNew("cus_"),
			FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(1000)),
			AdditionalCharges:   []*shipmentdomain.AdditionalCharge{},
			ChargeAllocations: []*shipmentdomain.ChargeAllocation{{
				ChargeKind:       shipmentdomain.ChargeAllocationKindFreight,
				BillToCustomerID: pulid.MustNew("cus_"),
				Method:           shipmentdomain.ChargeAllocationMethodPercent,
				Percent:          decimal.NewNullDecimal(decimal.NewFromInt(50)),
			}},
		}

		assert.Empty(t, billingSplitSummaryToModel(entity))
	})
}
