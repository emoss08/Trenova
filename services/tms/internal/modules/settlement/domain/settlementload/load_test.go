package settlementload_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementload"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func id(raw string) *pulid.ID {
	value := pulid.ID(raw)
	return &value
}

func TestSharesTotalsEachLoadOnceInTheOrderItFirstAppears(t *testing.T) {
	t.Parallel()

	shares := settlementload.Shares([]settlementload.Line{
		{ShipmentID: id("shp_b"), MoveID: id("smv_b1"), ProNumber: "S-2", AmountMinor: 30000},
		{ShipmentID: id("shp_a"), MoveID: id("smv_a1"), ProNumber: "S-1", AmountMinor: 61220},
		{ShipmentID: id("shp_b"), MoveID: id("smv_b2"), AmountMinor: 4990},
		{ShipmentID: id("shp_b"), MoveID: id("smv_b1"), AmountMinor: -1000},
		{AmountMinor: -9600},
		{ShipmentID: id(""), AmountMinor: 500},
	})

	assert.Equal(t, []settlementload.Share{
		{
			ShipmentID:  "shp_b",
			MoveIDs:     []pulid.ID{"smv_b1", "smv_b2"},
			ProNumber:   "S-2",
			AmountMinor: 33990,
			LineCount:   3,
		},
		{
			ShipmentID:  "shp_a",
			MoveIDs:     []pulid.ID{"smv_a1"},
			ProNumber:   "S-1",
			AmountMinor: 61220,
			LineCount:   1,
		},
	}, shares)
}

func TestSharesTakesTheProNumberFromTheFirstLineThatCarriesOne(t *testing.T) {
	t.Parallel()

	shares := settlementload.Shares([]settlementload.Line{
		{ShipmentID: id("shp_a"), AmountMinor: 100},
		{ShipmentID: id("shp_a"), ProNumber: "S-1", AmountMinor: 100},
	})

	assert.Equal(t, "S-1", shares[0].ProNumber)
	assert.Empty(t, shares[0].MoveIDs)
}

func TestShipmentIDsListsEachLoadOnce(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []pulid.ID{"shp_a", "shp_b"}, settlementload.ShipmentIDs([]settlementload.Line{
		{ShipmentID: id("shp_a")},
		{ShipmentID: id("shp_b")},
		{ShipmentID: id("shp_a")},
		{},
	}))
	assert.Empty(t, settlementload.ShipmentIDs(nil))
}
