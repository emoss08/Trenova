package billingqueue

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

type splitFixture struct {
	shipment  *shipment.Shipment
	acme      pulid.ID
	peak      pulid.ID
	detention *shipment.AdditionalCharge
	names     map[pulid.ID]PayerRef
}

// newSplitFixture is the shipment from the field report: $2,850 freight split by
// amount between Peak ($1,500) and Acme ($1,350), and a $1,154.38 detention
// charge nobody split, so it stays whole with Acme, the shipment's customer.
func newSplitFixture() *splitFixture {
	acme := pulid.MustNew("cus_")
	peak := pulid.MustNew("cus_")
	detention := &shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Method:              accessorialcharge.MethodFlat,
		Amount:              dec("1154.38"),
		Unit:                1,
		AccessorialCharge:   &accessorialcharge.AccessorialCharge{Code: "DET", Description: "Detention Fee"},
	}
	shipmentID := pulid.MustNew("shp_")
	shp := &shipment.Shipment{
		ID:                  shipmentID,
		CustomerID:          acme,
		FreightChargeAmount: decimal.NewNullDecimal(dec("2850")),
		OtherChargeAmount:   decimal.NewNullDecimal(dec("1154.38")),
		TotalChargeAmount:   decimal.NewNullDecimal(dec("4004.38")),
		AdditionalCharges:   []*shipment.AdditionalCharge{detention},
		ChargeAllocations: []*shipment.ChargeAllocation{
			{
				ID:               pulid.MustNew("chal_"),
				ShipmentID:       &shipmentID,
				ChargeKind:       shipment.ChargeAllocationKindFreight,
				BillToCustomerID: peak,
				Method:           shipment.ChargeAllocationMethodAmount,
				Amount:           decimal.NewNullDecimal(dec("1500")),
				Sequence:         0,
			},
			{
				ID:               pulid.MustNew("chal_"),
				ShipmentID:       &shipmentID,
				ChargeKind:       shipment.ChargeAllocationKindFreight,
				BillToCustomerID: acme,
				Method:           shipment.ChargeAllocationMethodAmount,
				Amount:           decimal.NewNullDecimal(dec("1350")),
				Sequence:         1,
			},
		},
	}

	return &splitFixture{
		shipment:  shp,
		acme:      acme,
		peak:      peak,
		detention: detention,
		names: map[pulid.ID]PayerRef{
			acme: {ID: acme, Name: "Acme Manufacturing", Code: "ACME"},
			peak: {ID: peak, Name: "Peak Distributing", Code: "PEAK"},
		},
	}
}

func TestBuildPayerShare_ShowsOnlyWhatThePayerOwes(t *testing.T) {
	t.Parallel()

	f := newSplitFixture()

	share := BuildPayerShare(f.shipment, f.acme, f.names)

	require.Empty(t, share.ResolutionError)
	assert.True(t, share.IsSplit)
	assert.True(t, share.TotalAmount.Equal(dec("2504.38")), "total %s", share.TotalAmount)
	assert.True(t, share.FreightAmount.Equal(dec("1350")))
	assert.True(t, share.AccessorialAmount.Equal(dec("1154.38")))
	assert.True(t, share.ShipmentTotal.Equal(dec("4004.38")))

	require.Len(t, share.Lines, 2)
	freight := share.Lines[0]
	assert.Equal(t, shipment.ChargeAllocationKindFreight, freight.Kind)
	assert.True(t, freight.ChargeTotal.Equal(dec("2850")))
	assert.True(t, freight.Amount.Equal(dec("1350")))
	assert.True(t, freight.Partial)
	assert.Equal(t, shipment.ChargeAllocationMethodAmount, freight.Method)
	require.Len(t, freight.Payers, 2)

	detention := share.Lines[1]
	assert.Equal(t, shipment.ChargeAllocationKindAccessorial, detention.Kind)
	assert.Equal(t, f.detention.ID, detention.AdditionalChargeID)
	assert.Equal(t, "Detention Fee", detention.Description)
	assert.True(t, detention.Amount.Equal(dec("1154.38")))
	assert.False(t, detention.Partial)
	assert.Empty(t, detention.Method)

	assert.Empty(t, share.OtherPayerLines)
	require.Len(t, share.Payers, 2)
	assert.Equal(t, f.acme, share.Payers[0].ID, "the shipment's own payer is listed first")
}

func TestBuildPayerShare_ChargesOwnedByOthersMoveToTheirOwnGroup(t *testing.T) {
	t.Parallel()

	f := newSplitFixture()

	share := BuildPayerShare(f.shipment, f.peak, f.names)

	require.Empty(t, share.ResolutionError)
	assert.True(t, share.TotalAmount.Equal(dec("1500")), "total %s", share.TotalAmount)
	require.Len(t, share.Lines, 1)
	assert.Equal(t, shipment.ChargeAllocationKindFreight, share.Lines[0].Kind)
	assert.True(t, share.Lines[0].Amount.Equal(dec("1500")))

	require.Len(t, share.OtherPayerLines, 1)
	other := share.OtherPayerLines[0]
	assert.Equal(t, f.detention.ID, other.AdditionalChargeID)
	assert.True(t, other.Amount.IsZero())
	assert.True(t, other.ChargeTotal.Equal(dec("1154.38")))
	require.Len(t, other.Payers, 1)
	assert.Equal(t, "Acme Manufacturing", other.Payers[0].PayerName)
	assert.True(t, other.Payers[0].Amount.Equal(dec("1154.38")))
}

func TestBuildPayerShare_UnsplitShipmentIsOneFullShare(t *testing.T) {
	t.Parallel()

	f := newSplitFixture()
	f.shipment.ChargeAllocations = nil

	share := BuildPayerShare(f.shipment, f.acme, f.names)

	assert.False(t, share.IsSplit)
	assert.True(t, share.TotalAmount.Equal(dec("4004.38")))
	assert.Len(t, share.Payers, 1)
	assert.Empty(t, share.OtherPayerLines)
}

func TestBuildPayerShare_ReportsASplitThatNoLongerResolves(t *testing.T) {
	t.Parallel()

	f := newSplitFixture()
	f.shipment.FreightChargeAmount = decimal.NewNullDecimal(dec("3000"))

	share := BuildPayerShare(f.shipment, f.peak, f.names)

	assert.Contains(t, share.ResolutionError, "must add up to 3000.00")
	assert.Empty(t, share.Lines)
	assert.True(t, share.ShipmentTotal.Equal(dec("4004.38")))
}

func TestBuildPayerShare_FallsBackToTheCustomerRelationForNames(t *testing.T) {
	t.Parallel()

	f := newSplitFixture()
	f.shipment.Customer = &customer.Customer{ID: f.acme, Name: "Acme From Relation", Code: "ACM"}
	delete(f.names, f.acme)

	share := BuildPayerShare(f.shipment, f.peak, f.names)

	require.Len(t, share.OtherPayerLines, 1)
	assert.Equal(t, "Acme From Relation", share.OtherPayerLines[0].Payers[0].PayerName)
}
