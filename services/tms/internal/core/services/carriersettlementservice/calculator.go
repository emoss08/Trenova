package carriersettlementservice

import (
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

type CostEventInput struct {
	EventType      carriersettlement.CostEventType
	IdempotencyKey string
	AmountMinor    int64
	Description    string
}

func costEventIdempotencyKey(
	assignmentID pulid.ID,
	eventType carriersettlement.CostEventType,
	index int,
) string {
	return "carrier-cost:" + assignmentID.String() + ":" + eventType.String() + ":" +
		strconv.Itoa(index)
}

// BuildAssignmentCostEventInputs materializes the buy-rate lines on an active
// carrier assignment into cost-event inputs: one linehaul event for the base
// amount, one fuel-surcharge event when nonzero, and one accessorial event per
// accessorial line. Decimal amounts convert to minor units at this boundary.
func BuildAssignmentCostEventInputs(
	assignment *shipment.CarrierAssignment,
) []CostEventInput {
	if assignment == nil {
		return nil
	}

	inputs := make([]CostEventInput, 0, 2+len(assignment.Accessorials))
	inputs = append(inputs, CostEventInput{
		EventType: carriersettlement.CostEventTypeLinehaulCost,
		IdempotencyKey: costEventIdempotencyKey(
			assignment.ID,
			carriersettlement.CostEventTypeLinehaulCost,
			0,
		),
		AmountMinor: money.MinorUnits(assignment.BaseAmount),
		Description: "Linehaul (" + assignment.RateMethod.String() + ")",
	})

	if !assignment.FuelSurcharge.IsZero() {
		inputs = append(inputs, CostEventInput{
			EventType: carriersettlement.CostEventTypeFuelSurcharge,
			IdempotencyKey: costEventIdempotencyKey(
				assignment.ID,
				carriersettlement.CostEventTypeFuelSurcharge,
				0,
			),
			AmountMinor: money.MinorUnits(assignment.FuelSurcharge),
			Description: "Fuel surcharge",
		})
	}

	accessorialIdx := 0
	for _, acc := range assignment.Accessorials {
		if acc == nil {
			continue
		}
		inputs = append(inputs, CostEventInput{
			EventType: carriersettlement.CostEventTypeAccessorial,
			IdempotencyKey: costEventIdempotencyKey(
				assignment.ID,
				carriersettlement.CostEventTypeAccessorial,
				accessorialIdx,
			),
			AmountMinor: money.MinorUnits(acc.Amount),
			Description: acc.Description,
		})
		accessorialIdx++
	}

	return inputs
}
