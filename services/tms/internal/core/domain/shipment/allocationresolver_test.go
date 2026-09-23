package shipment

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func allocDec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func allocationShipment(freight string, charges ...*AdditionalCharge) *Shipment {
	return &Shipment{
		ID:                  pulid.MustNew("shp_"),
		OrganizationID:      pulid.MustNew("org_"),
		BusinessUnitID:      pulid.MustNew("bu_"),
		CustomerID:          pulid.MustNew("cus_"),
		ProNumber:           "PRO-1",
		FreightChargeAmount: decimal.NewNullDecimal(allocDec(freight)),
		AdditionalCharges:   charges,
	}
}

func flatCharge(amount string, unit int16) *AdditionalCharge {
	return &AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Method:              accessorialcharge.MethodFlat,
		Amount:              allocDec(amount),
		Unit:                unit,
	}
}

func percentCharge(percent string) *AdditionalCharge {
	return &AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Method:              accessorialcharge.MethodPercentage,
		Amount:              allocDec(percent),
		Unit:                1,
	}
}

func percentAllocation(
	kind ChargeAllocationKind,
	payer pulid.ID,
	percent string,
	sequence int16,
) *ChargeAllocation {
	return &ChargeAllocation{
		ID:               pulid.MustNew("chal_"),
		ChargeKind:       kind,
		BillToCustomerID: payer,
		Method:           ChargeAllocationMethodPercent,
		Percent:          decimal.NewNullDecimal(allocDec(percent)),
		Sequence:         sequence,
	}
}

func amountAllocation(
	kind ChargeAllocationKind,
	payer pulid.ID,
	amount string,
	sequence int16,
) *ChargeAllocation {
	return &ChargeAllocation{
		ID:               pulid.MustNew("chal_"),
		ChargeKind:       kind,
		BillToCustomerID: payer,
		Method:           ChargeAllocationMethodAmount,
		Amount:           decimal.NewNullDecimal(allocDec(amount)),
		Sequence:         sequence,
	}
}

func forCharge(allocation *ChargeAllocation, charge *AdditionalCharge) *ChargeAllocation {
	id := charge.ID
	allocation.AdditionalChargeID = &id
	return allocation
}

func TestResolveShares_NoAllocationsBillsThePayerInFull(t *testing.T) {
	t.Parallel()

	t.Run("customer pays when no bill-to is set", func(t *testing.T) {
		t.Parallel()

		shp := allocationShipment("1000", flatCharge("50", 2))

		resolution, err := ResolveShares(shp, nil)
		require.NoError(t, err)

		require.Len(t, resolution.Shares, 1)
		share := resolution.Shares[0]
		assert.Equal(t, shp.CustomerID, share.PayerID)
		assert.Equal(t, shp.CustomerID, resolution.DefaultPayerID)
		assert.False(t, resolution.IsSplit)
		assert.True(t, share.FreightAmount.Equal(allocDec("1000")))
		assert.True(t, share.AccessorialAmount.Equal(allocDec("100")))
		assert.True(t, share.TotalAmount.Equal(allocDec("1100")))
		require.Len(t, share.Charges, 2)
		assert.True(t, share.Charges[0].Percent.Valid)
		assert.True(t, share.Charges[0].Percent.Decimal.Equal(decimal.NewFromInt(100)))
		assert.False(t, share.Charges[0].Partial)
		assert.True(t, share.Charges[0].AllocationID.IsNil())
	})

	t.Run("explicit bill-to pays when set", func(t *testing.T) {
		t.Parallel()

		shp := allocationShipment("1000")
		billTo := pulid.MustNew("cus_")
		shp.BillToCustomerID = &billTo

		resolution, err := ResolveShares(shp, nil)
		require.NoError(t, err)

		require.Len(t, resolution.Shares, 1)
		assert.Equal(t, billTo, resolution.Shares[0].PayerID)
		assert.Equal(t, billTo, resolution.DefaultPayerID)
		assert.False(t, resolution.IsSplit)
	})
}

