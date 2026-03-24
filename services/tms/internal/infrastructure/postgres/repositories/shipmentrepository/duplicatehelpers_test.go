package shipmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDuplicatedShipmentGraph_ResetsOperationalState(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	source := duplicateSourceFixture()

	graph := buildDuplicatedShipmentGraph(source, []string{"PRO-1", "PRO-2"}, false, userID)

	require.Len(t, graph.shipments, 2)
	require.Len(t, graph.moves, 2)
	require.Len(t, graph.stops, 4)
	require.Len(t, graph.additionalCharges, 2)
	require.Len(t, graph.commodities, 2)

	firstCopy := graph.shipments[0]
	assert.Equal(t, shipment.StatusNew, firstCopy.Status)
	assert.Equal(t, "PRO-1", firstCopy.ProNumber)
	assert.Equal(t, source.FormulaTemplateID, firstCopy.FormulaTemplateID)
	assert.Equal(t, source.ServiceTypeID, firstCopy.ServiceTypeID)
	assert.Equal(t, source.CustomerID, firstCopy.CustomerID)
	assert.Equal(t, userID, firstCopy.EnteredByID)
	assert.True(t, firstCopy.OwnerID.IsNil())
	assert.Nil(t, firstCopy.ActualShipDate)
	assert.Nil(t, firstCopy.ActualDeliveryDate)
	assert.Nil(t, firstCopy.CanceledAt)
	assert.True(t, firstCopy.CanceledByID.IsNil())
	assert.NotEqual(t, source.ID, firstCopy.ID)
	assert.NotEqual(t, source.BOL, firstCopy.BOL)
	assert.Len(t, firstCopy.AdditionalCharges, 1)
	assert.Len(t, firstCopy.Commodities, 1)
	assert.NotEqual(t, source.AdditionalCharges[0].ID, firstCopy.AdditionalCharges[0].ID)
	assert.NotEqual(t, source.Commodities[0].ID, firstCopy.Commodities[0].ID)

	firstMove := firstCopy.Moves[0]
	assert.Equal(t, shipment.MoveStatusNew, firstMove.Status)
	assert.NotEqual(t, source.Moves[0].ID, firstMove.ID)

	firstStop := firstMove.Stops[0]
	assert.Equal(t, shipment.StopStatusNew, firstStop.Status)
	assert.Nil(t, firstStop.ActualArrival)
	assert.Nil(t, firstStop.ActualDeparture)
	assert.NotEqual(t, source.Moves[0].Stops[0].ID, firstStop.ID)
	assert.Equal(t, source.Moves[0].Stops[0].ScheduledWindowStart, firstStop.ScheduledWindowStart)
	assert.Equal(t, source.Moves[0].Stops[0].ScheduledWindowEnd, firstStop.ScheduledWindowEnd)
}

func TestDuplicateStops_OverrideDatesPreservesRelativeOffsets(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	source := duplicateSourceFixture().Moves[0].Stops
	originalGap := source[1].ScheduledWindowStart - source[0].ScheduledWindowStart

	duplicated := duplicateStops(source, moveID, true)

	require.Len(t, duplicated, 2)
	assert.Equal(t, originalGap, duplicated[1].ScheduledWindowStart-duplicated[0].ScheduledWindowStart)
	require.NotNil(t, source[0].ScheduledWindowEnd)
	require.NotNil(t, duplicated[0].ScheduledWindowEnd)
	assert.Equal(
		t,
		*source[0].ScheduledWindowEnd-source[0].ScheduledWindowStart,
		*duplicated[0].ScheduledWindowEnd-duplicated[0].ScheduledWindowStart,
	)
}

func TestDeriveDuplicateBOL_TruncatesToLimit(t *testing.T) {
	t.Parallel()

	bol := deriveDuplicateBOL("ABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZ", 12)

	assert.LessOrEqual(t, len([]rune(bol)), maxShipmentBOLLength)
	assert.Contains(t, bol, "-COPY-12")
}

