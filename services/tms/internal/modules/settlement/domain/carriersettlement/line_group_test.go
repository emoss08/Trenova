package carriersettlement_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lineRef(raw string) *pulid.ID {
	value := pulid.ID(raw)
	return &value
}

func cost(
	eventType carriersettlement.CostEventType,
	description, shipment string,
	amount int64,
) *carriersettlement.CarrierSettlementLine {
	return &carriersettlement.CarrierSettlementLine{
		EventType:   eventType,
		Description: description,
		AmountMinor: amount,
		ShipmentID:  lineRef(shipment),
		CostEventID: lineRef("cce_" + shipment + description),
	}
}

func TestCostGroupsSumsEachCostAcrossTheLoadsItWasAccruedOn(t *testing.T) {
	t.Parallel()

	groups := carriersettlement.CostGroups([]*carriersettlement.CarrierSettlementLine{
		cost(carriersettlement.CostEventTypeLinehaulCost, "Linehaul (FlatRate)", "shp_1", 219000),
		cost(carriersettlement.CostEventTypeFuelSurcharge, "Fuel surcharge", "shp_1", 32850),
		cost(carriersettlement.CostEventTypeLinehaulCost, "Linehaul (FlatRate)", "shp_2", 230000),
		cost(carriersettlement.CostEventTypeAccessorial, "Detention", "shp_2", 11000),
		cost(carriersettlement.CostEventTypeAccessorial, "Detention", "shp_2", 11000),
		cost(carriersettlement.CostEventTypeAccessorial, "Lumper", "shp_1", 15000),
	})

	assert.Equal(t, []carriersettlement.CostGroup{
		{
			EventType:     carriersettlement.CostEventTypeLinehaulCost,
			Description:   "Linehaul (FlatRate)",
			AmountMinor:   449000,
			LineCount:     2,
			ShipmentCount: 2,
		},
		{
			EventType:     carriersettlement.CostEventTypeFuelSurcharge,
			Description:   "Fuel surcharge",
			AmountMinor:   32850,
			LineCount:     1,
			ShipmentCount: 1,
		},
		{
			EventType:     carriersettlement.CostEventTypeAccessorial,
			Description:   "Detention",
			AmountMinor:   22000,
			LineCount:     2,
			ShipmentCount: 1,
		},
		{
			EventType:     carriersettlement.CostEventTypeAccessorial,
			Description:   "Lumper",
			AmountMinor:   15000,
			LineCount:     1,
			ShipmentCount: 1,
		},
	}, groups)
}

func TestCostGroupsTellsAnAdjustmentAddedByHandFromAnAcceptedVariance(t *testing.T) {
	t.Parallel()

	groups := carriersettlement.CostGroups([]*carriersettlement.CarrierSettlementLine{
		{
			EventType:   carriersettlement.CostEventTypeAdjustment,
			Description: "Claim deduction",
			AmountMinor: -35000,
		},
		cost(carriersettlement.CostEventTypeAdjustment, "Invoice variance", "shp_2", 15000),
		cost(carriersettlement.CostEventTypeLinehaulCost, "Linehaul (FlatRate)", "shp_2", 230000),
		nil,
	})

	require.Len(t, groups, 3)
	assert.Equal(t, carriersettlement.CostEventTypeLinehaulCost, groups[0].EventType)
	assert.True(t, groups[1].Manual)
	assert.Equal(t, int64(-35000), groups[1].AmountMinor)
	assert.Zero(t, groups[1].ShipmentCount)
	assert.False(t, groups[2].Manual)
	assert.Equal(t, "Invoice variance", groups[2].Description)
}

func TestCarrierLoadLinesCarriesEachLineToItsLoadAndMove(t *testing.T) {
	t.Parallel()

	lines := carriersettlement.LoadLines([]*carriersettlement.CarrierSettlementLine{
		{
			ShipmentID:  lineRef("shp_1"),
			MoveID:      lineRef("smv_1"),
			ProNumber:   "S-1",
			AmountMinor: 219000,
		},
		nil,
	})

	require.Len(t, lines, 1)
	assert.Equal(t, lineRef("smv_1"), lines[0].MoveID)
	assert.Equal(t, "S-1", lines[0].ProNumber)
}

func moveLine(move, shipment, pro string, amount int64) *carriersettlement.CarrierSettlementLine {
	return &carriersettlement.CarrierSettlementLine{
		MoveID:      lineRef(move),
		ShipmentID:  lineRef(shipment),
		ProNumber:   pro,
		AmountMinor: amount,
	}
}

func TestMoveInvoicesPairsEachMoveWithTheInvoiceThatStandsForIt(t *testing.T) {
	t.Parallel()

	original := &carriersettlement.InvoiceMatch{
		ID:            "cim_1",
		InvoiceNumber: "88412",
		Status:        carriersettlement.InvoiceMatchStatusVariance,
		CreatedAt:     100,
	}
	copied := &carriersettlement.InvoiceMatch{
		ID:                 "cim_2",
		InvoiceNumber:      "88412",
		Status:             carriersettlement.InvoiceMatchStatusSuggested,
		DuplicateOfMatchID: lineRef("cim_1"),
		CreatedAt:          300,
	}
	rejected := &carriersettlement.InvoiceMatch{
		ID:        "cim_3",
		Status:    carriersettlement.InvoiceMatchStatusRejected,
		CreatedAt: 400,
	}
	resent := &carriersettlement.InvoiceMatch{
		ID:        "cim_5",
		Status:    carriersettlement.InvoiceMatchStatusMatched,
		CreatedAt: 200,
	}
	earlier := &carriersettlement.InvoiceMatch{
		ID:        "cim_4",
		Status:    carriersettlement.InvoiceMatchStatusMatched,
		CreatedAt: 150,
	}

	invoices := carriersettlement.MoveInvoices(
		[]*carriersettlement.CarrierSettlementLine{
			moveLine("smv_1", "shp_1", "S-240177", 230000),
			moveLine("smv_2", "shp_2", "S-240190", 218000),
			moveLine("smv_1", "shp_1", "", 34500),
			moveLine("smv_3", "shp_3", "S-240191", 100000),
			{AmountMinor: -35000},
			nil,
		},
		map[pulid.ID][]*carriersettlement.InvoiceMatch{
			"smv_1": {copied, rejected, original},
			"smv_3": {earlier, resent},
		},
	)

	require.Len(t, invoices, 3)
	assert.Equal(t, carriersettlement.MoveInvoice{
		MoveID:     "smv_1",
		ShipmentID: lineRef("shp_1"),
		ProNumber:  "S-240177",
		CostMinor:  264500,
		Match:      original,
	}, invoices[0])
	assert.Nil(t, invoices[1].Match)
	assert.Equal(t, int64(218000), invoices[1].CostMinor)
	assert.Same(t, resent, invoices[2].Match)
}

func TestMoveIDsListsEachMoveOnceInTheOrderItFirstAppears(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []pulid.ID{"smv_2", "smv_1"}, carriersettlement.MoveIDs(
		[]*carriersettlement.CarrierSettlementLine{
			moveLine("smv_2", "shp_2", "", 1),
			moveLine("smv_1", "shp_1", "", 1),
			moveLine("smv_2", "shp_2", "", 1),
			{AmountMinor: 1},
			nil,
		},
	))
}