func TestResolveShares_IntelPaysFreightAndAmdPaysAccessorials(t *testing.T) {
	t.Parallel()

	detention := flatCharge("75", 2)
	lumper := flatCharge("120", 1)
	shp := allocationShipment("2450", detention, lumper)
	intel := shp.CustomerID
	amd := pulid.MustNew("cus_")

	resolution, err := ResolveShares(shp, []*ChargeAllocation{
		forCharge(percentAllocation(ChargeAllocationKindAccessorial, amd, "100", 0), detention),
		forCharge(percentAllocation(ChargeAllocationKindAccessorial, amd, "100", 0), lumper),
	})
	require.NoError(t, err)

	require.Len(t, resolution.Shares, 2)
	assert.True(t, resolution.IsSplit)

	intelShare := resolution.ShareFor(intel)
	require.NotNil(t, intelShare)
	assert.Same(t, resolution.Shares[0], intelShare, "the default payer leads")
	assert.True(t, intelShare.FreightAmount.Equal(allocDec("2450")))
	assert.True(t, intelShare.AccessorialAmount.IsZero())
	assert.True(t, intelShare.TotalAmount.Equal(allocDec("2450")))
	require.Len(t, intelShare.Charges, 1)
	assert.Equal(t, ChargeAllocationKindFreight, intelShare.Charges[0].Kind)

	amdShare := resolution.ShareFor(amd)
	require.NotNil(t, amdShare)
	assert.True(t, amdShare.FreightAmount.IsZero())
	assert.True(t, amdShare.AccessorialAmount.Equal(allocDec("270")))
	assert.True(t, amdShare.TotalAmount.Equal(allocDec("270")))
	require.Len(t, amdShare.Charges, 2)
	for _, charge := range amdShare.Charges {
		assert.Equal(t, ChargeAllocationKindAccessorial, charge.Kind)
		assert.False(t, charge.Partial, "a whole charge redirected to another payer is not partial")
		assert.True(t, charge.AllocationID.IsNotNil())
	}

	total := intelShare.TotalAmount.Add(amdShare.TotalAmount)
	assert.True(t, total.Equal(allocDec("2720")), "the shares add back up to the shipment")
}

func TestResolveShares_PercentFreightSplitPutsTheRemainderOnTheLastRow(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("1000.01")
	first := shp.CustomerID
	second := pulid.MustNew("cus_")

	resolution, err := ResolveShares(shp, []*ChargeAllocation{
		percentAllocation(ChargeAllocationKindFreight, second, "40", 1),
		percentAllocation(ChargeAllocationKindFreight, first, "60", 0),
	})
	require.NoError(t, err)
	require.Len(t, resolution.Shares, 2)
	assert.True(t, resolution.IsSplit)

	// 60% of 1000.01 is 600.006, which rounds to 600.01; the second row by
	// sequence takes whatever is left so nothing is lost to rounding.
	assert.Equal(t, "600.01", resolution.ShareFor(first).FreightAmount.StringFixed(2))
	assert.Equal(t, "400.00", resolution.ShareFor(second).FreightAmount.StringFixed(2))
	for _, share := range resolution.Shares {
		require.Len(t, share.Charges, 1)
		assert.True(t, share.Charges[0].Partial)
		assert.True(t, share.Charges[0].Percent.Valid)
	}
	assert.Equal(t, "60", resolution.ShareFor(first).Charges[0].Percent.Decimal.String())
}

func TestResolveShares_ThreeWaySplitStillSumsToTheCharge(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("100")
	a := shp.CustomerID
	b := pulid.MustNew("cus_")
	c := pulid.MustNew("cus_")

	resolution, err := ResolveShares(shp, []*ChargeAllocation{
		percentAllocation(ChargeAllocationKindFreight, a, "33.333333", 0),
		percentAllocation(ChargeAllocationKindFreight, b, "33.333333", 1),
		percentAllocation(ChargeAllocationKindFreight, c, "33.333334", 2),
	})
	require.NoError(t, err)

	sum := decimal.Zero
	for _, share := range resolution.Shares {
		sum = sum.Add(share.FreightAmount)
	}
	assert.True(t, sum.Equal(allocDec("100")))
	assert.Equal(t, "33.34", resolution.ShareFor(c).FreightAmount.StringFixed(2))
}

func TestResolveShares_PercentageAccessorialIsComputedOnFullFreightThenSplit(t *testing.T) {
	t.Parallel()

	fuel := percentCharge("10")
	shp := allocationShipment("1000", fuel)
	first := shp.CustomerID
	second := pulid.MustNew("cus_")

	resolution, err := ResolveShares(shp, []*ChargeAllocation{
		percentAllocation(ChargeAllocationKindFreight, first, "60", 0),
		percentAllocation(ChargeAllocationKindFreight, second, "40", 1),
		forCharge(percentAllocation(ChargeAllocationKindAccessorial, first, "60", 0), fuel),
		forCharge(percentAllocation(ChargeAllocationKindAccessorial, second, "40", 1), fuel),
	})
	require.NoError(t, err)

	// The 10% surcharge is 100 on the whole 1000 linehaul; the payer with 40%
	// of the freight owes 40 of it, not 10% of their own 400.
	assert.Equal(t, "60.00", resolution.ShareFor(first).AccessorialAmount.StringFixed(2))
	assert.Equal(t, "40.00", resolution.ShareFor(second).AccessorialAmount.StringFixed(2))
	fuelShare := resolution.ShareFor(second).Charges[1]
	assert.Equal(t, ChargeAllocationKindAccessorial, fuelShare.Kind)
	assert.True(t, fuelShare.ChargeTotal.Equal(allocDec("100")))
}

