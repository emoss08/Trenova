package ratesimulationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func amount(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func historicalShipment(billed string) *shipment.Shipment {
	customerID := pulid.MustNew("cus_")

	return &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		ProNumber:         "PRO-1001",
		CustomerID:        customerID,
		TotalChargeAmount: decimal.NewNullDecimal(amount(billed)),
	}
}

func simulated(value string) *services.RatedShipment {
	ruleID := pulid.MustNew("ragr_")

	return &services.RatedShipment{
		Amount:  amount(value),
		Outcome: ratequote.OutcomeRated,
		RuleID:  &ruleID,
		Quote:   &ratequote.RateQuote{Trace: &ratetypes.Trace{}},
	}
}

// The whole point of a simulation is the comparison, and the number it compares
// against has to be what the shipment was actually invoiced. Anything else and
// the delta measures the wrong thing.
func TestReplay_ComparesAgainstWhatTheShipmentWasBilled(t *testing.T) {
	t.Parallel()

	result := replayResult(historicalShipment("2000"), simulated("2200"), nil)

	assert.True(t, result.BeforeAmount.Equal(amount("2000")), "before %s", result.BeforeAmount)
	assert.True(t, result.AfterAmount.Equal(amount("2200")), "after %s", result.AfterAmount)
	assert.True(t, result.Delta.Equal(amount("200")), "delta %s", result.Delta)
	// 200 of 2000 is 10%.
	assert.True(t, result.DeltaPercent.Equal(amount("10")), "pct %s", result.DeltaPercent)
}

// A shipment the simulated contract does not cover is the most important row in
// the whole report: it is revenue the change would drop on the floor. Leaving
// it out would make the simulation look better than the change is.
func TestReplay_AShipmentTheContractDoesNotCoverIsRecordedNotSkipped(t *testing.T) {
	t.Parallel()

	uncovered := &services.RatedShipment{Outcome: ratequote.OutcomeNoRateFound}

	result := replayResult(historicalShipment("2000"), uncovered, nil)

	assert.True(t, result.Failed())
	assert.Equal(t, ratequote.OutcomeNoRateFound, result.Outcome)
	assert.True(t, result.BeforeAmount.Equal(amount("2000")))
}

func TestReplay_ARatingThatErroredCarriesTheReason(t *testing.T) {
	t.Parallel()

	result := replayResult(historicalShipment("2000"), nil, assert.AnError)

	require.True(t, result.Failed())
	assert.Equal(t, ratequote.OutcomeError, result.Outcome)
	assert.NotEmpty(t, result.Error)
}

// The result grid is grouped by customer and by lane, and it reads them off the
// row rather than joining back to a shipment that may since have been edited. A
// simulation is a record of a moment.
func TestReplay_CopiesTheGroupingKeysOntoTheRow(t *testing.T) {
	t.Parallel()

	entity := historicalShipment("2000")
	equipmentID := pulid.MustNew("trt_")
	entity.TractorTypeID = equipmentID

	rated := simulated("2200")
	rated.Quote.Trace = &ratetypes.Trace{
		Candidates: []ratetypes.Candidate{{RuleID: "ragr_x", LaneKey: "ST:il>ST:ga", Won: true}},
	}

	result := replayResult(entity, rated, nil)

	require.NotNil(t, result.CustomerID)
	assert.Equal(t, entity.CustomerID, *result.CustomerID)
	assert.Equal(t, "ST:il>ST:ga", result.LaneKey)
	require.NotNil(t, result.EquipmentTypeID)
	assert.Equal(t, equipmentID, *result.EquipmentTypeID)
}

// Which rule priced it is what turns "this went up" into "this went up because
// this lane now wins", which is the only form of the answer somebody can act on.
func TestReplay_NamesTheRuleThatPricedIt(t *testing.T) {
	t.Parallel()

	rated := simulated("2200")

	result := replayResult(historicalShipment("2000"), rated, nil)

	require.NotNil(t, result.AfterRuleID)
	assert.Equal(t, *rated.RuleID, *result.AfterRuleID)
}

// A shipment that was never billed anything has no base to measure a move
// against. Dividing anyway would panic on a row nobody could interpret.
func TestReplay_AShipmentBilledAtNothingReportsNoPercentageMove(t *testing.T) {
	t.Parallel()

	entity := historicalShipment("2000")
	entity.TotalChargeAmount = decimal.NullDecimal{}

	result := replayResult(entity, simulated("2200"), nil)

	assert.True(t, result.BeforeAmount.IsZero())
	assert.True(t, result.DeltaPercent.IsZero())
	assert.True(t, result.Delta.Equal(amount("2200")))
}

func TestReplay_CarriesTheProNumberSoARowCanBeFound(t *testing.T) {
	t.Parallel()

	result := replayResult(historicalShipment("2000"), simulated("2200"), nil)

	assert.Equal(t, "PRO-1001", result.ProNumber)
}
