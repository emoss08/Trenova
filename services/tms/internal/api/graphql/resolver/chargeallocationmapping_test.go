package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func allocationAuthCtx() *authctx.AuthContext {
	return &authctx.AuthContext{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
	}
}

func TestShipmentChargeAllocationsFromInput_NothingSentLeavesAllocationsAlone(t *testing.T) {
	t.Parallel()

	rows, err := shipmentChargeAllocationsFromInput(&gqlmodel.ShipmentInput{
		AdditionalCharges: []*gqlmodel.ShipmentAdditionalChargeInput{
			{AccessorialChargeID: pulid.MustNew("acc_").String()},
			nil,
		},
	}, allocationAuthCtx())
	require.NoError(t, err)
	assert.Nil(t, rows, "nil means the repository must not touch stored allocations")
}

func TestShipmentChargeAllocationsFromInput_FlattensFreightAndChargeRows(t *testing.T) {
	t.Parallel()

	authCtx := allocationAuthCtx()
	intel := pulid.MustNew("cus_")
	amd := pulid.MustNew("cus_")
	savedChargeID := pulid.MustNew("ac_")
	existingRowID := pulid.MustNew("chal_")
	sequence := 1
	version := 3

	rows, err := shipmentChargeAllocationsFromInput(&gqlmodel.ShipmentInput{
		FreightAllocations: []*gqlmodel.ChargeAllocationInput{
			{
				BillToCustomerID: intel.String(),
				Method:           shipmentdomain.ChargeAllocationMethodPercent,
				Percent:          new("60"),
			},
			{
				ID:               new(existingRowID.String()),
				BillToCustomerID: amd.String(),
				Method:           shipmentdomain.ChargeAllocationMethodPercent,
				Percent:          new("40"),
				Sequence:         &sequence,
				Version:          &version,
			},
		},
		AdditionalCharges: []*gqlmodel.ShipmentAdditionalChargeInput{
			{
				ID:                  new(savedChargeID.String()),
				AccessorialChargeID: pulid.MustNew("acc_").String(),
				Allocations: []*gqlmodel.ChargeAllocationInput{
					{
						BillToCustomerID: amd.String(),
						Method:           shipmentdomain.ChargeAllocationMethodAmount,
						Amount:           new("12.50"),
					},
				},
			},
			{
				AccessorialChargeID: pulid.MustNew("acc_").String(),
			},
			{
				AccessorialChargeID: pulid.MustNew("acc_").String(),
				Allocations: []*gqlmodel.ChargeAllocationInput{
					{
						BillToCustomerID: amd.String(),
						Method:           shipmentdomain.ChargeAllocationMethodPercent,
						Percent:          new("100"),
					},
				},
			},
		},
	}, authCtx)
	require.NoError(t, err)
	require.Len(t, rows, 4)

	freight := rows[0]
	assert.Equal(t, shipmentdomain.ChargeAllocationKindFreight, freight.ChargeKind)
	assert.Equal(t, intel, freight.BillToCustomerID)
	assert.Equal(t, authCtx.OrganizationID, freight.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, freight.BusinessUnitID)
	assert.True(t, freight.Percent.Decimal.Equal(decimal.NewFromInt(60)))
	assert.False(t, freight.Amount.Valid)
	assert.Nil(t, freight.AdditionalChargeIndex)
	assert.Nil(t, freight.AdditionalChargeID)

	second := rows[1]
	assert.Equal(t, existingRowID, second.ID)
	assert.Equal(t, int16(1), second.Sequence)
	assert.Equal(t, int64(3), second.Version)

	saved := rows[2]
	assert.Equal(t, shipmentdomain.ChargeAllocationKindAccessorial, saved.ChargeKind)
	require.NotNil(t, saved.AdditionalChargeID)
	assert.Equal(t, savedChargeID, *saved.AdditionalChargeID, "a saved charge is named by id")
	assert.Nil(t, saved.AdditionalChargeIndex)
	assert.True(t, saved.Amount.Decimal.Equal(decimal.RequireFromString("12.50")))
	assert.Equal(t, shipmentdomain.ChargeAllocationMethodAmount, saved.Method)

	unsaved := rows[3]
	assert.Equal(t, shipmentdomain.ChargeAllocationKindAccessorial, unsaved.ChargeKind)
	assert.Nil(t, unsaved.AdditionalChargeID)
	require.NotNil(t, unsaved.AdditionalChargeIndex)
	assert.Equal(t, 2, *unsaved.AdditionalChargeIndex, "an id-less charge is named by its position")
}

func TestShipmentChargeAllocationsFromInput_EmptyListsClearTheSplit(t *testing.T) {
	t.Parallel()

	rows, err := shipmentChargeAllocationsFromInput(&gqlmodel.ShipmentInput{
		FreightAllocations: []*gqlmodel.ChargeAllocationInput{},
	}, allocationAuthCtx())
	require.NoError(t, err)
	require.NotNil(t, rows)
	assert.Empty(t, rows, "an empty list bills everything to the shipment's payer")
}

func TestChargeAllocationsFromInput_RejectsBadValues(t *testing.T) {
	t.Parallel()

	authCtx := allocationAuthCtx()
	payer := pulid.MustNew("cus_").String()
	negative := -1

	_, err := chargeAllocationsFromInput([]*gqlmodel.ChargeAllocationInput{
		{
			BillToCustomerID: payer,
			Method:           shipmentdomain.ChargeAllocationMethodPercent,
			Percent:          new("sixty"),
		},
	}, shipmentdomain.ChargeAllocationKindFreight, "freightAllocations", authCtx, nil)
	require.Error(t, err)

	_, err = chargeAllocationsFromInput([]*gqlmodel.ChargeAllocationInput{
		{
			BillToCustomerID: payer,
			Method:           shipmentdomain.ChargeAllocationMethodAmount,
			Amount:           new("ten"),
		},
	}, shipmentdomain.ChargeAllocationKindFreight, "freightAllocations", authCtx, nil)
	require.Error(t, err)

	_, err = chargeAllocationsFromInput([]*gqlmodel.ChargeAllocationInput{
		{
			BillToCustomerID: payer,
			Method:           shipmentdomain.ChargeAllocationMethodPercent,
			Percent:          new("60"),
			Sequence:         &negative,
		},
	}, shipmentdomain.ChargeAllocationKindFreight, "freightAllocations", authCtx, nil)
	require.Error(t, err)

	_, err = chargeAllocationsFromInput([]*gqlmodel.ChargeAllocationInput{
		{
			BillToCustomerID: "",
			Method:           shipmentdomain.ChargeAllocationMethodPercent,
			Percent:          new("60"),
		},
	}, shipmentdomain.ChargeAllocationKindFreight, "freightAllocations", authCtx, nil)
	require.Error(t, err)

	rows, err := chargeAllocationsFromInput(
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