func TestResolveShares_AmountRowsMustAddUpToTheCharge(t *testing.T) {
	t.Parallel()

	charge := flatCharge("100", 1)
	shp := allocationShipment("500", charge)
	other := pulid.MustNew("cus_")

	_, err := ResolveShares(shp, []*ChargeAllocation{
		forCharge(
			amountAllocation(ChargeAllocationKindAccessorial, shp.CustomerID, "60", 0),
			charge,
		),
		forCharge(amountAllocation(ChargeAllocationKindAccessorial, other, "39.99", 1), charge),
	})
	require.Error(t, err)

	var typed *errortypes.Error
	require.True(t, errors.As(err, &typed))
	assert.Equal(t, "additionalCharges[0].allocations", typed.Field)
	require.Len(t, typed.Args, 1)
	assert.Equal(t, "100.00", typed.Args[0])
}

func TestResolveShares_ExactAmountRowsAreAccepted(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("500")
	other := pulid.MustNew("cus_")

	resolution, err := ResolveShares(shp, []*ChargeAllocation{
		amountAllocation(ChargeAllocationKindFreight, shp.CustomerID, "300", 0),
		amountAllocation(ChargeAllocationKindFreight, other, "200", 1),
	})
	require.NoError(t, err)
	assert.Equal(t, "300.00", resolution.ShareFor(shp.CustomerID).FreightAmount.StringFixed(2))
	assert.Equal(t, "200.00", resolution.ShareFor(other).FreightAmount.StringFixed(2))
	assert.Equal(t, "40", resolution.ShareFor(other).Charges[0].Percent.Decimal.String())
}

func TestResolveShares_MixedMethodsOnOneChargeAreRefused(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("500")
	other := pulid.MustNew("cus_")

	_, err := ResolveShares(shp, []*ChargeAllocation{
		percentAllocation(ChargeAllocationKindFreight, shp.CustomerID, "60", 0),
		amountAllocation(ChargeAllocationKindFreight, other, "200", 1),
	})
	require.Error(t, err)

	var typed *errortypes.Error
	require.True(t, errors.As(err, &typed))
	assert.Equal(t, "freightAllocations", typed.Field)
}

func TestResolveShares_OrdersTheDefaultPayerFirstThenByID(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("300")
	payerZ := pulid.ID("cus_ZZZ")
	payerA := pulid.ID("cus_AAA")

	resolution, err := ResolveShares(shp, []*ChargeAllocation{
		percentAllocation(ChargeAllocationKindFreight, payerZ, "50", 0),
		percentAllocation(ChargeAllocationKindFreight, payerA, "50", 1),
	})
	require.NoError(t, err)

	require.Len(t, resolution.Shares, 3)
	assert.Equal(t, shp.CustomerID, resolution.Shares[0].PayerID)
	assert.True(
		t,
		resolution.Shares[0].TotalAmount.IsZero(),
		"the default payer keeps an empty share",
	)
	assert.Equal(t, payerA, resolution.Shares[1].PayerID)
	assert.Equal(t, payerZ, resolution.Shares[2].PayerID)
	assert.Equal(t, []pulid.ID{shp.CustomerID, payerA, payerZ}, resolution.PayerIDs())
	assert.Same(t, resolution.Shares[0], resolution.Primary())
}

func TestResolveShares_ShareForUnknownPayerIsNil(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("300")
	resolution, err := ResolveShares(shp, nil)
	require.NoError(t, err)

	assert.Nil(t, resolution.ShareFor(pulid.MustNew("cus_")))

	var empty *ShareResolution
	assert.Nil(t, empty.ShareFor(shp.CustomerID))
	assert.Nil(t, empty.PayerIDs())
	assert.Nil(t, empty.Primary())
}

func TestResolveShares_RedirectingAWholeChargeIsASplit(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("300")
	other := pulid.MustNew("cus_")

	resolution, err := ResolveShares(shp, []*ChargeAllocation{
		percentAllocation(ChargeAllocationKindFreight, other, "100", 0),
	})
	require.NoError(t, err)

	assert.True(t, resolution.IsSplit)
	assert.False(t, resolution.ShareFor(other).Charges[0].Partial)
	assert.True(t, resolution.ShareFor(shp.CustomerID).TotalAmount.IsZero())
}

