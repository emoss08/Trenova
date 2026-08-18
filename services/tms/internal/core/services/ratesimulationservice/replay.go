package ratesimulationservice

import (
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/ratesimulation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	simmath "github.com/emoss08/trenova/pkg/ratesimulation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// replayResult turns one replayed shipment into the row a simulation stores.
//
// A shipment that could not be priced is recorded rather than skipped. It is
// the most important row in the report — revenue the proposed change would drop
// on the floor — and leaving it out would make the change look better than it is.
func replayResult(
	entity *shipment.Shipment,
	rated *services.RatedShipment,
	err error,
) *ratesimulation.RateSimulationResult {
	result := &ratesimulation.RateSimulationResult{
		ShipmentID:      entity.ID,
		ProNumber:       entity.ProNumber,
		CustomerID:      idOrNil(entity.CustomerID),
		EquipmentTypeID: idOrNil(entity.TractorTypeID),
		BeforeAmount:    billedAmount(entity),
	}

	if err != nil {
		result.Outcome = ratequote.OutcomeError
		result.Error = err.Error()

		return result
	}

	if rated == nil {
		result.Outcome = ratequote.OutcomeNoRateFound

		return result
	}

	result.Outcome = rated.Outcome
	result.AfterAmount = rated.Amount
	result.AfterRuleID = rated.RuleID
	result.LaneKey = winningLaneKey(rated)

	delta := simmath.Delta{Before: result.BeforeAmount, After: result.AfterAmount}
	result.Delta = delta.Amount()
	result.DeltaPercent = delta.Percent()

	return result
}

// billedAmount is what the shipment was actually invoiced, which is the only
// number a simulated rate is worth comparing against.
func billedAmount(entity *shipment.Shipment) decimal.Decimal {
	if !entity.TotalChargeAmount.Valid {
		return decimal.Zero
	}

	return entity.TotalChargeAmount.Decimal
}

// winningLaneKey reads the lane the simulated rate was found under, so the
// result grid can group by lane without joining back to a rule that may since
// have been amended.
func winningLaneKey(rated *services.RatedShipment) string {
	if rated.Quote == nil || rated.Quote.Trace == nil {
		return ""
	}

	winner := rated.Quote.Trace.Winner()
	if winner == nil {
		return ""
	}

	return winner.LaneKey
}

func idOrNil(id pulid.ID) *pulid.ID {
	if id.IsNil() {
		return nil
	}

	return &id
}