func duplicateSourceFixture() *shipment.Shipment {
	actualArrival := int64(150)
	actualDeparture := int64(160)
	actualShipDate := int64(170)
	actualDeliveryDate := int64(180)
	canceledAt := int64(190)
	pieces := int64(12)
	weight := int64(1200)
	tempMin := int16(34)
	tempMax := int16(40)
	distance := 450.5

	return &shipment.Shipment{
		ID:                 pulid.MustNew("shp_"),
		OrganizationID:     pulid.MustNew("org_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		ServiceTypeID:      pulid.MustNew("svc_"),
		ShipmentTypeID:     pulid.MustNew("sht_"),
		CustomerID:         pulid.MustNew("cus_"),
		TractorTypeID:      pulid.MustNew("eqt_"),
		TrailerTypeID:      pulid.MustNew("eqt_"),
		FormulaTemplateID:  pulid.MustNew("fmt_"),
		OwnerID:            pulid.MustNew("usr_"),
		CanceledByID:       pulid.MustNew("usr_"),
		Status:             shipment.StatusAssigned,
		BOL:                "ORIGINAL-BOL",
		ActualShipDate:     &actualShipDate,
		ActualDeliveryDate: &actualDeliveryDate,
		CanceledAt:         &canceledAt,
		Pieces:             &pieces,
		Weight:             &weight,
		TemperatureMin:     &tempMin,
		TemperatureMax:     &tempMax,
		AdditionalCharges: []*shipment.AdditionalCharge{
			{
				ID:                  pulid.MustNew("ac_"),
				BusinessUnitID:      pulid.MustNew("bu_"),
				OrganizationID:      pulid.MustNew("org_"),
				ShipmentID:          pulid.MustNew("shp_"),
				AccessorialChargeID: pulid.MustNew("acc_"),
				Method:              "Flat",
				Amount:              decimal.NewFromInt(25),
				Unit:                1,
			},
		},
		Commodities: []*shipment.ShipmentCommodity{
			{
				ID:             pulid.MustNew("sc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				ShipmentID:     pulid.MustNew("shp_"),
				CommodityID:    pulid.MustNew("com_"),
				Weight:         1200,
				Pieces:         12,
			},
		},
		Moves: []*shipment.ShipmentMove{
			{
				ID:             pulid.MustNew("sm_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				ShipmentID:     pulid.MustNew("shp_"),
				Status:         shipment.MoveStatusAssigned,
				Loaded:         true,
				Sequence:       0,
				Distance:       &distance,
				Stops: []*shipment.Stop{
					{
						ID:                   pulid.MustNew("stp_"),
						BusinessUnitID:       pulid.MustNew("bu_"),
						OrganizationID:       pulid.MustNew("org_"),
						ShipmentMoveID:       pulid.MustNew("sm_"),
						LocationID:           pulid.MustNew("loc_"),
						Status:               shipment.StopStatusCompleted,
						Type:                 shipment.StopTypePickup,
						ScheduleType:         shipment.StopScheduleTypeOpen,
						Sequence:             0,
						Pieces:               &pieces,
						Weight:               &weight,
						ScheduledWindowStart: 100,
						ScheduledWindowEnd:   int64Ptr(110),
						ActualArrival:        &actualArrival,
						ActualDeparture:      &actualDeparture,
					},
					{
						ID:                   pulid.MustNew("stp_"),
						BusinessUnitID:       pulid.MustNew("bu_"),
						OrganizationID:       pulid.MustNew("org_"),
						ShipmentMoveID:       pulid.MustNew("sm_"),
						LocationID:           pulid.MustNew("loc_"),
						Status:               shipment.StopStatusInTransit,
						Type:                 shipment.StopTypeDelivery,
						ScheduleType:         shipment.StopScheduleTypeOpen,
						Sequence:             1,
						ScheduledWindowStart: 200,
						ScheduledWindowEnd:   int64Ptr(210),
					},
				},
			},
		},
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