func TestResolveShares_RequiresAShipment(t *testing.T) {
	t.Parallel()

	_, err := ResolveShares(nil, nil)
	require.Error(t, err)
}

func TestResolveShares_IgnoresOrderChargeRowsAndUnknownTargets(t *testing.T) {
	t.Parallel()

	charge := flatCharge("100", 1)
	shp := allocationShipment("500", charge)
	other := pulid.MustNew("cus_")
	orderChargeID := pulid.MustNew("ordchg_")
	orphan := percentAllocation(ChargeAllocationKindAccessorial, other, "100", 0)

	resolution, err := ResolveShares(shp, []*ChargeAllocation{
		{
			ID:               pulid.MustNew("chal_"),
			ChargeKind:       ChargeAllocationKindOrderCharge,
			OrderChargeID:    &orderChargeID,
			BillToCustomerID: other,
			Method:           ChargeAllocationMethodPercent,
			Percent:          decimal.NewNullDecimal(decimal.NewFromInt(100)),
		},
		orphan,
	})
	require.NoError(t, err)

	require.Len(t, resolution.Shares, 1)
	assert.Equal(t, "600.00", resolution.Shares[0].TotalAmount.StringFixed(2))
	assert.False(t, resolution.IsSplit)
}

func TestResolveOrderChargeShares(t *testing.T) {
	t.Parallel()

	defaultPayer := pulid.MustNew("cus_")
	other := pulid.MustNew("cus_")
	brokerage := OrderChargeRef{
		ID:          pulid.MustNew("ordchg_"),
		Description: "Customs brokerage",
		Amount:      allocDec("250"),
	}
	fuel := OrderChargeRef{
		ID:          pulid.MustNew("ordchg_"),
		Description: "Order fuel",
		Amount:      allocDec("80"),
	}
	fuelID := fuel.ID

	resolution, err := ResolveOrderChargeShares(
		[]OrderChargeRef{brokerage, fuel},
		[]*ChargeAllocation{
			{
				ID:               pulid.MustNew("chal_"),
				ChargeKind:       ChargeAllocationKindOrderCharge,
				OrderChargeID:    &fuelID,
				BillToCustomerID: other,
				Method:           ChargeAllocationMethodPercent,
				Percent:          decimal.NewNullDecimal(decimal.NewFromInt(100)),
			},
		},
		defaultPayer,
	)
	require.NoError(t, err)

	require.Len(t, resolution.Shares, 2)
	assert.True(t, resolution.IsSplit)

	defaultShare := resolution.ShareFor(defaultPayer)
	require.NotNil(t, defaultShare)
	require.Len(t, defaultShare.Charges, 1)
	assert.Equal(t, brokerage.ID, defaultShare.Charges[0].OrderChargeID)
	assert.Equal(t, "Customs brokerage", defaultShare.Charges[0].Description)
	assert.Equal(t, ChargeAllocationKindOrderCharge, defaultShare.Charges[0].Kind)
	assert.True(t, defaultShare.Charges[0].AllocationID.IsNil())
	assert.Equal(t, "250.00", defaultShare.AccessorialAmount.StringFixed(2))

	otherShare := resolution.ShareFor(other)
	require.NotNil(t, otherShare)
	require.Len(t, otherShare.Charges, 1)
	assert.Equal(t, fuel.ID, otherShare.Charges[0].OrderChargeID)
	assert.True(t, otherShare.Charges[0].AllocationID.IsNotNil())
	assert.Equal(t, "80.00", otherShare.TotalAmount.StringFixed(2))
}

func TestResolveOrderChargeShares_BadAllocationNamesTheCharge(t *testing.T) {
	t.Parallel()

	chargeID := pulid.MustNew("ordchg_")
	_, err := ResolveOrderChargeShares(
		[]OrderChargeRef{{ID: chargeID, Description: "Brokerage", Amount: allocDec("100")}},
		[]*ChargeAllocation{
			{
				ID:               pulid.MustNew("chal_"),
				ChargeKind:       ChargeAllocationKindOrderCharge,
				OrderChargeID:    &chargeID,
				BillToCustomerID: pulid.MustNew("cus_"),
				Method:           ChargeAllocationMethodPercent,
				Percent:          decimal.NewNullDecimal(decimal.NewFromInt(50)),
			},
		},
		pulid.MustNew("cus_"),
	)
	require.Error(t, err)

	var typed *errortypes.Error
	require.True(t, errors.As(err, &typed))
	assert.Equal(t, "charges[0].allocations", typed.Field)
}

func TestValidateAllocations(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	active := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         domaintypes.StatusActive,
	}
	inactive := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         domaintypes.StatusInactive,
	}
	foreign := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: buID,
		Status:         domaintypes.StatusActive,
	}
	missing := pulid.MustNew("cus_")
	customers := map[pulid.ID]*customer.Customer{
		active.ID:   active,
		inactive.ID: inactive,
		foreign.ID:  foreign,
	}

	locked := percentAllocation(ChargeAllocationKindFreight, active.ID, "100", 0)
	rows := []*ChargeAllocation{
		percentAllocation(ChargeAllocationKindFreight, active.ID, "25", 0),
		percentAllocation(ChargeAllocationKindFreight, inactive.ID, "25", 1),
		percentAllocation(ChargeAllocationKindFreight, foreign.ID, "25", 2),
		percentAllocation(ChargeAllocationKindFreight, missing, "25", 3),
		locked,
		{
			ChargeKind:       ChargeAllocationKindFreight,
			BillToCustomerID: active.ID,
			Method:           ChargeAllocationMethodPercent,
		},
	}

	multiErr := ValidateAllocations(&ValidateAllocationsParams{
		TenantOrgID: orgID,
		TenantBuID:  buID,
		Allocations: rows,
		Customers:   customers,
		Locked:      map[pulid.ID]struct{}{locked.ID: {}},
		Field:       "chargeAllocations",
	})
	require.True(t, multiErr.HasErrors())

	fields := make(map[string]string, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields[e.Field] = e.Message
	}
	assert.NotContains(t, fields, "chargeAllocations[0].billToCustomerId")
	assert.Contains(t, fields["chargeAllocations[1].billToCustomerId"], "not active")
	assert.Contains(t, fields["chargeAllocations[2].billToCustomerId"], "another organization")
	assert.Contains(t, fields["chargeAllocations[3].billToCustomerId"], "not found")
	assert.Contains(t, fields["chargeAllocations[4]"], "posted invoice")
	assert.Contains(t, fields["chargeAllocations[5].percent"], "required")

	assert.False(t, ValidateAllocations(nil).HasErrors())
	assert.False(t, ValidateAllocations(&ValidateAllocationsParams{
		TenantOrgID: orgID,
		TenantBuID:  buID,
		Allocations: []*ChargeAllocation{
			percentAllocation(ChargeAllocationKindFreight, active.ID, "100", 0),
		},
		Customers: customers,
	}).HasErrors())
}

func TestChargeAllocation_TargetIDAndValidation(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	chargeID := pulid.MustNew("ac_")
	orderChargeID := pulid.MustNew("ordchg_")

	assert.Equal(
		t,
		shipmentID,
		(&ChargeAllocation{ChargeKind: ChargeAllocationKindFreight, ShipmentID: &shipmentID}).TargetID(),
	)
	assert.Equal(
		t,
		chargeID,
		(&ChargeAllocation{ChargeKind: ChargeAllocationKindAccessorial, AdditionalChargeID: &chargeID}).TargetID(),
	)
	assert.Equal(
		t,
		orderChargeID,
		(&ChargeAllocation{ChargeKind: ChargeAllocationKindOrderCharge, OrderChargeID: &orderChargeID}).TargetID(),
	)
	assert.True(t, (&ChargeAllocation{}).TargetID().IsNil())
	var none *ChargeAllocation
	assert.True(t, none.TargetID().IsNil())
	assert.False(t, none.IsInvoiced())

	multiErr := errortypes.NewMultiError()
	(&ChargeAllocation{
		ChargeKind:       ChargeAllocationKindAccessorial,
		BillToCustomerID: pulid.MustNew("cus_"),
		Method:           ChargeAllocationMethodAmount,
		Amount:           decimal.NewNullDecimal(decimal.NewFromInt(10)),
		Percent:          decimal.NewNullDecimal(decimal.NewFromInt(10)),
	}).Validate(multiErr)
	fields := make(map[string]struct{}, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields[e.Field] = struct{}{}
	}
	assert.Contains(t, fields, "percent", "an amount row may not also carry a percent")
	assert.Contains(t, fields, "additionalChargeId", "an accessorial row must name its charge")

	multiErr = errortypes.NewMultiError()
	idx := 0
	(&ChargeAllocation{
		ChargeKind:            ChargeAllocationKindAccessorial,
		BillToCustomerID:      pulid.MustNew("cus_"),
		Method:                ChargeAllocationMethodPercent,
		Percent:               decimal.NewNullDecimal(decimal.NewFromInt(100)),
		AdditionalChargeIndex: &idx,
	}).Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), "a positional charge reference satisfies the target rule")
}
